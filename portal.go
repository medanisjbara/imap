package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"mybridge/database"
	"mybridge/msgconv"
	"mybridge/pkg/emailmeow/events"
)

type msgconvContextKey int

const (
	msgconvContextKeyIntent msgconvContextKey = iota
	msgconvContextKeyClient
)

type portalEmailMessage struct {
	evt  *events.ChatEvent
	user *User
}

type portalMatrixMessage struct {
	evt  *event.Event
	user *User
}

// Portal implementation
type Portal struct {
	*database.Portal

	MsgConv *msgconv.MessageConverter

	bridge *MyBridge
	log    zerolog.Logger

	emailMessages  chan portalEmailMessage
	matrixMessages chan portalMatrixMessage
}

func (portal *Portal) IsEncrypted() bool {
	return portal.Encrypted
}

func (portal *Portal) IsPrivateChat() bool {
	// FIXME
	return false
}

func (portal *Portal) MainIntent() *appservice.IntentAPI {
	dmPuppet := portal.GetDMPuppet()
	if dmPuppet != nil {
		return dmPuppet.DefaultIntent()
	}

	return portal.bridge.Bot
}

func (portal *Portal) GetDMPuppet() *Puppet {
	if portal.EmailAddress == "" {
		return nil
	}
	return portal.bridge.GetPuppetByEmailAddress(portal.EmailAddress)
}

func (portal *Portal) MarkEncrypted() {
	portal.Encrypted = true
	err := portal.Update(context.TODO())
	if err != nil {
		portal.log.Err(err).Msg("Failed to update portal in database after marking as encrypted")
	}
}

func (portal *Portal) ReceiveMatrixEvent(user bridge.User, evt *event.Event) {
	if user.GetPermissionLevel() >= bridgeconfig.PermissionLevelUser || portal.HasRelaybot() {
		portal.matrixMessages <- portalMatrixMessage{user: user.(*User), evt: evt}
	}
}

func (br *MyBridge) GetAllIPortals() (iportals []bridge.Portal) {
	portals, err := br.dbPortalsToPortals(br.DB.Portal.GetAllWithMXID(context.TODO()))
	if err != nil {
		br.ZLog.Err(err).Msg("Failed to get all portals with mxid")
		return nil
	}
	iportals = make([]bridge.Portal, len(portals))
	for i, portal := range portals {
		iportals[i] = portal
	}
	return iportals
}

func (portal *Portal) UpdateBridgeInfo(ctx context.Context) {
	if len(portal.MXID) == 0 {
		portal.log.Debug().Msg("Not updating bridge info: no Matrix room created")
		return
	}
	portal.log.Debug().Msg("Updating bridge info...")
	stateKey, content := portal.getBridgeInfo()
	_, err := portal.MainIntent().SendStateEvent(ctx, portal.MXID, event.StateBridge, stateKey, content)
	if err != nil {
		portal.log.Warn().Err(err).Msg("Failed to update m.bridge")
	}
	// TODO remove this once https://github.com/matrix-org/matrix-doc/pull/2346 is in spec
	_, err = portal.MainIntent().SendStateEvent(ctx, portal.MXID, event.StateHalfShotBridge, stateKey, content)
	if err != nil {
		portal.log.Warn().Err(err).Msg("Failed to update uk.half-shot.bridge")
	}
}

func (portal *Portal) HasRelaybot() bool {
	return portal.bridge.Config.Bridge.Relay.Enabled && len(portal.RelayUserID) > 0
}

func (portal *Portal) getBridgeInfo() (string, string) {
	return "", ""
}

func (portal *Portal) messageLoop() {
	for {
		select {
		case msg := <-portal.matrixMessages:
			portal.handleMatrixMessages(msg)
		case msg := <-portal.emailMessages:
			portal.handleEmailMessage(msg)
		}
	}
}

func (portal *Portal) handleMatrixMessages(msg portalMatrixMessage) {
	log := portal.log.With().
		Str("action", "handle matrix event").
		Stringer("event_id", msg.evt.ID).
		Str("event_type", msg.evt.Type.String()).
		Logger()
	ctx := log.WithContext(context.TODO())

	switch msg.evt.Type {
	case event.EventMessage, event.EventSticker:
		portal.handleMatrixMessage(ctx, msg.user, msg.evt)
	default:
		log.Warn().Str("type", msg.evt.Type.Type).Msg("Unhandled matrix message type")
	}
}

