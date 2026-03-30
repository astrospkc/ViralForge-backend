package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
    Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		
        // UP
        _, err := db.Exec(`
            ALTER TABLE video_uploads
            ADD COLUMN IF NOT EXISTS category VARCHAR
            
        `)
        return err
    }, func(ctx context.Context, db *bun.DB) error {
        // DOWN
        _, err := db.Exec(`
            ALTER TABLE video_uploads
            DROP COLUMN IF EXISTS category
            
        `)
		
        return err
    })
}