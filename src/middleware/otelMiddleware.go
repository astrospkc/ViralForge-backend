package middleware

import (
	"context"
	"log"
	"time"
	"viralforge/src/utils"

	"github.com/gofiber/contrib/v3/otel"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

func OtelMiddleware(app *fiber.App) {
	// Initialize OpenTelemetry tracer
	tp, err := utils.InitTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := tp.Shutdown(ctx); err != nil {
			log.Printf(
				"error shutting down tracer provider: %v",
				err,
			)
		}
	}()

	// Standard middleware
	app.Use(recover.New())

	// OpenTelemetry tracing middleware
	app.Use(otel.Middleware())

	// Structured tracking, duration, metrics, and correlation IDs
	app.Use(RequestTracker())

	log.Println("otel & tracking middleware started")
}
