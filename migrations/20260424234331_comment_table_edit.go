package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {

		// 🔥 1. Drop FK constraint on root_comment_id (if exists)
		_, err := db.Exec(`
			ALTER TABLE comments
			DROP CONSTRAINT IF EXISTS fk_comments_root;
		`)
		if err != nil {
			return err
		}

		// 🔥 2. Ensure column allows NULL
		_, err = db.Exec(`
			ALTER TABLE comments
			ALTER COLUMN root_comment_id DROP NOT NULL;
		`)
		if err != nil {
			return err
		}

		// 🔥 3. Clean bad data (important!)
		// convert invalid 0 → NULL
		_, err = db.Exec(`
			UPDATE comments
			SET root_comment_id = NULL
			WHERE root_comment_id = 0;
		`)
		if err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {

		// 🔻 DOWN migration (rollback)

		// ⚠️ Only safe if no NULL values exist
		_, err := db.Exec(`
			UPDATE comments
			SET root_comment_id = id
			WHERE root_comment_id IS NULL;
		`)
		if err != nil {
			return err
		}

		// Re-add NOT NULL (optional)
		_, err = db.Exec(`
			ALTER TABLE comments
			ALTER COLUMN root_comment_id SET NOT NULL;
		`)
		if err != nil {
			return err
		}

		// Re-add FK constraint
		_, err = db.Exec(`
			ALTER TABLE comments
			ADD CONSTRAINT fk_comments_root
			FOREIGN KEY (root_comment_id)
			REFERENCES comments(id)
			ON DELETE CASCADE;
		`)
		if err != nil {
			return err
		}

		return nil
	})
}