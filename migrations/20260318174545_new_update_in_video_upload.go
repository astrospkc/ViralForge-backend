package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
    Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
		CREATE TYPE publish_status_enum AS ENUM ('draft', 'published')
		`)
		if err != nil {
			return err
		}
        // UP
        _, err = db.Exec(`
            ALTER TABLE video_uploads
            ADD COLUMN IF NOT EXISTS title TEXT,
            ADD COLUMN IF NOT EXISTS description TEXT,
            ADD COLUMN IF NOT EXISTS tags TEXT[],
            ADD COLUMN IF NOT EXISTS likes_count BIGINT DEFAULT 0 NOT NULL,
            ADD COLUMN IF NOT EXISTS views_count BIGINT DEFAULT 0 NOT NULL,
            ADD COLUMN IF NOT EXISTS publish_status publish_status_enum DEFAULT 'draft' NOT NULL
        `)
        return err
    }, func(ctx context.Context, db *bun.DB) error {
        // DOWN
        _, err := db.Exec(`
            ALTER TABLE video_uploads
            DROP COLUMN IF EXISTS title,
            DROP COLUMN IF EXISTS description,
            DROP COLUMN IF EXISTS tags,
            DROP COLUMN IF EXISTS likes_count,
            DROP COLUMN IF EXISTS views_count,
            DROP COLUMN IF EXISTS publish_status
        `)
		if err != nil {
            return err
        }

        _, err = db.Exec(`
            DROP TYPE IF EXISTS publish_status_enum
        `)
        return err
    })
}