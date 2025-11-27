// internal/platform/httputil/handlers.go
package httputil

import (
	"net/http"

	"go.uber.org/zap"
)

// NotFoundHandler is a chi-compatible handler for 404 responses.
// It returns a JSON error body and logs at Info level.
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	zap.L().Info("not_found",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_ip", r.RemoteAddr),
	)

	JSONError(w, http.StatusNotFound,
		"not_found",
		"The requested resource was not found",
	)
}

// MethodNotAllowedHandler is a chi-compatible handler for 405 responses.
// It returns a JSON error body and logs at Info level.
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	zap.L().Info("method_not_allowed",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote_ip", r.RemoteAddr),
	)

	JSONError(w, http.StatusMethodNotAllowed,
		"method_not_allowed",
		"The requested HTTP method is not allowed for this resource",
	)
}
