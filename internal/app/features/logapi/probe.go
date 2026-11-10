package logapi

import "net/http"

// ProbeHandler answers the play page's reachability probe (POST /api/log/probe). The
// page sends it exactly as the game sends its own requests (POST, a JSON body,
// an Authorization header, so the browser precedes it with an OPTIONS
// preflight), because a school's web filter can let a plain GET through and
// still block those. A readable {"ok":true} from this service is the proof the
// request got here; a filter's own answer cannot produce it. Nothing is read,
// checked or stored: the key is not validated (the probe must not depend on it
// and must not show up as a refused request), and the body is ignored.
func ProbeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true,"service":"stratalog"}`))
}
