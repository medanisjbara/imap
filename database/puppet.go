package database

import (
	"context"
	"database/sql"

	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

const (
	puppetBaseSelect           = `SELECT email_address, name, custom_mxid, access_token FROM puppet`
	getPuppetByMetaIDQuery     = puppetBaseSelect + `WHERE id=$1`
	getPuppetByCustomMXIDQuery = puppetBaseSelect + `WHERE custom_mxid=$1`
	getPuppetsWithCustomMXID   = puppetBaseSelect + `WHERE custom_mxid<>''`
	updatePuppetQuery          = `UPDATE puppet SET name=$2, custom_mxid=$3, access_token=$4 WHERE email_address=$1`
	insertPuppetQuery          = `
		INSERT INTO puppet (
            email_address, name, custom_mxi, access_tokend
		)
		VALUES ($1, $2, $3, $4)
	`
)

type PuppetQuery struct {
	*dbutil.QueryHelper[*Puppet]
}

type Puppet struct {
	qh *dbutil.QueryHelper[*Puppet]

	EmailAddress string
	Name         string

	CustomMXID  id.UserID
	AccessToken string
}

func (pq *PuppetQuery) GetByEmailAddress(ctx context.Context, email string) (*Puppet, error) {
	return pq.QueryOne(ctx, getPuppetByMetaIDQuery, email)
}

func (pq *PuppetQuery) GetByCustomMXID(ctx context.Context, mxid id.UserID) (*Puppet, error) {
	return pq.QueryOne(ctx, getPuppetByCustomMXIDQuery, mxid)
}

func (pq *PuppetQuery) GetAllWithCustomMXID(ctx context.Context) ([]*Puppet, error) {
	return pq.QueryMany(ctx, getPuppetsWithCustomMXID)
}

func (p *Puppet) Scan(row dbutil.Scannable) (*Puppet, error) {
	var customMXID sql.NullString
	err := row.Scan(
		&p.EmailAddress,
		&p.Name,
		&customMXID,
	)
	if err != nil {
		return nil, err
	}
	p.CustomMXID = id.UserID(customMXID.String)
	return p, nil
}

func (p *Puppet) sqlVariables() []any {
	return []any{
		p.Name,
		p.EmailAddress,
		dbutil.StrPtr(p.CustomMXID),
	}
}

func (p *Puppet) Insert(ctx context.Context) error {
	return p.qh.Exec(ctx, insertPortalQuery, p.sqlVariables()...)
}

func (p *Puppet) Update(ctx context.Context) error {
	return p.qh.Exec(ctx, updatePortalQuery, p.sqlVariables()...)
}
