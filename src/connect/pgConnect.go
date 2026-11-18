package connect

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	// "time"

	"viralforge/src/env"
	"viralforge/src/metrics"
	"viralforge/src/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var Db *bun.DB

// QueryTrackerHook measures SQL duration and flags slow queries
type QueryTrackerHook struct{}

func (h *QueryTrackerHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

func (h *QueryTrackerHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)
	durationSec := duration.Seconds()
	durationMs := float64(duration.Microseconds()) / 1000.0

	op := "OTHER"
	q := event.Query
	if len(q) >= 6 {
		prefix := q[:6]
		switch {
		case prefix == "SELECT" || prefix == "select":
			op = "SELECT"
		case prefix == "INSERT" || prefix == "insert":
			op = "INSERT"
		case prefix == "UPDATE" || prefix == "update":
			op = "UPDATE"
		case prefix == "DELETE" || prefix == "delete":
			op = "DELETE"
		}
	}

	metrics.DbQueryDuration.WithLabelValues(op).Observe(durationSec)

	// Flag slow queries (>200ms)
	if duration > 200*time.Millisecond {
		metrics.DbSlowQueriesTotal.Inc()
		log.Printf("⚠️ SLOW DB QUERY [%.2fms] [%s]: %s\n", durationMs, op, event.Query)
	}
}

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
	sqldb.SetConnMaxLifetime(1 * time.Hour)
	sqldb.SetConnMaxIdleTime(30 * time.Minute)

	// IMPORTANT: verify connection
	if err := sqldb.PingContext(ctx); err != nil {
		log.Fatalf("Neon DB connection failed: %v", err)
	}

	Db = bun.NewDB(sqldb, pgdialect.New())
	Db.AddQueryHook(&QueryTrackerHook{})
	fmt.Println("Db; ", Db)

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

	if _, err:= Db.NewCreateTable().Model((*models.VideoQuality)(nil)).IfNotExists().Exec(ctx); err!=nil{
		log.Fatal((err))
	}

	// Ensure PostGIS extension and spec tables exist
	if _, err := Db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS postgis;"); err != nil {
		log.Println("PostGIS extension notice:", err)
	}

	for _, model := range []interface{}{
		(*models.Role)(nil),
		(*models.UserRole)(nil),
		(*models.MerchantProfile)(nil),
		(*models.MerchantMember)(nil),
		(*models.Shop)(nil),
		(*models.Product)(nil),
		(*models.ShopProduct)(nil),
		(*models.UserSavedLocation)(nil),
	} {
		if _, err := Db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			log.Printf("Create table notice for %T: %v\n", model, err)
		}
	}
			
	log.Println("Connected to Neon DB successfully 🚀")
}