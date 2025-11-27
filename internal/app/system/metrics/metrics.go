package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests.",
			// buckets in seconds
			Buckets: []float64{0.01, 0.1, 0.3, 1.2, 5},
		},
		[]string{"path", "method", "status"},
	)
)

// MustRegisterMetrics registers the histogram with Prometheus's default registry.
// Call this once in init() of main package.
func MustRegisterMetrics() {
	prometheus.MustRegister(reqDuration)
}

// Metrics records duration & status.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		duration := time.Since(start).Seconds()

		reqDuration.WithLabelValues(
			r.URL.Path, r.Method, strconv.Itoa(ww.Status()),
		).Observe(duration)
	})
}

// RegisterDefaultPrometheus registers Go GC & process collectors.
// Call once in main(). It is safe if someone else already registered them.
func RegisterDefaultPrometheus() {
	// Go runtime metrics
	if err := prometheus.Register(collectors.NewGoCollector()); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// Only panic on unexpected errors
			panic(err)
		}
	}

	// Process metrics (CPU, RSS, FDs, etc.)
	if err := prometheus.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}
}
