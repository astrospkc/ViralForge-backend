package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE video_uploads
			ADD COLUMN IF NOT EXISTS MasterCdnUrl TEXT;
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE video_uploads
			DROP COLUMN IF EXISTS MasterCdnUrl;
		`)
		return err
	})
}