package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const (
	traceIDKey   contextKey = "trace_id"
	requestIDKey contextKey = "request_id"
	tenantIDKey  contextKey = "tenant_id"
	deviceIDKey  contextKey = "device_id"
)

// New builds a JSON structured logger for the configured level.
func New(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

// WithTraceID adds a trace identifier to the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// WithRequestID adds a request identifier to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// WithTenantID adds a tenant identifier to the context.
func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

// WithDeviceID adds a device identifier to the context.
func WithDeviceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, deviceIDKey, id)
}

// WithContext returns an *slog.Logger carrying the known context fields.
func WithContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	attrs := make([]slog.Attr, 0, 4)
	if v, ok := ctx.Value(traceIDKey).(string); ok && v != "" {
		attrs = append(attrs, slog.String(string(traceIDKey), v))
	}
	if v, ok := ctx.Value(requestIDKey).(string); ok && v != "" {
		attrs = append(attrs, slog.String(string(requestIDKey), v))
	}
	if v, ok := ctx.Value(tenantIDKey).(string); ok && v != "" {
		attrs = append(attrs, slog.String(string(tenantIDKey), v))
	}
	if v, ok := ctx.Value(deviceIDKey).(string); ok && v != "" {
		attrs = append(attrs, slog.String(string(deviceIDKey), v))
	}
	if len(attrs) == 0 {
		return base
	}
	return base.With(attrsToAny(attrs)...)
}

func attrsToAny(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i := range attrs {
		out[i] = attrs[i]
	}
	return out
}
