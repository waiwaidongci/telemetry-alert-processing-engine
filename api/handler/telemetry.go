package handler

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/internal/application"
	deviceapp "github.com/example/telemetry-alert/internal/application/device"
	telemetryapp "github.com/example/telemetry-alert/internal/application/telemetry"
	devicedomain "github.com/example/telemetry-alert/internal/domain/device"
	telemetrydomain "github.com/example/telemetry-alert/internal/domain/telemetry"
)

type TelemetryHandler struct {
	service *telemetryapp.Service
	devices *deviceapp.Service
	metrics *api.Metrics
}

func NewTelemetryHandler(service *telemetryapp.Service, devices *deviceapp.Service, metrics *api.Metrics) *TelemetryHandler {
	return &TelemetryHandler{service: service, devices: devices, metrics: metrics}
}

type telemetryPointRequest struct {
	DeviceID       string            `json:"device_id,omitempty"`
	MetricName     string            `json:"metric_name"`
	Value          float64           `json:"value"`
	Timestamp      string            `json:"timestamp"`
	Labels         map[string]string `json:"labels,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

type batchRequest struct {
	Points []telemetryPointRequest `json:"points"`
}

func (h *TelemetryHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var req telemetryPointRequest
	if err := decodeLimitedJSON(r, &req, 1<<20); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	d, err := h.authenticate(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if req.DeviceID != "" && req.DeviceID != d.ID {
		writeError(w, application.ErrForbidden)
		return
	}
	in, err := toInput(req)
	if err != nil {
		writeError(w, err)
		return
	}
	point, err := h.service.IngestSingle(r.Context(), d.TenantID, d.ID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	h.metrics.AddIngest(1)
	api.WriteJSON(w, http.StatusCreated, point)
}

func (h *TelemetryHandler) IngestBatch(w http.ResponseWriter, r *http.Request) {
	var req batchRequest
	if err := decodeLimitedJSON(r, &req, 8<<20); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	d, err := h.authenticate(r)
	if err != nil {
		writeError(w, err)
		return
	}
	inputs := make([]telemetryapp.IngestInput, 0, len(req.Points))
	for _, p := range req.Points {
		if p.DeviceID != "" && p.DeviceID != d.ID {
			writeError(w, application.ErrForbidden)
			return
		}
		in, err := toInput(p)
		if err != nil {
			writeError(w, err)
			return
		}
		inputs = append(inputs, in)
	}
	points, err := h.service.IngestBatch(r.Context(), d.TenantID, d.ID, inputs)
	if err != nil {
		writeError(w, err)
		return
	}
	h.metrics.AddIngest(len(points))
	api.WriteJSON(w, http.StatusCreated, map[string]any{"inserted": len(points), "points": points})
}

func (h *TelemetryHandler) QueryRaw(w http.ResponseWriter, r *http.Request) {
	start, err := parseTimeQuery(r, "start")
	if err != nil {
		writeError(w, err)
		return
	}
	end, err := parseTimeQuery(r, "end")
	if err != nil {
		writeError(w, err)
		return
	}
	limit, offset := parseLimitOffset(r, 200, 2000)
	q := telemetryapp.RawQuery{
		TenantID:   pathTenant(r),
		DeviceID:   r.URL.Query().Get("device_id"),
		MetricName: r.URL.Query().Get("metric_name"),
		Start:      start,
		End:        end,
		Limit:      limit,
		Offset:     offset,
	}
	points, err := h.service.QueryRaw(r.Context(), q)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": points, "meta": api.PageMeta{Limit: limit, Offset: offset, Count: len(points)}})
}

func (h *TelemetryHandler) QueryAggregates(w http.ResponseWriter, r *http.Request) {
	start, err := parseTimeQuery(r, "start")
	if err != nil {
		writeError(w, err)
		return
	}
	end, err := parseTimeQuery(r, "end")
	if err != nil {
		writeError(w, err)
		return
	}
	limit, offset := parseLimitOffset(r, 200, 2000)
	q := telemetryapp.AggregateQuery{
		TenantID:    pathTenant(r),
		DeviceID:    r.URL.Query().Get("device_id"),
		MetricName:  r.URL.Query().Get("metric_name"),
		Granularity: telemetrydomain.Granularity(r.URL.Query().Get("granularity")),
		Start:       start,
		End:         end,
		Limit:       limit,
		Offset:      offset,
	}
	buckets, err := h.service.QueryAggregates(r.Context(), q)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": buckets, "meta": api.PageMeta{Limit: limit, Offset: offset, Count: len(buckets)}})
}

func (h *TelemetryHandler) authenticate(r *http.Request) (devicedomain.Device, error) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" || token == r.Header.Get("Authorization") {
		token = r.Header.Get("X-Device-Token")
	}
	return h.devices.Authenticate(r.Context(), token)
}

func decodeLimitedJSON(r *http.Request, dst any, limit int64) error {
	body := io.LimitReader(r.Body, limit)
	var reader io.Reader = body
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(body)
		if err != nil {
			return application.ValidationError{Field: "body", Message: "invalid gzip body"}
		}
		defer gz.Close()
		reader = gz
	}
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return application.ValidationError{Field: "body", Message: "invalid JSON body"}
	}
	return nil
}

func toInput(req telemetryPointRequest) (telemetryapp.IngestInput, error) {
	t, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		return telemetryapp.IngestInput{}, application.ValidationError{Field: "timestamp", Message: "must be RFC3339 timestamp"}
	}
	return telemetryapp.IngestInput{
		DeviceID:       req.DeviceID,
		MetricName:     req.MetricName,
		Value:          req.Value,
		Timestamp:      t,
		Labels:         req.Labels,
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}
