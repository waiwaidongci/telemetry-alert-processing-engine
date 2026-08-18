package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/telemetry-alert/internal/application"
)

func tenantFromRequest(r *http.Request) (string, error) {
	if v := r.Header.Get("X-Tenant-ID"); v != "" {
		return v, nil
	}
	if v := r.URL.Query().Get("tenant_id"); v != "" {
		return v, nil
	}
	return "", application.ValidationError{Field: "tenant_id", Message: "X-Tenant-ID header or tenant_id query is required"}
}

func pathTenant(r *http.Request) string {
	return r.PathValue("tenantID")
}

func parseLimitOffset(r *http.Request, def, max int) (int, int) {
	limit := parseIntQuery(r, "limit", def)
	if limit <= 0 || limit > max {
		limit = def
	}
	offset := parseIntQuery(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func parseIntQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func parseTimeQuery(r *http.Request, key string) (time.Time, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, application.ValidationError{Field: key, Message: "must be RFC3339 timestamp"}
	}
	return t.UTC(), nil
}
