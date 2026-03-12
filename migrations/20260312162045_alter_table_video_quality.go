package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// UP
		_, err := db.Exec(`
			ALTER TABLE video_qualities
			ADD COLUMN IF NOT EXISTS thumbnail TEXT
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		// DOWN
		_, err := db.Exec(`
			ALTER TABLE video_qualities
			DROP COLUMN IF EXISTS thumbnail
		`)
		return err
	})
}