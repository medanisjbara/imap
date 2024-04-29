// mautrix-imap - A Matrix-Email puppeting bridge.
// Copyright (C) 2024 Tulir Asokan
// Copyright (C) 2024 Med Anis Jbara
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"maps"
	"math"
	"mime"
	"net/http"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/medanisjbara/mautrix-imap/mail/types"

	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"go.mau.fi/util/exzerolog"
	cwebp "go.mau.fi/webp"
	"golang.org/x/exp/slices"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/exmime"
	"go.mau.fi/util/ffmpeg"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/random"
	"go.mau.fi/util/variationselector"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/bridge/status"
	"maunium.net/go/mautrix/crypto/attachment"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"

	"github.com/medanisjbara/mautrix-imap/database"
)

func (br *IMAPBridge) GetPortalByMXID(mxid id.RoomID) *Portal {
	ctx := context.TODO()
	br.portalsLock.Lock()
	defer br.portalsLock.Unlock()
	portal, ok := br.portalsByMXID[mxid]
	if !ok {
		dbPortal, err := br.DB.Portal.GetByMXID(ctx, mxid)
		if err != nil {
			br.ZLog.Err(err).Stringer("mxid", mxid).Msg("Failed to get portal by MXID")
			return nil
		}
		return br.loadDBPortal(ctx, dbPortal, nil)
	}
	return portal
}

func (br *IMAPBridge) GetIPortal(mxid id.RoomID) bridge.Portal {
	p := br.GetPortalByMXID(mxid)
	if p == nil {
		return nil
	}
	return p
}

func (br *IMAPBridge) GetPortalByJID(key database.PortalKey) *Portal {
	ctx := context.TODO()
	br.portalsLock.Lock()
	defer br.portalsLock.Unlock()
	portal, ok := br.portalsByJID[key]
	if !ok {
		dbPortal, err := br.DB.Portal.GetByJID(ctx, key)
		if err != nil {
			br.ZLog.Err(err).Str("key", key.String()).Msg("Failed to get portal by JID")
			return nil
		}
		return br.loadDBPortal(ctx, dbPortal, &key)
	}
	return portal
}

func (br *IMAPBridge) GetExistingPortalByJID(key database.PortalKey) *Portal {
	ctx := context.TODO()
	br.portalsLock.Lock()
	defer br.portalsLock.Unlock()
	portal, ok := br.portalsByJID[key]
	if !ok {
		dbPortal, err := br.DB.Portal.GetByJID(ctx, key)
		if err != nil {
			br.ZLog.Err(err).Str("key", key.String()).Msg("Failed to get portal by JID")
			return nil
		}
		return br.loadDBPortal(ctx, dbPortal, nil)
	}
	return portal
}

func (br *IMAPBridge) GetAllPortals() []*Portal {
	return br.dbPortalsToPortals(br.DB.Portal.GetAll(context.TODO()))
}

func (br *IMAPBridge) GetAllIPortals() (iportals []bridge.Portal) {
	portals := br.GetAllPortals()
	iportals = make([]bridge.Portal, len(portals))
	for i, portal := range portals {
		iportals[i] = portal
	}
	return iportals
}

func (br *IMAPBridge) GetAllPortalsByJID(jid types.JID) []*Portal {
	return br.dbPortalsToPortals(br.DB.Portal.GetAllByJID(context.TODO(), jid))
}

func (br *IMAPBridge) GetAllByParentGroup(jid types.JID) []*Portal {
	return br.dbPortalsToPortals(br.DB.Portal.GetAllByParentGroup(context.TODO(), jid))
}

func (br *IMAPBridge) dbPortalsToPortals(dbPortals []*database.Portal, err error) []*Portal {
	if err != nil {
		br.ZLog.Err(err).Msg("Failed to get portals")
		return nil
	}
	br.portalsLock.Lock()
	defer br.portalsLock.Unlock()
	output := make([]*Portal, len(dbPortals))
	for index, dbPortal := range dbPortals {
		if dbPortal == nil {
			continue
		}
		portal, ok := br.portalsByJID[dbPortal.Key]
		if !ok {
			portal = br.loadDBPortal(context.TODO(), dbPortal, nil)
		}
		output[index] = portal
	}
	return output
}

