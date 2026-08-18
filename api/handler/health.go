package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/example/telemetry-alert/api"
)

type HealthHandler struct {
	db    *sql.DB
	ready func() bool
}

func NewHealthHandler(db *sql.DB, ready func() bool) *HealthHandler {
	return &HealthHandler{db: db, ready: ready}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, _ *http.Request) {
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, _ *http.Request) {
	if !h.ready() {
		api.WriteError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.db.PingContext(ctx); err != nil {
		api.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "database is not reachable")
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
