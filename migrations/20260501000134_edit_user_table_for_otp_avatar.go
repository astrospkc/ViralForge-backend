package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE users
			ADD COLUMN IF NOT EXISTS confirm_password TEXT,
			ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT FALSE,
			ADD COLUMN IF NOT EXISTS active BOOLEAN DEFAULT TRUE,
			ADD COLUMN IF NOT EXISTS avatar TEXT,
			ADD COLUMN IF NOT EXISTS otp TEXT,
			ADD COLUMN IF NOT EXISTS otp_expiry TIMESTAMP;
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE users
			DROP COLUMN IF EXISTS confirm_password,
			DROP COLUMN IF EXISTS is_verified,
			DROP COLUMN IF EXISTS active,
			DROP COLUMN IF EXISTS avatar,
			DROP COLUMN IF EXISTS otp,
			DROP COLUMN IF EXISTS otp_expiry;
		`)
		return err
	})
}