package upgrades

import (
	"context"
	"embed"
	"errors"

	"go.mau.fi/util/dbutil"
)

var Table dbutil.UpgradeTable

//go:embed *.sql
var rawUpgrades embed.FS

func init() {
	Table.Register(-1, 12, 0, "Unsupported version", false, func(ctx context.Context, database *dbutil.Database) error {
		return errors.New("please upgrade to mautrix-imap-bridge v0.4.3 before upgrading to a newer version")
	})
	Table.Register(1, 13, 0, "Jump to version 13", false, func(ctx context.Context, database *dbutil.Database) error {
		return nil
	})
	Table.RegisterFS(rawUpgrades)
}
