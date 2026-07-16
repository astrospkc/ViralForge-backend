package database

import (
	"context"
	"fmt"
	"viralforge/migrations"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

func RunMigrations(ctx context.Context, db *bun.DB) error {

	migrator := migrate.NewMigrator(db, migrations.Migrations)

	if err := migrator.Init(ctx); err != nil {
		fmt.Println("error 1")
		return err
	}

	if err := migrator.Lock(ctx); err != nil {
		fmt.Println("error 2")

		return err
	}
	defer migrator.Unlock(ctx)

	group, err := migrator.Migrate(ctx)
	if err != nil {
		fmt.Println("error 3")

		return err
	}

	println("migrated:", group.ID)

	return nil
}