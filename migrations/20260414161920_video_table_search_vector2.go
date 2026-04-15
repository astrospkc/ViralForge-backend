package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			// 1. add search vector column
			_, err := db.Exec(`
				ALTER TABLE video_uploads
				ADD COLUMN IF NOT EXISTS search_vector tsvector;
			`)
			if err != nil {
				return err
			}

			// 2. backfill existing rows
			_, err = db.Exec(`
				UPDATE video_uploads
				SET search_vector =
					setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
					setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
					setweight(to_tsvector('english', coalesce(tags, '')), 'C');
			`)
			if err != nil {
				return err
			}

			// 3. create gin index
			_, err = db.Exec(`
				CREATE INDEX IF NOT EXISTS idx_videos_search_vector
				ON video_uploads
				USING GIN(search_vector);
			`)
			if err != nil {
				return err
			}

			// 4. trigger function
			_, err = db.Exec(`
				CREATE OR REPLACE FUNCTION videos_search_vector_update()
				RETURNS trigger AS $$
				BEGIN
					NEW.search_vector :=
						setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
						setweight(to_tsvector('english', coalesce(NEW.description, '')), 'B') ||
						setweight(to_tsvector('english', coalesce(NEW.tags, '')), 'C');
					RETURN NEW;
				END
				$$ LANGUAGE plpgsql;
			`)
			if err != nil {
				return err
			}

			// 5. create trigger
			_, err = db.Exec(`
				CREATE TRIGGER trg_videos_search_vector
				BEFORE INSERT OR UPDATE ON video_uploads
				FOR EACH ROW
				EXECUTE FUNCTION videos_search_vector_update();
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.Exec(`
				DROP TRIGGER IF EXISTS trg_videos_search_vector ON video_uploads;
			`)
			if err != nil {
				return err
			}

			_, err = db.Exec(`
				DROP FUNCTION IF EXISTS videos_search_vector_update();
			`)
			if err != nil {
				return err
			}

			_, err = db.Exec(`
				DROP INDEX IF EXISTS idx_videos_search_vector;
			`)
			if err != nil {
				return err
			}

			_, err = db.Exec(`
				ALTER TABLE video_uploads
				DROP COLUMN IF EXISTS search_vector;
			`)
			return err
		},
	)
}