func (portal *Portal) handleMatrixMessage(ctx context.Context, sender *User, evt *event.Event) {
	log := zerolog.Ctx(ctx)
	evtTS := time.UnixMilli(evt.Timestamp)
	timings := messageTimings{
		initReceive:  evt.Mautrix.ReceivedAt.Sub(evtTS),
		decrypt:      evt.Mautrix.DecryptionDuration,
		totalReceive: time.Since(evtTS),
	}
	implicitRRStart := time.Now()
	timings.implicitRR = time.Since(implicitRRStart)
	start := time.Now()

	messageAge := timings.totalReceive
	ms := metricSender{portal: portal, timings: &timings, ctx: ctx}
	log.Debug().
		Stringer("sender", evt.Sender).
		Dur("age", messageAge).
		Msg("Received message")

	errorAfter := portal.bridge.Config.Bridge.MessageHandlingTimeout.ErrorAfter
	deadline := portal.bridge.Config.Bridge.MessageHandlingTimeout.Deadline
	isScheduled, _ := evt.Content.Raw["com.beeper.scheduled"].(bool)
	if isScheduled {
		log.Debug().Msg("Message is a scheduled message, extending handling timeouts")
		errorAfter *= 10
		deadline *= 10
	}

	if errorAfter > 0 {
		remainingTime := errorAfter - messageAge
		if remainingTime < 0 {
			go ms.sendMessageMetrics(evt, errTimeoutBeforeHandling, "Timeout handling", true)
			return
		} else if remainingTime < 1*time.Second {
			log.Warn().
				Dur("remaining_time", remainingTime).
				Dur("max_timeout", errorAfter).
				Msg("Message was delayed before reaching the bridge")
		}
		go func() {
			time.Sleep(remainingTime)
			ms.sendMessageMetrics(evt, errMessageTakingLong, "Timeout handling", false)
		}()
	}

	if deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, deadline)
		defer cancel()
	}

	timings.preproc = time.Since(start)
	start = time.Now()

	content, ok := evt.Content.Parsed.(*event.MessageEventContent)
	if !ok {
		log.Error().Type("content_type", content).Msg("Unexpected parsed content type")
		go ms.sendMessageMetrics(evt, fmt.Errorf("%w %T", errUnexpectedParsedContentType, evt.Content.Parsed), "Error converting", true)
		return
	}

	realSenderMXID := sender.MXID
	isRelay := false
	if !sender.IsLoggedIn() {
		sender = portal.GetRelayUser()
		if sender == nil {
			go ms.sendMessageMetrics(evt, errUserNotLoggedIn, "Ignoring", true)
			return
		} else if !sender.IsLoggedIn() {
			go ms.sendMessageMetrics(evt, errRelaybotNotLoggedIn, "Ignoring", true)
			return
		}
		isRelay = true
	}

	var editTargetMsg *database.Message
	if editTarget := content.RelatesTo.GetReplaceID(); editTarget != "" {
		var err error
		editTargetMsg, err = portal.bridge.DB.Message.GetByMXID(ctx, editTarget)
		if err != nil {
			log.Err(err).Stringer("edit_target_mxid", editTarget).Msg("Failed to get edit target message")
			go ms.sendMessageMetrics(evt, errFailedToGetEditTarget, "Error converting", true)
			return
		} else if editTargetMsg == nil {
			log.Err(err).Stringer("edit_target_mxid", editTarget).Msg("Edit target message not found")
			go ms.sendMessageMetrics(evt, errEditUnknownTarget, "Error converting", true)
			return
		} else if editTargetMsg.Sender != sender.EmailAddress {
			go ms.sendMessageMetrics(evt, errEditDifferentSender, "Error converting", true)
			return
		}
		if content.NewContent != nil {
			content = content.NewContent
			evt.Content.Parsed = content
		}
	}

	// relaybotFormatted := isRelay && portal.addRelaybotFormat(ctx, realSenderMXID, evt, content)
	if content.MsgType == event.MsgNotice && !portal.bridge.Config.Bridge.BridgeNotices {
		go ms.sendMessageMetrics(evt, errMNoticeDisabled, "Error converting", true)
		return
	}
	ctx = context.WithValue(ctx, msgconvContextKeyClient, sender.Client)
	msg, err := portal.MsgConv.ToEmail(ctx, evt, content)
	if err != nil {
		log.Err(err).Msg("Failed to convert message")
		go ms.sendMessageMetrics(evt, err, "Error converting", true)
		return
	}

	timings.convert = time.Since(start)
	start = time.Now()

	err = portal.sendEmailMessage(ctx, msg, sender, evt.ID)

	timings.totalSend = time.Since(start)
	go ms.sendMessageMetrics(evt, err, "Error sending", true)
	if err == nil {
		if editTargetMsg != nil {
			err = editTargetMsg.SetTimestamp(ctx, msg.GetTimestamp())
			if err != nil {
				log.Err(err).Msg("Failed to update message timestamp in database after editing")
			}
		} else {
			portal.storeMessageInDB(ctx, evt.ID, sender.EmailAddress, msg.GetTimestamp(), 0)
			if portal.ExpirationTime > 0 {
				portal.addDisappearingMessage(ctx, evt.ID, uint32(portal.ExpirationTime), true)
			}
		}
	}
}

