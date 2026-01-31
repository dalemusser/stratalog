package logbrowser

import (
	"github.com/dalemusser/stratalog/internal/app/system/auth"
	"github.com/go-chi/chi/v5"
)

// Routes returns the router for the log browser feature.
// All routes require authentication with admin or developer role.
func Routes(h *Handler, sessionMgr *auth.SessionManager) chi.Router {
	r := chi.NewRouter()

	// Require admin or developer role
	r.Use(sessionMgr.RequireRole("admin", "developer"))

	// Main browser page
	r.Get("/", h.ServeList)

	// Playground, documentation, and recent logs
	r.Get("/playground", h.ServePlayground)
	r.Get("/docs", h.ServeDocs)
	r.Get("/recent", h.ServeRecentLogs)
	r.Get("/recent/stream", h.ServeRecentLogsStream)

	// HTMX partials
	r.Get("/players", h.ServePlayers)
	r.Get("/game-picker", h.ServeGamePicker)
	r.Get("/data", h.ServeLogs)

	// Download operations
	r.Get("/download", h.HandleDownloadPlayerLogs)

	// Delete operations (admin only in practice, checked in handler)
	r.Post("/{game}/{id}/delete", h.HandleDeleteLog)
	r.Post("/{game}/player/{playerID}/delete", h.HandleDeletePlayerLogs)

	return r
}
