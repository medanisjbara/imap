package main

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/commands"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"

	"mybridge/config"
	"mybridge/database"
)

//go:embed example-config.yaml
var ExampleConfig string

var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type MyBridge struct {
	bridge.Bridge

	Config *config.Config
	DB     *database.Database

	usersByMXID         map[id.UserID]*User
	usersByEmailAddress map[string]*User
	usersLock           sync.Mutex

	managementRooms     map[id.RoomID]*User
	managementRoomsLock sync.Mutex

	portalsByMXID map[id.RoomID]*Portal
	portalsByID   map[database.PortalKey]*Portal
	portalsLock   sync.Mutex

	puppets             map[string]*Puppet
	puppetsByCustomMXID map[id.UserID]*Puppet
	puppetsLock         sync.Mutex
}

var _ bridge.ChildOverride = (*MyBridge)(nil)

func (br *MyBridge) GetExampleConfig() string {
	return ExampleConfig
}

func (br *MyBridge) GetConfigPtr() interface{} {
	br.Config = &config.Config{
		BaseConfig: &br.Bridge.Config,
	}
	br.Config.BaseConfig.Bridge = &br.Config.Bridge
	return br.Config
}

func (br *MyBridge) Init() {
	br.DB = database.New(br.Bridge.DB)

	br.CommandProcessor = commands.NewProcessor(&br.Bridge)
	br.RegisterCommands()

	ss := br.Config.Bridge.Provisioning.SharedSecret
	if len(ss) > 0 && ss != "disable" {
		// TODO: br.provisioning = &ProvisioningAPI{bridge: br, log: br.ZLog.With().Str("component", "provisioning").Logger()}
	}
}

func (br *MyBridge) Start() {
	go br.StartUsers()
}

func (br *MyBridge) Stop() {
	// Stop your bridge here
	fmt.Println("Stop")
}

func (br *MyBridge) GetIPortal(mxid id.RoomID) bridge.Portal {
	// Implement your portal retrieval logic here
	fmt.Println("Get Portal")
	return nil
}

func (br *MyBridge) GetIUser(mxid id.UserID, create bool) bridge.User {
	p := br.GetUserByMXID(mxid)
	if p == nil {
		return nil
	}
	return p
}

func (br *MyBridge) IsGhost(mxid id.UserID) bool {
	_, isGhost := br.ParsePuppetMXID(mxid)
	return isGhost
}

func (br *MyBridge) GetIGhost(mxid id.UserID) bridge.Ghost {
	// Implement your ghost retrieval logic here
	fmt.Println("Is I Ghost")
	return nil
}

func (br *MyBridge) CreatePrivatePortal(roomID id.RoomID, brInviter bridge.User, brGhost bridge.Ghost) {
	inviter := brInviter.(*User)
	puppet := brGhost.(*Puppet)

	log := br.ZLog.With().
		Str("action", "create private portal").
		Stringer("target_room_id", roomID).
		Stringer("inviter_mxid", brInviter.GetMXID()).
		Str("inviter_email_address", puppet.EmailAddress).
		Logger()
	log.Debug().Msg("Creating private chat portal")

	key := database.NewPortalKey(puppet.EmailAddress, inviter.EmailAddress)
	portal := br.GetPortalByChatID(key)
	ctx := log.WithContext(context.TODO())

	if len(portal.MXID) == 0 {
		br.createPrivatePortalFromInvite(ctx, roomID, inviter, puppet, portal)
		return
	}
	log.Debug().
		Stringer("existing_room_id", portal.MXID).
		Msg("Existing private chat portal found, trying to invite user")

	ok := portal.ensureUserInvited(ctx, inviter)
	if !ok {
		log.Warn().Msg("Failed to invite user to existing private chat portal. Redirecting portal to new room")
		br.createPrivatePortalFromInvite(ctx, roomID, inviter, puppet, portal)
		return
	}
	intent := puppet.DefaultIntent()
	errorMessage := fmt.Sprintf("You already have a private chat portal with me at [%[1]s](https://matrix.to/#/%[1]s)", portal.MXID)
	errorContent := format.RenderMarkdown(errorMessage, true, false)
	_, _ = intent.SendMessageEvent(ctx, roomID, event.EventMessage, errorContent)
	log.Debug().Msg("Leaving ghost from private chat room after accepting invite because we already have a chat with the user")
	_, _ = intent.LeaveRoom(ctx, roomID)
}

func (br *MyBridge) createPrivatePortalFromInvite(ctx context.Context, roomID id.RoomID, inviter *User, puppet *Puppet, portal *Portal) {
	log := zerolog.Ctx(ctx)
	log.Debug().Msg("Creating private portal from invite")

	// Check if room is already encrypted
	var existingEncryption event.EncryptionEventContent
	var encryptionEnabled bool
	err := portal.MainIntent().StateEvent(ctx, roomID, event.StateEncryption, "", &existingEncryption)
	if err != nil {
		log.Err(err).Msg("Failed to check if encryption is enabled in private chat room")
	} else {
		encryptionEnabled = existingEncryption.Algorithm == id.AlgorithmMegolmV1
	}
	portal.MXID = roomID
	br.portalsLock.Lock()
	br.portalsByMXID[portal.MXID] = portal
	br.portalsLock.Unlock()
	intent := puppet.DefaultIntent()

	if br.Config.Bridge.Encryption.Default || encryptionEnabled {
		log.Debug().Msg("Adding bridge bot to new private chat portal as encryption is enabled")
		_, err = intent.InviteUser(ctx, roomID, &mautrix.ReqInviteUser{UserID: br.Bot.UserID})
		if err != nil {
			log.Err(err).Msg("Failed to invite bridge bot to enable e2be")
		}
		err = br.Bot.EnsureJoined(ctx, roomID)
		if err != nil {
			log.Err(err).Msg("Failed to join as bridge bot to enable e2be")
		}
		if !encryptionEnabled {
			_, err = intent.SendStateEvent(ctx, roomID, event.StateEncryption, "", portal.getEncryptionEventContent())
			if err != nil {
				log.Err(err).Msg("Failed to enable e2be")
			}
		}
		br.AS.StateStore.SetMembership(ctx, roomID, inviter.MXID, event.MembershipJoin)
		br.AS.StateStore.SetMembership(ctx, roomID, puppet.MXID, event.MembershipJoin)
		br.AS.StateStore.SetMembership(ctx, roomID, br.Bot.UserID, event.MembershipJoin)
		portal.Encrypted = true
	}
	portal.UpdateDMInfo(ctx, true)
	_, _ = intent.SendNotice(ctx, roomID, "Private chat portal created")
	log.Info().Msg("Created private chat portal after invite")
}

func main() {
	br := &MyBridge{
		usersByMXID:         make(map[id.UserID]*User),
		usersByEmailAddress: make(map[string]*User),

		managementRooms: make(map[id.RoomID]*User),

		portalsByMXID: make(map[id.RoomID]*Portal),
		portalsByID:   make(map[database.PortalKey]*Portal),
	}
	br.Bridge = bridge.Bridge{
		Name:        "mybridge",
		Description: "Your bridge description here.",
		Version:     "0.1.0",

		ConfigUpgrader: &configupgrade.StructUpgrader{
			SimpleUpgrader: configupgrade.SimpleUpgrader(config.DoUpgrade),
			Blocks:         config.SpacedBlocks,
			Base:           ExampleConfig,
		},

		Child: br,
	}
	br.InitVersion(Tag, Commit, BuildTime)

	br.Main()
}
