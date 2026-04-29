package migrations

import (
	"context"

	"github.com/uptrace/bun"
)


func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE video_uploads
			RENAME COLUMN "mastercdnurl" TO master_cdn_url;
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE video_uploads
			RENAME COLUMN master_cdn_url TO "mastercdnurl";
		`)
		return err
	})
}