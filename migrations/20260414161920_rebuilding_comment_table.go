package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// 🔥 STEP 1: Drop existing table
		_, err := db.Exec(`
			DROP TABLE IF EXISTS comments CASCADE;
		`)
		if err != nil {
			return err
		}

		// 🔥 STEP 2: Recreate with correct schema
		_, err = db.Exec(`
			CREATE TABLE comments (
				id BIGSERIAL PRIMARY KEY,
				video_id BIGINT NOT NULL,
				user_id BIGINT NOT NULL,
				parent_comment_id BIGINT NULL,
				root_comment_id BIGINT,
				depth INT NOT NULL DEFAULT 0,
				content TEXT NOT NULL,
				like_count BIGINT NOT NULL DEFAULT 0,
				reply_count BIGINT NOT NULL DEFAULT 0,
				status VARCHAR(20) NOT NULL DEFAULT 'VISIBLE',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				deleted_at TIMESTAMP NULL,

				-- 🔥 Foreign Keys
				CONSTRAINT fk_comments_user 
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,

				CONSTRAINT fk_comments_parent 
					FOREIGN KEY (parent_comment_id) REFERENCES comments(id) ON DELETE CASCADE,

				CONSTRAINT fk_comments_root 
					FOREIGN KEY (root_comment_id) REFERENCES comments(id) ON DELETE CASCADE
			);
		`)
		if err != nil {
			return err
		}

		// 🔥 STEP 3: Indexes (important for scale)
		indexQueries := []string{
			`CREATE INDEX idx_comments_video_created 
			 ON comments(video_id, created_at DESC)`,

			`CREATE INDEX idx_comments_root_created 
			 ON comments(root_comment_id, created_at ASC)`,

			`CREATE INDEX idx_comments_parent 
			 ON comments(parent_comment_id)`,

			`CREATE INDEX idx_comments_user 
			 ON comments(user_id)`,
		}

		for _, q := range indexQueries {
			if _, err := db.Exec(q); err != nil {
				return err
			}
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		// 🔻 DOWN migration
		_, err := db.Exec(`
			DROP TABLE IF EXISTS comments CASCADE;
		`)
		return err
	})
}