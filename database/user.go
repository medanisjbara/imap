package database

import (
	"context"
	"database/sql"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

const (
	getUserByMXIDQuery         = `SELECT mxid, email_address, management_room, space_room FROM "user" WHERE mxid=$1`
	getUserByEmailAddressQuery = `SELECT mxid, email_address, management_room, space_room FROM "user" WHERE email_address=$1`
	getAllLoggedInUsersQuery   = `SELECT mxid, email_address, management_room, space_room FROM "user" WHERE cookies IS NOT NULL`
	insertUserQuery            = `INSERT INTO "user" (mxid, email_address, management_room, space_room) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	updateUserQuery            = `UPDATE "user" SET meta_id=$2, wa_device_id=$3, cookies=$4, inbox_fetched=$5, management_room=$6, space_room=$7 WHERE mxid=$1`
)

type UserQuery struct {
	*dbutil.QueryHelper[*User]
}

func (uq *UserQuery) GetByMXID(ctx context.Context, mxid id.UserID) (*User, error) {
	return uq.QueryOne(ctx, getUserByMXIDQuery, mxid)
}

func (uq *UserQuery) GetByEmailAddress(ctx context.Context, address string) (*User, error) {
	return uq.QueryOne(ctx, getUserByEmailAddressQuery, address)
}

func (uq *UserQuery) GetAllLoggedIn(ctx context.Context) ([]*User, error) {
	return uq.QueryMany(ctx, getAllLoggedInUsersQuery)
}

type User struct {
	qh *dbutil.QueryHelper[*User]

	MXID           id.UserID
	EmailAddress   string
	ManagementRoom id.RoomID
	SpaceRoom      id.RoomID
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

func (u *User) sqlVariables() []any {
	return []any{
		u.MXID,
		dbutil.StrPtr(u.EmailAddress),
		dbutil.StrPtr(u.ManagementRoom),
		dbutil.StrPtr(u.SpaceRoom),
	}
}

func (u *User) Insert(ctx context.Context) error {
	return u.qh.Exec(ctx, insertUserQuery, u.sqlVariables()...)
}

func (u *User) Update(ctx context.Context) error {
	return u.qh.Exec(ctx, updateUserQuery, u.sqlVariables()...)
}
