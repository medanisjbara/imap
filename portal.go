package main

import (
	"context"
	"strconv"

	"github.com/rs/zerolog"

	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/event"

	"mybridge/database"
	"mybridge/msgconv"
	"mybridge/pkg/emailmeow/events"
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
		PortalMethods:        portal,
		MaxFileSize:          br.MediaConfig.UploadSize,
	}
	go portal.messageLoop()

	return portal
}
