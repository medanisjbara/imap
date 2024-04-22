// mautrix-imap - A Matrix-Email puppeting bridge.
// Copyright (C) 2024 Tulir Asokan
// Copyright (C) 2024 Med Anis Jbara
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package database

import (
	"github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/util/dbutil"

	"github.com/medanisjbara/mautrix-imap/database/upgrades"
)

func init() {
	sqlstore.PostgresArrayWrapper = pq.Array
}

type Database struct {
	*dbutil.Database

	User    *UserQuery
	Portal  *PortalQuery
	Puppet  *PuppetQuery
	Message *MessageQuery
	Thread  *ThreadQuery
}

func New(db *dbutil.Database) *Database {
	db.UpgradeTable = upgrades.Table
	return &Database{
		Database: db,
		User:     &UserQuery{dbutil.MakeQueryHelper(db, newUser)},
		Portal:   &PortalQuery{dbutil.MakeQueryHelper(db, newPortal)},
		Puppet:   &PuppetQuery{dbutil.MakeQueryHelper(db, newPuppet)},
		Thread:   &ThreadQuery{dbutil.MakeQueryHelper(db, newThread)},
		Message:  &MessageQuery{dbutil.MakeQueryHelper(db, newMessage)},
	}
}
