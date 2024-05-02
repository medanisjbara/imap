package database

import (
	"context"
	"database/sql"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

const (
	portalBaseSelect = `
        SELECT thread_id, receiver, mxid, name, topic, avatar_path, avatar_hash, avatar_url,
               name_set, avatar_set, topic_set, revision, encrypted, relay_user_id, expiration_time
        FROM portal
    `
	getAllPortalsWithMXIDQuery = portalBaseSelect + `WHERE mxid IS NOT NULL`
)

type PortalKey struct {
	ThreadID int64
	Receiver int64
}

type PortalQuery struct {
	*dbutil.QueryHelper[*Portal]
}

type Portal struct {
	qh *dbutil.QueryHelper[*Portal]

	PortalKey
	MXID           id.RoomID
	Name           string
	Topic          string
	AvatarPath     string
	AvatarHash     string
	AvatarURL      id.ContentURI
	NameSet        bool
	AvatarSet      bool
	TopicSet       bool
	Revision       uint32
	Encrypted      bool
	RelayUserID    id.UserID
	ExpirationTime uint32
}

func (pq *PortalQuery) GetAllWithMXID(ctx context.Context) ([]*Portal, error) {
	return pq.QueryMany(ctx, getAllPortalsWithMXIDQuery)
}

func (p *Portal) Scan(row dbutil.Scannable) (*Portal, error) {
	var mxid sql.NullString
	err := row.Scan(
		&p.ThreadID,
		&p.Receiver,
		&mxid,
		&p.Name,
		&p.Topic,
		&p.AvatarPath,
		&p.AvatarHash,
		&p.AvatarURL,
		&p.NameSet,
		&p.AvatarSet,
		&p.TopicSet,
		&p.Revision,
		&p.Encrypted,
		&p.RelayUserID,
		&p.ExpirationTime,
	)
	if err != nil {
		return nil, err
	}
	p.MXID = id.RoomID(mxid.String)
	return p, nil
}
