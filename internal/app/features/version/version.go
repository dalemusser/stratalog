// internal/app/features/version/version.go
package version

import (
	"net/http"
	"runtime"

	"github.com/dalemusser/stratalog/internal/app/system/handler"
	"github.com/dalemusser/stratalog/internal/app/system/versioninfo"
	"github.com/dalemusser/stratalog/internal/platform/httputil"
)

// Handler keeps a reference to the shared application handler in case we ever
// want config/DB info here. For now, it's not strictly needed.
type Handler struct{ h *handler.Handler }

type response struct {
	Service   string `json:"service"`
	Version   string `json:"version"`
	GitCommit string `json:"git_commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	GoVersion string `json:"go_version"`
}

// Serve handles GET /version and returns build/runtime info as JSON.
func (vh *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	resp := response{
		Service:   versioninfo.Service,
		Version:   versioninfo.Version,
		GitCommit: versioninfo.GitCommit,
		BuildTime: versioninfo.BuildTime,
		GoVersion: runtime.Version(),
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}
