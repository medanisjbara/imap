package main

import (
	_ "embed"
	"fmt"
	"sync"

	"go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/commands"
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
	// Implement your user retrieval logic here
	fmt.Println("Get I User")
	return nil
}

func (br *MyBridge) IsGhost(mxid id.UserID) bool {
	// Implement your ghost checking logic here
	fmt.Println("Is Ghost")
	return false
}

func (br *MyBridge) GetIGhost(mxid id.UserID) bridge.Ghost {
	// Implement your ghost retrieval logic here
	fmt.Println("Is I Ghost")
	return nil
}

func (br *MyBridge) CreatePrivatePortal(roomID id.RoomID, brInviter bridge.User, brGhost bridge.Ghost) {
	// Implement your private portal creation logic here
	fmt.Println("Create Private Portal")
}

func main() {
	br := &MyBridge{
		// Initialize your bridge fields here
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
