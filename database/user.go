package database

import (
    "sync"
    "database/sql"

    "go.mau.fi/util/dbutil"
    "maunium.net/go/mautrix/id"
)

type UserQuery struct {
    *dbutil.QueryHelper[*User]
}

type User struct {
    qh *dbutil.QueryHelper[*User]

    MXID           id.UserID
    EmailAddress   string
    ManagementRoom id.RoomID
    SpaceRoom      id.RoomID

    lastReadCache     map[PortalKey]uint64
    lastReadCacheLock sync.Mutex
    inSpaceCache      map[PortalKey]bool
    inSpaceCacheLock  sync.Mutex
}

func (u *User) Scan(row dbutil.Scannable) (*User, error) {
	var emailAddress, managementRoom, spaceRoom sql.NullString
	err := row.Scan(
		&u.MXID,
		&emailAddress,
		&managementRoom,
		&spaceRoom,
	)
	if err != nil {
		return nil, err
	}
	u.EmailAddress = emailAddress.String
	u.ManagementRoom = id.RoomID(managementRoom.String)
	u.SpaceRoom = id.RoomID(spaceRoom.String)
	return u, nil
}

