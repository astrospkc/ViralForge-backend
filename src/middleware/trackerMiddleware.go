package middleware

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
	"viralforge/src/metrics"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

var RequestLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// RequestTracker captures request ID, execution duration, metrics, and structured logs.
func RequestTracker() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// 1. In-flight requests gauge
		metrics.HttpRequestsInFlight.Inc()
		defer metrics.HttpRequestsInFlight.Dec()

		// 2. Correlation Request ID
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Set("X-Request-ID", reqID)
		c.Locals("requestId", reqID)

		// 3. Process next handler / pipeline
		chainErr := c.Next()

		// 4. Calculate timing
		duration := time.Since(start)
		durationSec := duration.Seconds()
		durationMs := float64(duration.Microseconds()) / 1000.0
		status := c.Response().StatusCode()
		statusStr := strconv.Itoa(status)

		// Determine route template or fallback to sanitized path
		routePattern := c.Route().Path
		if routePattern == "" {
			routePattern = c.Path()
		}
		method := c.Method()

		// 5. Update Prometheus Metrics
		metrics.HttpRequestsTotal.WithLabelValues(method, routePattern, statusStr).Inc()
		metrics.HttpRequestDuration.WithLabelValues(method, routePattern, statusStr).Observe(durationSec)

		// 6. Extract TraceID if present from OTel
		var traceID string
		span := trace.SpanFromContext(c.Context())
		if span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}

		// 7. Extract authenticated user_id if present
		var userID any
		if userLocals := c.Locals("user"); userLocals != nil {
			if claims, ok := userLocals.(jwt.MapClaims); ok {
				if uid, ok := claims["user_id"]; ok {
					userID = uid
				}
			}
		}

		// 8. Structured Logging
		logFields := []any{
			slog.String("request_id", reqID),
			slog.String("method", method),
			slog.String("path", c.Path()),
			slog.String("route", routePattern),
			slog.Int("status", status),
			slog.Float64("duration_ms", durationMs),
			slog.String("ip", c.IP()),
		}

		if traceID != "" {
			logFields = append(logFields, slog.String("trace_id", traceID))
		}
		if userID != nil {
			logFields = append(logFields, slog.Any("user_id", userID))
		}
		if chainErr != nil {
			logFields = append(logFields, slog.String("error", chainErr.Error()))
		}

		if status >= 500 || chainErr != nil {
			RequestLogger.Error("http_request_error", logFields...)
		} else if durationMs > 500 {
			logFields = append(logFields, slog.String("warning", fmt.Sprintf("request took %.2fms", durationMs)))
			RequestLogger.Warn("http_slow_request", logFields...)
		} else {
			RequestLogger.Info("http_request", logFields...)
		}

		return chainErr
	}
}
