package database

import (
    "context"
    "strings"
    "fmt"

    "github.com/lib/pq"
	"go.mau.fi/util/dbutil"
	"maunium.net/go/mautrix/id"
)

// Queries
// Message attrs: Sender, Timestamp, PartIndex, EmailAddress, EmailReceiver, MXID, RoomID,
const (
    getMessageByMXIDQuery = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE mxid=$1
    `
    getMessagePartByEmailAddressQuery = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE sender=$1 AND timestamp=$2 AND part_index=$3 AND email_receiver=$4
    `
    getLastMessagePartByEmailAddressQuery = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE sender=$1 AND timestamp=$2 AND email_receiver=$3
        ORDER BY part_index DESC LIMIT 1
    `
    getAllMessagePartsByEmailAddressQuery = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE sender=$1 AND timestamp=$2 AND email_receiver=$3
    `
    getMessageLastPartByEmailAddressWithUnknownReceiverQuery = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE sender=$1 AND timestamp=$2 AND (email_receiver=$3 OR email_receiver='00000000-0000-0000-0000-000000000000')
        ORDER BY part_index DESC LIMIT 1
    `
    getManyMessagesByEmailAddressQueryPostgres = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE sender=$1 AND (email_receiver=$2 OR email_receiver=$3) AND timestamp=ANY($4)
        ORDER BY timestamp DESC, part_index DESC
    `
    getManyMessagesByEmailAddressQuerySQLite = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE sender=?1 AND (email_receiver=?2 OR email_receiver=?3) AND timestamp IN (?4)
        ORDER BY timestamp DESC, part_index DESC
    `
    getFirstBeforeQuery = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE mx_room=$1 AND timestamp <= $2
        ORDER BY timestamp DESC
        LIMIT 1
    `
    getMessagesBetweenTimeQuery = `
        SELECT sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room FROM message
        WHERE email_address=$1 AND email_receiver=$2 AND timestamp>$3 AND timestamp<=$4 AND part_index=0
        ORDER BY timestamp ASC
    `
    insertMessageQuery = `
        INSERT INTO message (sender, timestamp, part_index, email_address, email_receiver, mxid, mx_room)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `
    deleteMessageQuery = `
        DELETE FROM message
        WHERE sender=$1 AND timestamp=$2 AND part_index=$3 AND email_receiver=$4
    `
    updateMessageTimestampQuery = `
        UPDATE message SET timestamp=$4 WHERE sender=$1 AND timestamp=$2 AND email_receiver=$3
    `


)

// Message Query
type MessageQuery struct {
	*dbutil.QueryHelper[*Message]
}

func newMessage(qh *dbutil.QueryHelper[*Message]) *Message {
	return &Message{qh: qh}
}

func (mq *MessageQuery) GetByMXID(ctx context.Context, mxid id.EventID) (*Message, error) {
	return mq.QueryOne(ctx, getMessageByMXIDQuery, mxid)
}

func (mq *MessageQuery) GetByEmailAddress(ctx context.Context, sender string, timestamp uint64, partIndex int, receiver string) (*Message, error) {
	return mq.QueryOne(ctx, getMessagePartByEmailAddressQuery, sender, timestamp, partIndex, receiver)
}

func (mq *MessageQuery) GetLastPartByEmailAddress(ctx context.Context, sender string, timestamp uint64, receiver string) (*Message, error) {
	return mq.QueryOne(ctx, getLastMessagePartByEmailAddressQuery, sender, timestamp, receiver)
}

func (mq *MessageQuery) GetAllPartsByEmailAddress(ctx context.Context, sender string, timestamp uint64, receiver string) ([]*Message, error) {
	return mq.QueryMany(ctx, getAllMessagePartsByEmailAddressQuery, sender, timestamp, receiver)
}

func (mq *MessageQuery) GetAllBetweenTimestamps(ctx context.Context, key PortalKey, min, max uint64) ([]*Message, error) {
	return mq.QueryMany(ctx, getMessagesBetweenTimeQuery, key.ThreadID, key.Receiver, int64(min), int64(max))
}

func (mq *MessageQuery) GetLastPartByEmailAddressWithUnknownReceiver(ctx context.Context, sender string, timestamp uint64, receiver string) (*Message, error) {
	return mq.QueryOne(ctx, getMessageLastPartByEmailAddressWithUnknownReceiverQuery, sender, timestamp, receiver)
}

func (mq *MessageQuery) GetManyByEmailAddress(ctx context.Context, sender string, timestamps []uint64, receiver string, strictReceiver bool) ([]*Message, error) {
	receiver2 := ""
	if strictReceiver {
		receiver2 = receiver
	}
	if mq.GetDB().Dialect == dbutil.Postgres {
		int64Array := make([]int64, len(timestamps))
		for i, timestamp := range timestamps {
			int64Array[i] = int64(timestamp)
		}
		return mq.QueryMany(ctx, getManyMessagesByEmailAddressQueryPostgres, sender, receiver, receiver2, pq.Array(int64Array))
	} else {
		const varargIndex = 3
		arguments := make([]any, len(timestamps)+varargIndex)
		placeholders := make([]string, len(timestamps))
		arguments[0] = sender
		arguments[1] = receiver
		arguments[2] = receiver2
		for i, timestamp := range timestamps {
			arguments[i+varargIndex] = timestamp
			placeholders[i] = fmt.Sprintf("?%d", i+varargIndex+1)
		}
		return mq.QueryMany(ctx, strings.Replace(getManyMessagesByEmailAddressQuerySQLite, fmt.Sprintf("?%d", varargIndex+1), strings.Join(placeholders, ", "), 1), arguments...)
	}
}

// Message
type Message struct {
	qh *dbutil.QueryHelper[*Message]

	Sender    string
	Timestamp uint64
	PartIndex int

	EmailAddress  int64
	EmailReceiver int64

	MXID   id.EventID
	RoomID id.RoomID
}

func (msg *Message) Scan(row dbutil.Scannable) (*Message, error) {
	return dbutil.ValueOrErr(msg, row.Scan(
		&msg.Sender,
		&msg.Timestamp,
		&msg.PartIndex,
		&msg.EmailAddress,
		&msg.EmailReceiver,
		&msg.MXID,
		&msg.RoomID,
	))
}

func (msg *Message) sqlVariables() []any {
	return []any{msg.Sender, msg.Timestamp, msg.PartIndex, msg.EmailAddress, msg.EmailReceiver, msg.MXID, msg.RoomID}
}

func (msg *Message) Insert(ctx context.Context) error {
	return msg.qh.Exec(ctx, insertMessageQuery, msg.sqlVariables()...)
}

func (msg *Message) Delete(ctx context.Context) error {
	return msg.qh.Exec(ctx, deleteMessageQuery, msg.Sender, msg.Timestamp, msg.PartIndex, msg.EmailReceiver)
}

func (msg *Message) SetTimestamp(ctx context.Context, editTime uint64) error {
	return msg.qh.Exec(ctx, updateMessageTimestampQuery, msg.Sender, msg.Timestamp, msg.EmailReceiver, editTime)
}
