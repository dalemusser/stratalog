// internal/app/system/server/mw_zaplogger.go
package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func zapRequestLogger(l *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			latency := time.Since(start)

			// Derive scheme as best we can.
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			} else if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
				scheme = xf
			}

			// Route pattern (e.g. "/health", "/metrics", "/debug/pprof/{name}")
			route := ""
			if rctx := chi.RouteContext(r.Context()); rctx != nil {
				route = rctx.RoutePattern()
			}

			l.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("route", route),
				zap.String("host", r.Host),
				zap.String("scheme", scheme),
				zap.String("proto", r.Proto),
				zap.Int("status", ww.Status()),
				zap.Int("bytes", ww.BytesWritten()),
				zap.String("remote_ip", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("referer", r.Referer()),
				zap.Duration("latency", latency),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
