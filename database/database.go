// database.go

package database

import (
	_ "embed"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/util/dbutil"

	"mybridge/database/upgrades"
)

type Database struct {
	*dbutil.Database

	User    *UserQuery
	Portal  *PortalQuery
	Puppet  *PuppetQuery
	Message *MessageQuery
}

func New(db *dbutil.Database) *Database {
	db.UpgradeTable = upgrades.Table
	return &Database{
		Database: db,
		User:     &UserQuery{dbutil.MakeQueryHelper(db, newUser)},
		Portal:   &PortalQuery{dbutil.MakeQueryHelper(db, newPortal)},
		Puppet:   &PuppetQuery{dbutil.MakeQueryHelper(db, newPuppet)},
		Message:  &MessageQuery{dbutil.MakeQueryHelper(db, newMessage)},
	}
}
