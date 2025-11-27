// internal/platform/httputil/limit.go
package httputil

import "net/http"

// LimitBodySize wraps r.Body with http.MaxBytesReader so that handlers cannot
// read more than maxBytes from the request body.
//
// If a handler attempts to read more than maxBytes, reads will fail and JSON
// decoders, etc., will return an error (often "http: request body too large").
//
// Handlers can optionally detect *http.MaxBytesError if they want to return a
// specific 413 response; otherwise they will just see a generic error.
func LimitBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If maxBytes <= 0, don't enforce a limit (no-op).
			if maxBytes > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
