package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// UP
		_, err := db.Exec(`
			ALTER TABLE comments
			ALTER COLUMN id TYPE INTEGER
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		// DOWN
		_, err := db.Exec(`
			ALTER TABLE comments
			ALTER COLUMN id TYPE INTEGER
		`)
		return err
	})
}