package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"viralforge/cmd/worker"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/storage/redis/v3"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
)

func main() {
    envs := env.NewEnv()
    connect.PgConnect()
    connect.AsynqConnect()

    app := fiber.New()

    store := redis.New(redis.Config{
		Host:     envs.AIVEN_HOST,
		Port:     15543,                
		Password: envs.AIVEN_PASSWORD,
		Database:  0,
		TLSConfig:  &tls.Config{}, // or proper TLS config if needed
        PoolSize:  10 * runtime.GOMAXPROCS(0),
	})

    app.Use(limiter.New(limiter.Config{
		Max:        4,
		Expiration: 1 * time.Minute,
		Storage:    store,
        KeyGenerator: func(c fiber.Ctx) string {
            key := c.IP()
            fmt.Println("Limiter Key:", key)
            return key
        },
        LimitReached: func(c fiber.Ctx) error {
            fmt.Println("❌ Rate limit hit for:", c.IP())
            return c.Status(429).SendString("Too many requests")
        },
	}))

    app.Use(cors.New(cors.Config{
        AllowOrigins: []string{"http://localhost:5173","https://www.viralforge.xastros.site/"},
        AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
    }))

    routes.NormalRoutes(app)
	u, err := url.Parse(envs.AIVEN_SERVICE_URI)
    if err != nil {
        log.Fatal("invalid aiven URI:", err)
    }

	log.Println("connecting to:", u.Host)
    // asynqmon — run on a SEPARATE port
    // avoids all the adaptor/prefix issues entirely
    monitor := asynqmon.New(asynqmon.Options{
        RootPath: "/",
        RedisConnOpt: asynq.RedisClientOpt{
            Addr:      u.Host,
            Password:  envs.AIVEN_PASSWORD,
            TLSConfig: &tls.Config{},
        },
    })

    // serve monitor on its own port using standard net/http
    go func() {
        monitorMux := http.NewServeMux()
        monitorMux.Handle("/", monitor)
        log.Println("asynqmon dashboard at http://localhost:8082")
        if err := http.ListenAndServe(":8082", monitorMux); err != nil {
            log.Fatal("monitor server failed:", err)
        }
    }()

    // HTTP server in background
    go func() {
        log.Println("HTTP server starting on :8081")
        if err := app.Listen(":8081"); err != nil {
            log.Fatal("HTTP server failed:", err)
        }
    }()

    // worker blocks main
    log.Println("starting worker server...")
    worker.StartWorkerServer()
}