func (br *IMAPBridge) loadDBPortal(ctx context.Context, dbPortal *database.Portal, key *database.PortalKey) *Portal {
	if dbPortal == nil {
		if key == nil {
			return nil
		}
		dbPortal = br.DB.Portal.New()
		dbPortal.Key = *key
		err := dbPortal.Insert(ctx)
		if err != nil {
			br.ZLog.Err(err).Str("key", key.String()).Msg("Failed to insert new portal")
			return nil
		}
	}
	portal := br.NewPortal(dbPortal)
	br.portalsByJID[portal.Key] = portal
	if len(portal.MXID) > 0 {
		br.portalsByMXID[portal.MXID] = portal
	}
	return portal
}

func (br *IMAPBridge) NewManualPortal(key database.PortalKey) *Portal {
	dbPortal := br.DB.Portal.New()
	dbPortal.Key = key
	return br.NewPortal(dbPortal)
}

func (br *IMAPBridge) NewPortal(dbPortal *database.Portal) *Portal {
	portal := &Portal{
		Portal: dbPortal,
		bridge: br,
	}
	// portal.updateLogger()
	go portal.handleMessageLoop()
	return portal
}

type portalEmailMessage struct {
	msg  interface{}
	user *User
}

type portalMatrixMessage struct {
	evt  *event.Event
	user *User
}

type recentlyHandledWrapper struct {
	id  types.MessageID
	err database.MessageErrorType
}

type Portal struct {
	*database.Portal

	bridge *IMAPBridge
	zlog   zerolog.Logger

	roomCreateLock sync.Mutex
	encryptLock    sync.Mutex
	backfillLock   sync.Mutex
	avatarLock     sync.Mutex

	latestEventBackfillLock sync.Mutex
	parentGroupUpdateLock   sync.Mutex

	currentlyTyping     []id.UserID
	currentlyTypingLock sync.Mutex

	galleryCache          []*event.MessageEventContent
	galleryCacheRootEvent id.EventID
	galleryCacheStart     time.Time
	galleryCacheSender    types.JID

	currentlySleepingToDelete sync.Map

	relayUser    *User
	parentPortal *Portal

	events string
}

const GalleryMaxTime = 10 * time.Minute

var (
	_ bridge.Portal                   = (*Portal)(nil)
	_ bridge.MembershipHandlingPortal = (*Portal)(nil)
	_ bridge.MetaHandlingPortal       = (*Portal)(nil)
	_ bridge.TypingPortal             = (*Portal)(nil)
)

type PortalMessage struct {
	source  *User
	content string
}

func pluralUnit(val int, name string) string {
	if val == 1 {
		return fmt.Sprintf("%d %s", val, name)
	} else if val == 0 {
		return ""
	}
	return fmt.Sprintf("%d %ss", val, name)
}

func naturalJoin(parts []string) string {
	if len(parts) == 0 {
		return ""
	} else if len(parts) == 1 {
		return parts[0]
	} else if len(parts) == 2 {
		return fmt.Sprintf("%s and %s", parts[0], parts[1])
	} else {
		return fmt.Sprintf("%s and %s", strings.Join(parts[:len(parts)-1], ", "), parts[len(parts)-1])
	}
}

func formatDuration(d time.Duration) string {
	const Day = time.Hour * 24

	var days, hours, minutes, seconds int
	days, d = int(d/Day), d%Day
	hours, d = int(d/time.Hour), d%time.Hour
	minutes, d = int(d/time.Minute), d%time.Minute
	seconds = int(d / time.Second)

	parts := make([]string, 0, 4)
	if days > 0 {
		parts = append(parts, pluralUnit(days, "day"))
	}
	if hours > 0 {
		parts = append(parts, pluralUnit(hours, "hour"))
	}
	if minutes > 0 {
		parts = append(parts, pluralUnit(seconds, "minute"))
	}
	if seconds > 0 {
		parts = append(parts, pluralUnit(seconds, "second"))
	}
	return naturalJoin(parts)
}

const UndecryptableMessageNotice = "Decrypting message from WhatsApp failed, waiting for sender to re-send... " +
	"([learn more](https://faq.whatsapp.com/general/security-and-privacy/seeing-waiting-for-this-message-this-may-take-a-while))"

