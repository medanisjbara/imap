package main

import (
	"context"
	"sync"

	"mybridge/database"
	"mybridge/pkg/emailmeow"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix/bridge"
	"maunium.net/go/mautrix/bridge/bridgeconfig"
	"maunium.net/go/mautrix/bridge/status"
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

func (user *User) GetRemoteID() string {
	return user.EmailAddress
}

func (user *User) GetRemoteName() string {
	// FIXME
	return user.EmailAddress
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
	// TODO
	return nil
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

func (user *User) Connect() {
	log := user.log.With().Str("component", "messagix").Logger()
	cli := emailmeow.NewClient(user.EmailAddress, user.Password, log)
	cli.EventHandler = user.eventHandler
	user.Client = cli
	// TODO maybe add user.lastFullReconnect = time.Now() ?
}

func (user *User) eventHandler(rawEvt any) {
	// TODO: FIXME
}

func (br *MyBridge) GetAllLoggedInUsers() []*User {
	br.usersLock.Lock()
	defer br.usersLock.Unlock()

	dbUsers, err := br.DB.User.GetAllLoggedIn(context.TODO())
	if err != nil {
		br.ZLog.Err(err).Msg("Error getting all logged in users")
		return nil
	}
	users := make([]*User, len(dbUsers))

	for idx, dbUser := range dbUsers {
		user, ok := br.usersByMXID[dbUser.MXID]
		if !ok {
			user = br.loadUser(context.TODO(), dbUser, nil)
		}
		users[idx] = user
	}
	return users
}

func (br *MyBridge) StartUsers() {
	br.ZLog.Debug().Msg("Starting users")

	usersWithToken := br.GetAllLoggedInUsers()
	for _, u := range usersWithToken {
		go u.Connect()
	}
	if len(usersWithToken) == 0 {
		br.SendGlobalBridgeState(status.BridgeState{StateEvent: status.StateUnconfigured}.Fill(nil))
	}

	br.ZLog.Debug().Msg("Starting custom puppets")
	for _, customPuppet := range br.GetAllPuppetsWithCustomMXID() {
		go func(puppet *Puppet) {
			br.ZLog.Debug().Stringer("user_id", puppet.CustomMXID).Msg("Starting custom puppet")

			if err := puppet.StartCustomMXID(true); err != nil {
				puppet.log.Error().Err(err).Msg("Failed to start custom puppet")
			}
		}(customPuppet)
	}
}

func (br *MyBridge) loadUser(ctx context.Context, dbUser *database.User, mxid *id.UserID) *User {
	if dbUser == nil {
		if mxid == nil {
			return nil
		}
		dbUser = br.DB.User.New()
		dbUser.MXID = *mxid
		err := dbUser.Insert(ctx)
		if err != nil {
			br.ZLog.Err(err).Msg("Error creating user %s")
			return nil
		}
	}

	user := br.NewUser(dbUser)
	br.usersByMXID[user.MXID] = user
	if user.EmailAddress != "" {
		br.usersByEmailAddress[user.EmailAddress] = user
	}
	if user.ManagementRoom != "" {
		br.managementRoomsLock.Lock()
		br.managementRooms[user.ManagementRoom] = user
		br.managementRoomsLock.Unlock()
	}
	return user
}

func (br *MyBridge) NewUser(dbUser *database.User) *User {
	user := &User{
		User:   dbUser,
		bridge: br,
		log:    br.ZLog.With().Stringer("user_id", dbUser.MXID).Logger(),

		PermissionLevel: br.Config.Bridge.Permissions.Get(dbUser.MXID),
	}
	user.Admin = user.PermissionLevel >= bridgeconfig.PermissionLevelAdmin
	user.BridgeState = br.NewBridgeStateQueue(user)
	return user
}
