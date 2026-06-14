package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/ayamschikov/url-shortener/internal/metrics"
)

// Metrics records Prometheus metrics for every HTTP request:
//   - http_requests_total      (Counter)   — by method, route, status
//   - http_request_duration_seconds (Histogram) — by method, route
//   - http_requests_in_flight  (Gauge)     — current concurrency
//
// The /metrics endpoint itself is skipped so Prometheus scrapes don't pollute
// the dashboards.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		metrics.RequestsInFlight.Inc()
		defer metrics.RequestsInFlight.Dec()

		// Wrap so we can read the status code the handler wrote.
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		// RoutePattern is only populated after routing (i.e. after ServeHTTP).
		// Using the pattern ("/{code}") instead of the raw path ("/abc123")
		// keeps label cardinality low — essential for Prometheus.
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}

		status := ww.Status()
		if status == 0 { // handler wrote body without an explicit WriteHeader
			status = http.StatusOK
		}

		metrics.RequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
		metrics.RequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}
