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

	// Define your database query structs here
}

func New(db *dbutil.Database) *Database {
	db.UpgradeTable = upgrades.Table
	return &Database{
		Database: db,
		// Initialize your query structs here using dbutil.MakeQueryHelper
		// Example:
		// MyQuery: &MyQuery{dbutil.MakeQueryHelper(db, newMyQuery)},
	}
}
