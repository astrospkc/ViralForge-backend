package connect

import (
	"context"
	"database/sql"
	"log"
	"time"

	// "time"

	"viralforge/src/env"
	"viralforge/src/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)
var Db *bun.DB

func PgConnect() {
	ctx := context.Background()

	envs := env.NewEnv()
	dbURL := envs.NEON_DB_URL

	// Create connector (Bun uses pgdriver internally)
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dbURL),
	))

	// Neon works best with controlled pool sizes
	sqldb.SetMaxOpenConns(10)
	sqldb.SetMaxIdleConns(5)
	sqldb.SetConnMaxLifetime(time.Minute * 5)

	// IMPORTANT: verify connection
	if err := sqldb.PingContext(ctx); err != nil {
		log.Fatalf("Neon DB connection failed: %v", err)
	}

	Db = bun.NewDB(sqldb, pgdialect.New())

	// Create tables safely
	if _, err := Db.NewCreateTable().
		Model((*models.User)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		log.Fatal(err)
	}

	if _, err := Db.NewCreateTable().
		Model((*models.VideoUpload)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		log.Fatal(err)
	}

	if _, err:= Db.NewCreateTable().Model((*models.VideoDetailsUpload)(nil)).IfNotExists().Exec(ctx); err!=nil{
		log.Fatal((err))
	}			
	log.Println("Connected to Neon DB successfully 🚀")
}