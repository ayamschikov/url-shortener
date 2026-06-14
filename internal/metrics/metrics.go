// Package metrics defines the Prometheus metrics exported by the service.
//
// Everything lives in one place so the rest of the code just calls e.g.
// metrics.RequestsTotal.WithLabelValues(...).Inc(). The metrics are registered
// on the default Prometheus registry via promauto, so the go_* runtime and
// process_* collectors (goroutines, memory, GC) are exported automatically too.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal — Counter: total HTTP requests, split by method, route
	// (the chi pattern, NOT the raw path) and status code. Read with rate()
	// to get RPS (the "R" in RED) and error rate (the "E", filtered by status).
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests, by method, route and status code.",
		},
		[]string{"method", "route", "status"},
	)

	// RequestDuration — Histogram: request latency in seconds, bucketed.
	// Read with histogram_quantile() to get p50/p95/p99 (the "D" in RED).
	// DefBuckets (.005 .. 10s) are fine for typical HTTP latencies.
	// We label only by method+route (not status) to keep the number of
	// time-series (buckets × labels) reasonable.
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds, by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	// RequestsInFlight — Gauge: number of requests being served right now.
	// Goes up and down; a saturation signal (the "S" in USE).
	RequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being served.",
		},
	)

	// CacheEvents — Counter: cache lookups split by result (hit / miss).
	// Lets us watch the cache hit ratio and tie it to the latency tail:
	// a miss goes to PostgreSQL and is slower, so a drop in hit ratio
	// should show up as a rising p99.
	CacheEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "url_cache_events_total",
			Help: "URL cache lookups, by result (hit or miss).",
		},
		[]string{"result"},
	)
)
