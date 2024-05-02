package main

import (
	"sync"

	"mybridge/database"
	"mybridge/pkg/emailmeow"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/id"
)

type User struct {
	*database.User

	sync.Mutex

	bridge *MyBridge
	log    zerolog.Logger

	Admin           bool
	PermissionLevel bridgeconfig.PermissionLevel

	Client *emailmeow.Client

	BridgeState *bridge.BridgeStateQueue

	spaceMembershipChecked bool
	spaceCreateLock        sync.Mutex
}

func (user *User) GetIDoublePuppet() bridge.DoublePuppet {
	p := user.bridge.GetPuppetByCustomMXID(user.MXID)
	if p == nil || p.CustomIntent() == nil {
		return nil
	}
	return p
}

func (user *User) GetIGhost() bridge.Ghost {
	p := user.bridge.GetPuppetByEmailAddress(user.EmailAddress)
	if p == nil {
		return nil
	}
	return p
}

func (user *User) GetMXID() id.UserID {
	return user.MXID
}

func (user *User) GetManagementRoomID() id.RoomID {
	return user.ManagementRoom
}

func (user *User) GetPermissionLevel() bridgeconfig.PermissionLevel {
	return user.PermissionLevel
}

func (user *User) IsLoggedIn() bool {
	user.Lock()
	defer user.Unlock()

	return user.Client != nil && user.Client.IsLoggedIn()
}

func (user *User) SetManagementRoom(roomID id.RoomID) {
	user.bridge.managementRoomsLock.Lock()
	defer user.bridge.managementRoomsLock.Unlock()

	existing, ok := user.bridge.managementRooms[roomID]
	if ok {
		existing.ManagementRoom = ""
		err := existing.Update(context.TODO())
		if err != nil {
			existing.log.Err(err).Msg("Failed to update user when removing management room")
		}
	}

	user.ManagementRoom = roomID
	user.bridge.managementRooms[user.ManagementRoom] = user
	err := user.Update(context.TODO())
	if err != nil {
		user.log.Error().Err(err).Msg("Error setting management room")
	}
}
