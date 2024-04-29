// mautrix-imap - A Matrix-Email puppeting bridge.
// Copyright (C) 2022 Tulir Asokan
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
	_ "embed"
	"sync"

	"go.mau.fi/util/configupgrade"

	"github.com/medanisjbara/mautrix-imap/mail/types"

	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/commands"
	"maunium.net/go/mautrix/id"

	"github.com/medanisjbara/mautrix-imap/config"
	"github.com/medanisjbara/mautrix-imap/database"
)

// Information to find out exactly which commit the bridge was built from.
// These are filled at build time with the -X linker flag.
var (
	Tag       = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

//go:embed example-config.yaml
var ExampleConfig string

type IMAPBridge struct {
	bridge.Bridge
	Config       *config.Config
	DB           *database.Database
	Provisioning *ProvisioningAPI

	usersByMXID    map[id.UserID]*User
	usersByAddress map[string]*User
	usersLock      sync.Mutex

	managementRooms     map[id.RoomID]*User
	managementRoomsLock sync.Mutex

	portalsByMXID map[id.RoomID]*Portal
	portalsByJID  map[database.PortalKey]*Portal
	portalsLock   sync.Mutex

	puppets             map[types.JID]*Puppet
	puppetsByCustomMXID map[id.UserID]*Puppet
	puppetsLock         sync.Mutex
}

func (br *IMAPBridge) Init() {
	br.CommandProcessor = commands.NewProcessor(&br.Bridge)
	br.RegisterCommands()

	br.DB = database.New(br.Bridge.DB)
}

func (br *IMAPBridge) Start() {
	if br.Provisioning != nil {
		br.Provisioning.Init()
	}
	br.WaitWebsocketConnected()
	// go br.StartUsers()

	// NOTE: We might need to uncomment this
	// go br.Loop()
}

func (br *IMAPBridge) Stop() {
	// br.Metrics.Stop()
	for _, user := range br.usersByAddress {
		if user.Session == nil {
			continue
		}
		user.zlog.Debug().Msg("Disconnecting user")
		user.Session.Disconnect()
		// close(user.historySyncs)
	}
}

func (br *IMAPBridge) GetExampleConfig() string {
	return ExampleConfig
}

func (br *IMAPBridge) GetConfigPtr() interface{} {
	br.Config = &config.Config{
		BaseConfig: &br.Bridge.Config,
	}
	br.Config.BaseConfig.Bridge = &br.Config.Bridge
	return br.Config
}

func main() {
	br := &IMAPBridge{
		usersByMXID:    make(map[id.UserID]*User),
		usersByAddress: make(map[string]*User),

		managementRooms: make(map[id.RoomID]*User),

		portalsByMXID: make(map[id.RoomID]*Portal),
		portalsByJID:  make(map[database.PortalKey]*Portal),

		puppets:             make(map[types.JID]*Puppet),
		puppetsByCustomMXID: make(map[id.UserID]*Puppet),
	}
	br.Bridge = bridge.Bridge{
		Name:              "mautrix-imap",
		URL:               "https://github.com/medanisjbara/mautrix-imap",
		Description:       "A Matrix-WhatsApp puppeting bridge.",
		Version:           "0.0.1",
		ProtocolName:      "IMAP",
		BeeperServiceName: "imap",
		BeeperNetworkName: "imap",

		// TODO check if this is to be edited
		CryptoPickleKey: "maunium.net/go/mautrix-imap",

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
