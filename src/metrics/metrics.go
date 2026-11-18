package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Requests Counter
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "viralforge_http_requests_total",
			Help: "Total count of HTTP requests processed, partitioned by method, route, and status code.",
		},
		[]string{"method", "route", "status_code"},
	)

	// HTTP Request Duration Histogram
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "viralforge_http_request_duration_seconds",
			Help: "Duration of HTTP requests in seconds, partitioned by method, route, and status code.",
			Buckets: []float64{
				0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
			},
		},
		[]string{"method", "route", "status_code"},
	)

	// Active In-Flight Requests Gauge
	HttpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "viralforge_http_requests_in_flight",
			Help: "Current number of in-flight HTTP requests.",
		},
	)

	// Database Query Duration Histogram
	DbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "viralforge_db_query_duration_seconds",
			Help: "Duration of database queries in seconds, partitioned by operation type.",
			Buckets: []float64{
				0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0,
			},
		},
		[]string{"operation"},
	)

	// Slow Database Queries Counter (>200ms)
	DbSlowQueriesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "viralforge_db_slow_queries_total",
			Help: "Total number of database queries that exceeded 200ms.",
		},
	)
)
