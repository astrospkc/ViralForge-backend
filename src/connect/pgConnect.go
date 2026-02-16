package connect

import (
	"context"
	"database/sql"

	"viralforge/src/env"
	"viralforge/src/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)
var Db *bun.DB




func PgConnect() {
	ctx := context.Background()
	envs:=env.NewEnv()
	db_url:= envs.SUPABASE_DB_URL
	sqldb := sql.OpenDB(pgdriver.NewConnector(
	pgdriver.WithDSN(db_url),
	))
	Db= bun.NewDB(sqldb, pgdialect.New())

	Db.NewCreateTable().Model((*models.User)(nil)).Exec(ctx)

}