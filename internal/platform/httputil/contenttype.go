// internal/platform/httputil/contenttype.go
package httputil

import (
	"net/http"
	"strings"
)

// RequireJSON enforces that requests have a JSON Content-Type,
// typically "application/json" or something ending in "+json",
// e.g. "application/problem+json".
//
// If the Content-Type is missing or not JSON, it returns 415 Unsupported Media Type.
func RequireJSON() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ct := strings.TrimSpace(r.Header.Get("Content-Type"))
			if ct == "" {
				JSONError(w, http.StatusUnsupportedMediaType,
					"unsupported_media_type",
					"Content-Type must be application/json",
				)
				return
			}

			// Strip any parameters, e.g. "; charset=utf-8"
			if idx := strings.Index(ct, ";"); idx != -1 {
				ct = ct[:idx]
			}
			ct = strings.ToLower(strings.TrimSpace(ct))

			if ct != "application/json" && !strings.HasSuffix(ct, "+json") {
				JSONError(w, http.StatusUnsupportedMediaType,
					"unsupported_media_type",
					"Content-Type must be application/json",
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
