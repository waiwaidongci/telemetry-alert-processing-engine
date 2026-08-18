package router

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/api/handler"
	"github.com/example/telemetry-alert/api/middleware"
	alertruleapp "github.com/example/telemetry-alert/internal/application/alertrule"
	deviceapp "github.com/example/telemetry-alert/internal/application/device"
	eventapp "github.com/example/telemetry-alert/internal/application/event"
	notificationapp "github.com/example/telemetry-alert/internal/application/notification"
	telemetryapp "github.com/example/telemetry-alert/internal/application/telemetry"
	tenantapp "github.com/example/telemetry-alert/internal/application/tenant"
)

type Dependencies struct {
	DB            *sql.DB
	Logger        *slog.Logger
	Metrics       *api.Metrics
	Ready         func() bool
	Tenants       *tenantapp.Service
	Devices       *deviceapp.Service
	Telemetry     *telemetryapp.Service
	Rules         *alertruleapp.Service
	Events        *eventapp.Service
	Notifications *notificationapp.Dispatcher
}

// New builds the full HTTP route table wrapped by common middleware.
func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	tenantHandler := handler.NewTenantHandler(deps.Tenants)
	deviceHandler := handler.NewDeviceHandler(deps.Devices)
	telemetryHandler := handler.NewTelemetryHandler(deps.Telemetry, deps.Devices, deps.Metrics)
	ruleHandler := handler.NewAlertRuleHandler(deps.Rules)
	eventHandler := handler.NewEventHandler(deps.Events)
	notificationHandler := handler.NewNotificationHandler(deps.Notifications)
	healthHandler := handler.NewHealthHandler(deps.DB, deps.Ready)

	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.HandleFunc("GET /readyz", healthHandler.Readyz)
	mux.Handle("GET /metrics", deps.Metrics.Handler())

	mux.HandleFunc("POST /v1/tenants", tenantHandler.Create)
	mux.HandleFunc("GET /v1/tenants", tenantHandler.List)
	mux.HandleFunc("GET /v1/tenants/{id}", tenantHandler.Get)
	mux.HandleFunc("PUT /v1/tenants/{id}", tenantHandler.Update)

	mux.HandleFunc("POST /v1/tenants/{tenantID}/devices", deviceHandler.Create)
	mux.HandleFunc("GET /v1/tenants/{tenantID}/devices", deviceHandler.List)
	mux.HandleFunc("GET /v1/tenants/{tenantID}/devices/{id}", deviceHandler.Get)
	mux.HandleFunc("PUT /v1/tenants/{tenantID}/devices/{id}", deviceHandler.Update)
	mux.HandleFunc("PATCH /v1/tenants/{tenantID}/devices/{id}/status", deviceHandler.SetStatus)
	mux.HandleFunc("DELETE /v1/tenants/{tenantID}/devices/{id}", deviceHandler.Delete)

	mux.HandleFunc("POST /v1/ingest", telemetryHandler.Ingest)
	mux.HandleFunc("POST /v1/ingest/batch", telemetryHandler.IngestBatch)
	mux.HandleFunc("GET /v1/tenants/{tenantID}/telemetry/raw", telemetryHandler.QueryRaw)
	mux.HandleFunc("GET /v1/tenants/{tenantID}/telemetry/aggregates", telemetryHandler.QueryAggregates)

	mux.HandleFunc("POST /v1/tenants/{tenantID}/rules", ruleHandler.Create)
	mux.HandleFunc("GET /v1/tenants/{tenantID}/rules", ruleHandler.List)
	mux.HandleFunc("GET /v1/tenants/{tenantID}/rules/{id}", ruleHandler.Get)
	mux.HandleFunc("PUT /v1/tenants/{tenantID}/rules/{id}", ruleHandler.Update)
	mux.HandleFunc("PATCH /v1/tenants/{tenantID}/rules/{id}/enabled", ruleHandler.SetEnabled)
	mux.HandleFunc("DELETE /v1/tenants/{tenantID}/rules/{id}", ruleHandler.Delete)

	mux.HandleFunc("GET /v1/tenants/{tenantID}/events", eventHandler.List)
	mux.HandleFunc("GET /v1/tenants/{tenantID}/notification-failures", notificationHandler.ListFailures)

	var h http.Handler = mux
	h = middleware.Timeout(30 * time.Second)(h)
	h = middleware.RequestLog(deps.Logger, deps.Metrics)(h)
	h = middleware.Recover(deps.Logger)(h)
	h = middleware.TraceID(h)
	return h
}
