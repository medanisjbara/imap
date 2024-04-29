package main

import (
	"context"
	"github.com/medanisjbara/mautrix-imap/mail"
	"github.com/medanisjbara/mautrix-imap/mail/types"

	"github.com/rs/zerolog"

	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/id"

	"github.com/medanisjbara/mautrix-imap/database"
)

type User struct {
	*database.User
	bridge *IMAPBridge
	zlog   zerolog.Logger

	Admin bool

	EmailAddress types.JID
	Session      *mail.Client
	BridgeState  *bridge.BridgeStateQueue
}

func (user *User) SetManagementRoom(roomID id.RoomID) {
	user.bridge.managementRoomsLock.Lock()
	defer user.bridge.managementRoomsLock.Unlock()

	existing, ok := user.bridge.managementRooms[roomID]
	if ok {
		existing.ManagementRoom = ""
		existing.Update()
	}

	user.ManagementRoom = roomID
	user.bridge.managementRooms[user.ManagementRoom] = user
	user.Update()
}

func (user *User) IsLoggedIn() bool {
	user.Lock()
	defer user.Unlock()

	// TODO: implement IsLoggedIn
	return true
}

func (user *User) GetPermissionLevel() bridgeconfig.PermissionLevel {
	return user.PermissionLevel
}

func (user *User) GetManagementRoomID() id.RoomID {
	return user.ManagementRoom
}

func (user *User) GetMXID() id.UserID {
	return user.MXID
}

func (user *User) IsConnected() bool {
	return user.Session != nil && user.Session.IsConnected()
}

func (user *User) Login(ctx context.Context) error {
	user.connLock.Lock()
	defer user.connLock.Unlock()
	if user.Session != nil {
		return nil, ErrAlreadyLoggedIn
	}
	newSession.Log = waLog.Zerolog(user.zlog.With().Str("component", "imap session").Logger())
	user.createSession()
	err = user.Session.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to imap: %w", err)
	}
	return nil
}

func (br *IMAPBridge) GetUserByMXID(userID id.UserID) *User {
	return br.getUserByMXID(userID, false)
}

func (br *IMAPBridge) GetIUser(userID id.UserID, create bool) bridge.User {
	u := br.getUserByMXID(userID, !create)
	if u == nil {
		return nil
	}
	return u
}