// FIXME: delete this
type DataMessage string

func (portal *Portal) handleEmailMessage(portalMessage portalEmailMessage) {
	sender := portal.bridge.GetPuppetByEmailAddress(portalMessage.evt.Info.Sender)
	if sender == nil {
		portal.log.Warn().
			Stringer("sender_id", portalMessage.evt.Info.Sender).
			Msg("Couldn't get puppet for message")
		return
	}
	var msgType string
	var timestamp uint64
	switch typedEvt := portalMessage.evt.Event.(type) {
	// FIXME
	case *DataMessage:
		msgType = "data"
		timestamp = typedEvt.GetTimestamp()
		portal.handleEmailDataMessage(portalMessage.user, sender, typedEvt)
	default:
		portal.log.Error().
			Type("data_type", typedEvt).
			Msg("Invalid inner event type inside ChatEvent")
	}
	portal.bridge.Metrics.TrackEmailMessage(time.UnixMilli(int64(timestamp)), msgType)
}

func (portal *Portal) sendMainIntentMessage(ctx context.Context, content *event.MessageEventContent) (*mautrix.RespSendEvent, error) {
	return portal.sendMatrixEvent(ctx, portal.MainIntent(), event.EventMessage, content, nil, 0)
}

func (portal *Portal) getBridgeInfoStateKey() string {
	return fmt.Sprintf("net.maunium.mybridge://bridge/%s", portal.ThreadID)
}

func (portal *Portal) GetRelayUser() *User {
	if !portal.HasRelaybot() {
		return nil
	} else if portal.relayUser == nil {
		portal.relayUser = portal.bridge.GetUserByMXID(portal.RelayUserID)
	}
	return portal.relayUser
}

func (portal *Portal) addRelaybotFormat(ctx context.Context, userID id.UserID, evt *event.Event, content *event.MessageEventContent) bool {
	member := portal.MainIntent().Member(ctx, portal.MXID, userID)
	if member == nil {
		member = &event.MemberEventContent{}
	}
	// Stickers can't have captions, so force them into images when relaying
	if evt.Type == event.EventSticker {
		content.MsgType = event.MsgImage
		evt.Type = event.EventMessage
	}
	content.EnsureHasHTML()
	data, err := portal.bridge.Config.Bridge.Relay.FormatMessage(content, userID, *member)
	if err != nil {
		portal.log.Err(err).Msg("Failed to apply relaybot format")
	}
	content.FormattedBody = data
	// Force FileName field so the formatted body is used as a caption
	if content.FileName == "" {
		content.FileName = content.Body
	}
	return true
}

// Bridge stuff related to Portals
func (br *MyBridge) dbPortalsToPortals(dbPortals []*database.Portal, err error) ([]*Portal, error) {
	if err != nil {
		return nil, err
	}
	br.portalsLock.Lock()
	defer br.portalsLock.Unlock()

	output := make([]*Portal, len(dbPortals))
	for index, dbPortal := range dbPortals {
		if dbPortal == nil {
			continue
		}

		portal, ok := br.portalsByID[dbPortal.PortalKey]
		if !ok {
			portal = br.loadPortal(context.TODO(), dbPortal, nil)
		}

		output[index] = portal
	}

	return output, nil
}

func (br *MyBridge) loadPortal(ctx context.Context, dbPortal *database.Portal, key *database.PortalKey) *Portal {
	if dbPortal == nil {
		if key == nil {
			return nil
		}

		dbPortal = br.DB.Portal.New()
		dbPortal.PortalKey = *key
		err := dbPortal.Insert(ctx)
		if err != nil {
			br.ZLog.Err(err).Msg("Failed to insert new portal")
			return nil
		}
	}

	portal := br.NewPortal(dbPortal)

	br.portalsByID[portal.PortalKey] = portal
	if portal.MXID != "" {
		br.portalsByMXID[portal.MXID] = portal
	}

	return portal
}

func (br *MyBridge) NewPortal(dbPortal *database.Portal) *Portal {

	threadIDStr := strconv.FormatInt(dbPortal.ThreadID, 10)

	log := br.ZLog.With().Str("thread_id", threadIDStr).Logger()

	if dbPortal.MXID != "" {
		log = log.With().Stringer("room_id", dbPortal.MXID).Logger()
	}

	portal := &Portal{
		Portal: dbPortal,
		bridge: br,
		log:    log,

		emailMessages:  make(chan portalEmailMessage, br.Config.Bridge.PortalMessageBuffer),
		matrixMessages: make(chan portalMatrixMessage, br.Config.Bridge.PortalMessageBuffer),
	}
	portal.MsgConv = &msgconv.MessageConverter{
		PortalMethods: portal,
		MaxFileSize:   br.MediaConfig.UploadSize,
	}
	go portal.messageLoop()

	return portal
}
