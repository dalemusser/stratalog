package logapi

import (
	"net/http"

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
//   - GET /api/log/list - List log entries with filters (only when
//     readEnabled; otherwise it answers 410, see Handler.DisabledHandler)
func Routes(h *Handler, statsRecorder *apistats.Recorder, ledgerConfig ledger.Config, apiKeys []string, restricted auth.RestrictedKey, readEnabled bool, logger *zap.Logger) chi.Router {
	r := chi.NewRouter()

	// Ledger middleware for error logging
	r.Use(ledger.Middleware(ledgerConfig))

	// API key authentication middleware
	r.Use(auth.APIKeyAuthRestricted(apiKeys, restricted, logger))

	// Submit endpoint
	r.Route("/submit", func(r chi.Router) {
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogSubmit)).Post("/", h.SubmitHandler)
	})

	// List endpoint
	r.Route("/list", func(r chi.Router) {
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogList)).Get("/", h.listOrDisabled(readEnabled))
	})

	return r
}

// LegacyRoutes returns the router for the legacy /logs endpoint.
// This maintains backward compatibility with existing clients.
// Endpoints:
//   - POST /logs - Submit single or batch log entries
//   - GET /logs - List log entries with filters
func LegacyRoutes(h *Handler, statsRecorder *apistats.Recorder, ledgerConfig ledger.Config, apiKeys []string, restricted auth.RestrictedKey, readEnabled bool, logger *zap.Logger) chi.Router {
	r := chi.NewRouter()

	// Ledger middleware for error logging
	r.Use(ledger.Middleware(ledgerConfig))

	// API key authentication middleware
	r.Use(auth.APIKeyAuthRestricted(apiKeys, restricted, logger))

	// API stats recording
	r.Route("/", func(r chi.Router) {
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogSubmit)).Post("/", h.SubmitHandler)
		r.With(apistats.MiddlewareWithRecorder(statsRecorder, apistatsstore.StatTypeLogList)).Get("/", h.listOrDisabled(readEnabled))
	})

	return r
}

// listOrDisabled is the list handler while the log read routes are on, and
// the 410 answer while they are off.
func (h *Handler) listOrDisabled(readEnabled bool) http.HandlerFunc {
	if readEnabled {
		return h.ListHandler
	}
	return h.DisabledHandler
}

// ListOrDisabled is listOrDisabled for routes assembled outside this package.
func (h *Handler) ListOrDisabled(readEnabled bool) http.HandlerFunc {
	return h.listOrDisabled(readEnabled)
}
