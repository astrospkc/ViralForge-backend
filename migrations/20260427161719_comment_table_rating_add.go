package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE comments
			ADD COLUMN IF NOT EXISTS rating INTEGER;
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE comments
			DROP COLUMN IF EXISTS rating;
		`)
		return err
	})
}