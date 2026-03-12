package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
	database "viralforge/internal"

	"viralforge/src/env"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main(){
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

	Db := bun.NewDB(sqldb, pgdialect.New())
	
	
	err := database.RunMigrations(ctx,Db)
	if err != nil {
		fmt.Println("error in migration")
		panic(err)
	}
	fmt.Println("migrate done")
}