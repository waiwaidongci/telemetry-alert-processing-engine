package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/internal/logging"
	"github.com/example/telemetry-alert/internal/system"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// TraceID injects a trace and request identifier into the request context.
func TraceID(next http.Handler) http.Handler {
	gen := system.UUIDGenerator{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = gen.New()
		}
		requestID := gen.New()
		ctx := logging.WithTraceID(r.Context(), traceID)
		ctx = logging.WithRequestID(ctx, requestID)
		w.Header().Set("X-Trace-ID", traceID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestLog emits one structured log line per request.
func RequestLog(logger *slog.Logger, metrics *api.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			metrics.Request()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			if sw.status >= 400 {
				metrics.Error()
			}
			logging.WithContext(r.Context(), logger).Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
			)
		})
	}
}

// Recover converts panics into a 500 response and preserves the stack trace.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logging.WithContext(r.Context(), logger).Error("panic recovered",
						"panic", rec,
						"stack", string(debug.Stack()),
					)
					api.WriteError(w, http.StatusInternalServerError, "internal_error", "unexpected server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout applies a request deadline without changing response semantics on
// normal completion.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
