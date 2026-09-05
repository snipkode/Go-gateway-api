package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

// Recovery converts panics into 500 responses and logs the stack, keeping the
// API process alive.
func Recovery(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, start: time.Now()}
		defer func() {
			if p := recover(); p != nil {
				logger.Error("panic recovered",
					slog.String("request_id", httpctx.RequestID(r.Context())),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())),
				)
				httpctx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
				return
			}
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			logger.Info("http request",
				slog.String("request_id", httpctx.RequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int("bytes", rec.bytes),
				slog.Duration("latency", time.Since(rec.start)),
			)
		}()
		next.ServeHTTP(rec, r)
	})
}

var _ = errors.New