var undecryptableMessageContent event.MessageEventContent

func init() {
	undecryptableMessageContent = format.RenderMarkdown(UndecryptableMessageNotice, true, false)
	undecryptableMessageContent.MsgType = event.MsgNotice
}

type Replyable interface {
	GetStanzaId() string
	GetParticipant() string
	GetRemoteJid() string
}

type ConvertedMessage struct {
	Intent  *appservice.IntentAPI
	Type    event.Type
	Content *event.MessageEventContent
	Extra   map[string]interface{}
	Caption *event.MessageEventContent

	MultiEvent []*event.MessageEventContent

	ExpiresIn time.Duration
	Error     database.MessageErrorType
	MediaKey  []byte
}

func (cm *ConvertedMessage) MergeCaption() {
	if cm.Caption == nil {
		return
	}
	cm.Content.FileName = cm.Content.Body
	extensibleCaption := map[string]interface{}{
		"org.matrix.msc1767.text": cm.Caption.Body,
	}
	cm.Extra["org.matrix.msc1767.caption"] = extensibleCaption
	cm.Content.Body = cm.Caption.Body
	if cm.Caption.Format == event.FormatHTML {
		cm.Content.Format = event.FormatHTML
		cm.Content.FormattedBody = cm.Caption.FormattedBody
		extensibleCaption["org.matrix.msc1767.html"] = cm.Caption.FormattedBody
	}
	cm.Caption = nil
}

func (portal *Portal) IsEncrypted() bool {
	return portal.Encrypted
}

func (portal *Portal) IsPrivateChat() bool {
	// TODO
	return true
}

func (portal *Portal) MainIntent() *appservice.IntentAPI {
	if portal.IsPrivateChat() && portal.OtherUserID != "" {
		return portal.bridge.GetPuppetByID(portal.OtherUserID).DefaultIntent()
	}

	return portal.bridge.Bot
}

func (portal *Portal) MarkEncrypted() {
	portal.Encrypted = true
	portal.Update()
}

func (portal *Portal) ReceiveMatrixEvent(user bridge.User, evt *event.Event) {
	if user.GetPermissionLevel() >= bridgeconfig.PermissionLevelUser || portal.RelayWebhookID != "" {
		portal.matrixMessages <- portalMatrixMessage{user: user.(*User), evt: evt}
	}
}

func (portal *Portal) UpdateBridgeInfo(ctx context.Context) {
	if len(portal.MXID) == 0 {
		portal.log.Debug().Msg("Not updating bridge info: no Matrix room created")
		return
	}
	portal.log.Debug().Msg("Updating bridge info...")
	stateKey, content := portal.getBridgeInfo()
	_, err := portal.MainIntent().SendStateEvent(portal.MXID, event.StateBridge, stateKey, content)
	if err != nil {
		portal.log.Warn().Err(err).Msg("Failed to update m.bridge")
	}
	// TODO remove this once https://github.com/matrix-org/matrix-doc/pull/2346 is in spec
	_, err = portal.MainIntent().SendStateEvent(portal.MXID, event.StateHalfShotBridge, stateKey, content)
	if err != nil {
		portal.log.Warn().Err(err).Msg("Failed to update uk.half-shot.bridge")
	}
}

func (portal *Portal) HandleMatrixKick(brSender bridge.User, brTarget bridge.Ghost, evt *event.Event) {
}
func (portal *Portal) HandleMatrixInvite(brSender bridge.User, brTarget bridge.Ghost, evt *event.Event) {
}
func (portal *Portal) HandleMatrixLeave(brSender bridge.User, evt *event.Event) {}
func (portal *Portal) HandleMatrixMeta(brSender bridge.User, evt *event.Event)  {}
func (portal *Portal) HandleMatrixTyping(typers []id.UserID)                    {}

func (portal *Portal) handleMessageLoop() {
	for {
		select {
		case msg := <-portal.matrixMessages:
			portal.handleMatrixMessages(msg)
		case msg := <-portal.emailMessages:
			portal.handleEmaildMessages(msg)
		}
	}
}

func (portal *Portal) handleMatrixMessages(msg portalMatrixMessage) {}
func (portal *Portal) handleEmaildMessages(msg portalEmailMessage)  {}
