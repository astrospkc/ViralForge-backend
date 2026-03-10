package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/url"

	"viralforge/cmd/worker"
	"viralforge/src/connect"
	"viralforge/src/env"
	"viralforge/src/routes"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/hibiken/asynq"
	"github.com/hibiken/asynqmon"
)

func main() {
    envs := env.NewEnv()
    connect.PgConnect()
    connect.AsynqConnect()

    app := fiber.New()

    app.Use(cors.New(cors.Config{
        AllowOrigins: []string{"http://localhost:5173"},
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