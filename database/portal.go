package database

import (
	"context"
	"database/sql"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

const (
	portalBaseSelect = `
        SELECT thread_id, receiver, mxid, name, email_address, topic, avatar_path, avatar_hash, avatar_url,
               name_set, avatar_set, topic_set, revision, encrypted, relay_user_id, expiration_time
        FROM portal
    `
	getAllPortalsWithMXIDQuery = portalBaseSelect + `WHERE mxid IS NOT NULL`
	getPortalsByAddressQuery   = portalBaseSelect + `WHERE email_address=$1`
	getPortalsByThreadIDQuery  = portalBaseSelect + `WHERE thread_id=$1 AND receiver=$2`
	getPortalByMXIDQuery       = portalBaseSelect + `WHERE mxid=$1`
	getPortalsByReceiverQuery  = portalBaseSelect + `WHERE receiver=$1`
	insertPortalQuery          = `
        INSERT INTO portal (
            thread_id, receiver, mxid, name, email_address, topic, avatar_path, avatar_hash, avatar_url,
            name_set, avatar_set, topic_set, revision, encrypted, relay_user_id, expiration_time
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
    `
	updatePortalQuery = `
        UPDATE portal SET
            mxid=$3, name=$4, email_address=$5, topic=$6, avatar_path=$7, avatar_hash=$8, avatar_url=$9,
            name_set=$10, avatar_set=$11, topic_set=$12, revision=$13, encrypted=$14, relay_user_id=$15, expiration_time=$16
        WHERE thread_id=$1 AND receiver=$2
    `
	deletePortalQuery = `DELETE FROM portal WHERE thread_id=$1 AND receiver=$2`
	reIDPortalQuery   = `UPDATE portal SET thread_id=$2 WHERE thread_id=$1 AND receiver=$3`
)

type PortalKey struct {
	ThreadID string
	Receiver string
}

type PortalQuery struct {
	*dbutil.QueryHelper[*Portal]
}

type Portal struct {
	qh *dbutil.QueryHelper[*Portal]

	PortalKey
	MXID           id.RoomID
	Name           string
	EmailAddress   string
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

func NewPortalKey(threadID string, receiver string) PortalKey {
	return PortalKey{
		ThreadID: threadID,
		Receiver: receiver,
	}
}

func newPortal(qh *dbutil.QueryHelper[*Portal]) *Portal {
	return &Portal{qh: qh}
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
		&p.EmailAddress,
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

func (p *Portal) sqlVariables() []any {
	return []any{
		p.ThreadID,
		p.Receiver,
		dbutil.StrPtr(p.MXID),
		p.Name,
		p.EmailAddress,
		p.Topic,
		p.AvatarPath,
		p.AvatarHash,
		&p.AvatarURL,
		p.NameSet,
		p.AvatarSet,
		p.TopicSet,
		p.Revision,
		p.Encrypted,
		p.RelayUserID,
		p.ExpirationTime,
	}
}

func (p *Portal) Insert(ctx context.Context) error {
	return p.qh.Exec(ctx, insertPortalQuery, p.sqlVariables()...)
}

func (p *Portal) Update(ctx context.Context) error {
	return p.qh.Exec(ctx, updatePortalQuery, p.sqlVariables()...)
}

func (p *Portal) Delete(ctx context.Context) error {
	return p.qh.Exec(ctx, deletePortalQuery, p.ThreadID, p.Receiver)
}

func (p *Portal) ReID(ctx context.Context, newID string) error {
	return p.qh.Exec(ctx, reIDPortalQuery, p.ThreadID, newID, p.Receiver)
}

func (pq *PortalQuery) GetByMXID(ctx context.Context, mxid id.RoomID) (*Portal, error) {
	return pq.QueryOne(ctx, getPortalByMXIDQuery, mxid)
}

func (pq *PortalQuery) FindPrivateChatsWith(ctx context.Context, address string) ([]*Portal, error) {
	return pq.QueryMany(ctx, getPortalsByAddressQuery, address)
}

func (pq *PortalQuery) FindPrivateChatsOf(ctx context.Context, receiver string) ([]*Portal, error) {
	return pq.QueryMany(ctx, getPortalsByReceiverQuery, receiver)
}

func (pq *PortalQuery) GetByThreadID(ctx context.Context, pk PortalKey) (*Portal, error) {
	return pq.QueryOne(ctx, getPortalsByThreadIDQuery, pk.ThreadID, pk.Receiver)
}
