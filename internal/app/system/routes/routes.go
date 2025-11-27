// internal/app/system/routes/routes.go
package routes

import (
	"net/http/pprof"

	"github.com/dalemusser/stratalog/internal/app/features/health"
	"github.com/dalemusser/stratalog/internal/app/features/version"
	"github.com/dalemusser/stratalog/internal/app/system/apikey"
	"github.com/dalemusser/stratalog/internal/app/system/handler"
	"github.com/dalemusser/stratalog/internal/app/system/metrics"
	"github.com/go-chi/chi/v5"
)

// RegisterAllRoutes mounts the routes for every feature in one place,
// passing `h` to each feature that needs DB or config references.
func RegisterAllRoutes(r chi.Router, h *handler.Handler) {
	// Public health endpoint
	health.MountRoutes(r, h)
	version.MountRoutes(r, h)

	// Protected system endpoints: /metrics and /debug/pprof/*
	r.Group(func(r chi.Router) {
		// Require admin (or ingest) API key:
		// - uses admin_api_key if set
		// - falls back to ingest_api_key if admin_api_key is empty
		r.Use(apikey.RequireAdminKey(h.Cfg))

		// Prometheus scrape endpoint
		r.Handle("/metrics", metrics.MetricsRoute())

		// pprof debug endpoints
		mountPprof(r)
	})
}

// mountPprof wires the standard net/http/pprof handlers under /debug/pprof.
func mountPprof(r chi.Router) {
	r.Route("/debug/pprof", func(r chi.Router) {
		r.Get("/", pprof.Index)
		r.Get("/cmdline", pprof.Cmdline)
		r.Get("/profile", pprof.Profile)
		r.Get("/symbol", pprof.Symbol)
		r.Post("/symbol", pprof.Symbol)
		r.Get("/trace", pprof.Trace)

		profiles := []string{"goroutine", "heap", "allocs", "block", "mutex", "threadcreate"}
		for _, name := range profiles {
			r.Handle("/"+name, pprof.Handler(name))
		}
	})
}

/*
From your machine, you’ll be able to do things like:

go tool pprof http://sussout.games/debug/pprof/profile

*/

/*
How the key is supplied:

For both /metrics and /debug/pprof/*, RequireAdminKey will accept the key as any of:
	•	Authorization: Bearer <admin or ingest key>
	•	X-API-Key: <admin or ingest key>
	•	?api_key=<admin or ingest key> (handy in a browser for quick checks)

So for Prometheus/AMP scraping you can, for example, configure:
	•	Extra HTTP header: X-API-Key: <admin_api_key>
or
	•	Authorization: Bearer <admin_api_key>
*/

/* RequireJSON example:
r.Group(func(r chi.Router) {
	r.Use(apikey.RequireIngestKey(h.Cfg))
	r.Use(httputil.RequireJSON())
	r.Post("/logs", h.LogsHandler)
	r.Post("/logs/batch", h.BatchLogsHandler)
})
*/
