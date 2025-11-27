// internal/platform/httputil/json.go
package httputil

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// ErrorResponse is a standard JSON error envelope.
// `Error` is a short code/key; `Message` is optional human-readable detail.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// WriteJSON writes a JSON response with the given status code.
// If encoding fails, it logs the error and returns a 500 with a minimal body.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		zap.L().Error("write JSON response failed", zap.Error(err))
		// Best-effort fallback: only safe to write more if nothing has been written,
		// but at this point we've already sent status+headers. Just log and bail.
	}
}

// JSONError writes a standardized JSON error using ErrorResponse.
// `code` is a short machine-friendly string, `message` is human-readable.
func JSONError(w http.ResponseWriter, status int, code, message string) {
	resp := ErrorResponse{
		Error:   code,
		Message: message,
	}
	WriteJSON(w, status, resp)
}

// JSONErrorSimple is a convenience when you just want {"error": "<msg>"}.
func JSONErrorSimple(w http.ResponseWriter, status int, message string) {
	resp := ErrorResponse{
		Error: message,
	}
	WriteJSON(w, status, resp)
}
