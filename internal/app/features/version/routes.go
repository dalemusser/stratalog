// internal/app/features/version/routes.go
package version

import (
	"github.com/dalemusser/stratalog/internal/app/system/handler"
	"github.com/go-chi/chi/v5"
)

// MountRoutes registers GET /version.
func MountRoutes(r chi.Router, h *handler.Handler) {
	vh := &Handler{h: h}
	r.Get("/version", vh.Serve)
}
