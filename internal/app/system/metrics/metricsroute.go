package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsRoute mounts GET /metrics with all registered collectors.
func MetricsRoute() http.Handler { return promhttp.Handler() }
