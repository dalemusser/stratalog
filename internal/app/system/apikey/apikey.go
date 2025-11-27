// internal/app/system/apikey/apikey.go
package apikey

import (
	"net/http"
	"strings"

	"github.com/dalemusser/stratalog/internal/app/system/config"
	"go.uber.org/zap"
)

// RequireIngestKey returns middleware that enforces the configured ingest_api_key.
// It checks, in order:
//   - Authorization: Bearer <key>
//   - X-API-Key: <key>
//   - ?api_key=<key>
func RequireIngestKey(cfg *config.Config) func(http.Handler) http.Handler {
	expected := strings.TrimSpace(cfg.IngestAPIKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expected == "" {
				// Config validation should prevent this in prod, but don't panic at runtime.
				zap.L().Warn("RequireIngestKey used but ingest_api_key is empty")
				http.Error(w, "server misconfigured", http.StatusInternalServerError)
				return
			}

			key, ok := apiKeyFromRequest(r)
			if !ok || key != expected {
				zap.L().Warn("ingest API key unauthorized",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.String("remote_ip", r.RemoteAddr),
				)
				w.Header().Set("WWW-Authenticate", `Bearer realm="stratalog-ingest"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminKey returns middleware that enforces the configured admin_api_key.
// Behavior:
//   - If admin_api_key is set: only that key is accepted.
//   - If admin_api_key is empty: fallback to ingest_api_key.
func RequireAdminKey(cfg *config.Config) func(http.Handler) http.Handler {
	admin := strings.TrimSpace(cfg.AdminAPIKey)
	ingest := strings.TrimSpace(cfg.IngestAPIKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expected := admin
			if expected == "" {
				expected = ingest
			}
			if expected == "" {
				zap.L().Warn("RequireAdminKey used but neither admin_api_key nor ingest_api_key is set")
				http.Error(w, "server misconfigured", http.StatusInternalServerError)
				return
			}

			key, ok := apiKeyFromRequest(r)
			if !ok || key != expected {
				zap.L().Warn("admin API key unauthorized",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
					zap.String("remote_ip", r.RemoteAddr),
				)
				w.Header().Set("WWW-Authenticate", `Bearer realm="stratalog-admin"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// apiKeyFromRequest extracts an API key from the request, checking:
//  1. Authorization: Bearer <token>
//  2. X-API-Key: <token>
//  3. URL query param: api_key=<token>
func apiKeyFromRequest(r *http.Request) (string, bool) {
	// 1) Authorization: Bearer <token>
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[len("Bearer "):])
		if token != "" {
			return token, true
		}
	}

	// 2) X-API-Key header
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key, true
	}

	// 3) api_key query param (handy for browser use)
	if key := strings.TrimSpace(r.URL.Query().Get("api_key")); key != "" {
		return key, true
	}

	return "", false
}
