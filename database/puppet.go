package database

import (
    "time"
    "database/sql"

    "go.mau.fi/util/dbutil"
    "maunium.net/go/mautrix/id"
)

type PuppetQuery struct {
	*dbutil.QueryHelper[*Puppet]
}

type Puppet struct {
	qh *dbutil.QueryHelper[*Puppet]

	EmailAddress string
	AvatarPath   string
	AvatarHash   string
	AvatarURL    id.ContentURI
	NameSet      bool
	AvatarSet    bool

	IsRegistered     bool
	ContactInfoSet   bool
	ProfileFetchedAt time.Time

	CustomMXID  id.UserID
}


func (p *Puppet) Scan(row dbutil.Scannable) (*Puppet, error) {
	var customMXID sql.NullString
	var profileFetchedAt sql.NullInt64
	err := row.Scan(
		&p.EmailAddress,
		&p.AvatarPath,
		&p.AvatarHash,
		&p.AvatarURL,
		&p.NameSet,
		&p.AvatarSet,
		&p.ContactInfoSet,
		&p.IsRegistered,
		&profileFetchedAt,
		&customMXID,
	)
	if err != nil {
		return nil, err
	}
	p.CustomMXID = id.UserID(customMXID.String)
	if profileFetchedAt.Valid {
		p.ProfileFetchedAt = time.UnixMilli(profileFetchedAt.Int64)
	}
	return p, nil
}

