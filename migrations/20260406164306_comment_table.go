package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// UP
		_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS comments (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				video_id UUID NOT NULL,
				user_id UUID NOT NULL,
				parent_comment_id UUID NULL,
				root_comment_id UUID NOT NULL,
				depth INT NOT NULL DEFAULT 0,
				content TEXT NOT NULL,
				like_count BIGINT NOT NULL DEFAULT 0,
				reply_count BIGINT NOT NULL DEFAULT 0,
				status VARCHAR(20) NOT NULL DEFAULT 'VISIBLE',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				deleted_at TIMESTAMP NULL
			)
		`)
		if err != nil {
			return err
		}

		// Important indexes for scale
		indexQueries := []string{
			`CREATE INDEX IF NOT EXISTS idx_comments_video_created 
			 ON comments(video_id, created_at DESC)`,

			`CREATE INDEX IF NOT EXISTS idx_comments_root_created 
			 ON comments(root_comment_id, created_at ASC)`,

			`CREATE INDEX IF NOT EXISTS idx_comments_parent 
			 ON comments(parent_comment_id)`,

			`CREATE INDEX IF NOT EXISTS idx_comments_user 
			 ON comments(user_id)`,
		}

		for _, q := range indexQueries {
			if _, err := db.Exec(q); err != nil {
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// DOWN
		_, err := db.Exec(`
			DROP TABLE IF EXISTS comments
		`)
		return err
	})
}