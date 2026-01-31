package logapi

import (
	apistatsstore "github.com/dalemusser/stratalog/internal/app/store/apistats"
	"github.com/dalemusser/stratalog/internal/app/system/apistats"
	"github.com/dalemusser/stratalog/internal/app/system/auth"
	"github.com/dalemusser/stratalog/internal/app/system/ledger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Routes returns the router for the new /api/log endpoints.
// Mounted at /api/log:
//   - POST /api/log/submit - Submit single or batch log entries
//   - GET /api/log/list - List log entries with filters
func Routes(h *Handler, statsRecorder *apistats.Recorder, ledgerConfig ledger.Config, apiKey string, logger *zap.Logger) chi.Router {
	r := chi.NewRouter()

	// Ledger middleware for error logging
	r.Use(ledger.Middleware(ledgerConfig))

	// API key authentication middleware
	r.Use(auth.APIKeyAuth(apiKey, logger))

	// Submit endpoint
	r.Route("/submit", func(r chi.Router) {
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogSubmit)).Post("/", h.SubmitHandler)
	})

	// List endpoint
	r.Route("/list", func(r chi.Router) {
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogList)).Get("/", h.ListHandler)
	})

	return r
}

// LegacyRoutes returns the router for the legacy /logs endpoint.
// This maintains backward compatibility with existing clients.
// Endpoints:
//   - POST /logs - Submit single or batch log entries
//   - GET /logs - List log entries with filters
func LegacyRoutes(h *Handler, statsRecorder *apistats.Recorder, ledgerConfig ledger.Config, apiKey string, logger *zap.Logger) chi.Router {
	r := chi.NewRouter()

	// Ledger middleware for error logging
	r.Use(ledger.Middleware(ledgerConfig))

	// API key authentication middleware
	r.Use(auth.APIKeyAuth(apiKey, logger))

	// API stats recording
	r.Route("/", func(r chi.Router) {
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogSubmit)).Post("/", h.SubmitHandler)
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogList)).Get("/", h.ListHandler)
	})

	return r
}

// PublicRoutes returns the router for public log view/download endpoints.
// These endpoints do not require authentication.
// Endpoints:
//   - GET /logs/view?game=<name> - HTML view of logs
//   - GET /logs/download?game=<name> - JSON download of logs
func PublicRoutes(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/view", h.ViewHandler)
	r.Get("/download", h.DownloadHandler)

	return r
}
