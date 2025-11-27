// internal/app/system/server/mw_recover.go
package server

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// zapRecoverer is a panic-recover middleware that logs panics and stack traces
// using zap, and returns a 500 to the client.
func zapRecoverer(l *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Normalize panic value into an error
					var err error
					switch v := rec.(type) {
					case error:
						err = v
					default:
						err = fmt.Errorf("%v", v)
					}

					// Capture stack trace
					stack := debug.Stack()

					l.Error("panic in HTTP handler",
						zap.Error(err),
						zap.Any("panic_value", rec),
						zap.ByteString("stacktrace", stack),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("remote_ip", r.RemoteAddr),
						zap.String("request_id", middleware.GetReqID(r.Context())),
					)

					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
