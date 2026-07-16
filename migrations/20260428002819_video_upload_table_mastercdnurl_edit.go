package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE video_uploads
			ALTER COLUMN master_cdn_url TYPE TEXT;
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		// rollback depends on previous type (example VARCHAR)
		_, err := db.Exec(`
			ALTER TABLE video_uploads
			ALTER COLUMN master_cdn_url TYPE VARCHAR(255);
		`)
		return err
	})
}