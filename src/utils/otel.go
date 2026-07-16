package utils

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// initTracer sets up OpenTelemetry with OTLP exporter
// This sends traces to collectors like Jaeger, OneUptime, or OTEL Collector
func InitTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// Create OTLP trace exporter using gRPC
	// By default connects to localhost:4317
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithInsecure(), // Use WithTLSCredentials in production
		otlptracegrpc.WithEndpoint("localhost:4317"),
	)
	if err != nil {
		return nil, err
	}

	// Define resource attributes identifying your service
	// These appear on every span and help identify the service
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("fiber-api-service"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.DeploymentEnvironmentKey.String("production"),
		semconv.TelemetrySDKLanguageKey.String("go"),
	)

	// Create tracer provider with batch span processor
	// Batching improves performance by reducing network overhead
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set as global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	// W3C Trace Context is the standard for distributed tracing
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tp, nil
}
