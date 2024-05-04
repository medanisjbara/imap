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

func (br *MyBridge) GetUserByMXID(userID id.UserID) *User {
	return br.maybeGetUserByMXID(userID, &userID)
}

func (br *MyBridge) GetUserByMXIDIfExists(userID id.UserID) *User {
	return br.maybeGetUserByMXID(userID, nil)
}

func (br *MyBridge) maybeGetUserByMXID(userID id.UserID, userIDPtr *id.UserID) *User {
	if userID == br.Bot.UserID || br.IsGhost(userID) {
		return nil
	}
	br.usersLock.Lock()
	defer br.usersLock.Unlock()

	user, ok := br.usersByMXID[userID]
	if !ok {
		dbUser, err := br.DB.User.GetByMXID(context.TODO(), userID)
		if err != nil {
			br.ZLog.Err(err).Msg("Failed to get user from database")
			return nil
		}
		return br.loadUser(context.TODO(), dbUser, userIDPtr)
	}
	return user
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
