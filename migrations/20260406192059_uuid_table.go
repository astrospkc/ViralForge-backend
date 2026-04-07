package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		return nil
	})
}