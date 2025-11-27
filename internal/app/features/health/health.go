// internal/app/features/health/health.go
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/dalemusser/stratalog/internal/app/system/handler"
	"github.com/dalemusser/stratalog/internal/platform/httputil"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// Handler keeps a reference to the shared application handler.
type Handler struct{ h *handler.Handler }

// Serve handles GET /health
//
// Success (200):
//
//	{ "status":"ok", "database":"connected" }
//
// Failure (503):
//
//	{ "error":"database_unavailable", "message":"Database unavailable" }
func (hh *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := hh.h.Client.Ping(ctx, readpref.Primary()); err != nil {
		zap.L().Error("health-check: mongo ping failed", zap.Error(err))
		httputil.JSONError(w, http.StatusServiceUnavailable,
			"database_unavailable",
			"Database unavailable",
		)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"database": "connected",
	})
}
