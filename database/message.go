package database

import (
    "go.mau.fi/util/dbutil"
    "maunium.net/go/mautrix/id"
)

type MessageQuery struct {
	*dbutil.QueryHelper[*Message]
}

type Message struct {
	qh *dbutil.QueryHelper[*Message]

	Sender    int64
	Timestamp uint64
	PartIndex int

	EmailAddress   int64
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
