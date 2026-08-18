package handler

import (
	"net/http"
	"time"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/internal/application"
	"github.com/example/telemetry-alert/internal/application/alertrule"
	"github.com/example/telemetry-alert/internal/domain/alert"
)

type AlertRuleHandler struct {
	service *alertrule.Service
}

func NewAlertRuleHandler(service *alertrule.Service) *AlertRuleHandler {
	return &AlertRuleHandler{service: service}
}

type alertRuleRequest struct {
	Name       string   `json:"name"`
	DeviceID   string   `json:"device_id,omitempty"`
	DeviceTag  string   `json:"device_tag,omitempty"`
	MetricName string   `json:"metric_name"`
	Operator   string   `json:"operator"`
	Threshold  float64  `json:"threshold"`
	Window     string   `json:"window"`
	Duration   string   `json:"duration"`
	Level      string   `json:"level"`
	Cooldown   string   `json:"cooldown"`
	Channels   []string `json:"channels"`
}

func (h *AlertRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	in, err := parseRuleRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rule, err := h.service.Create(r.Context(), pathTenant(r), in)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, rule)
}

func (h *AlertRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	rule, err := h.service.Get(r.Context(), pathTenant(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, rule)
}

func (h *AlertRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 100, 200)
	list, err := h.service.List(r.Context(), pathTenant(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": list, "meta": api.PageMeta{Limit: limit, Offset: offset, Count: len(list)}})
}

func (h *AlertRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	in, err := parseRuleRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rule, err := h.service.Update(r.Context(), pathTenant(r), r.PathValue("id"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, rule)
}

func (h *AlertRuleHandler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	rule, err := h.service.SetEnabled(r.Context(), pathTenant(r), r.PathValue("id"), req.Enabled)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, rule)
}

func (h *AlertRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), pathTenant(r), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseRuleRequest(r *http.Request) (alertrule.CreateInput, error) {
	var req alertRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		return alertrule.CreateInput{}, err
	}
	window, err := parseDuration(req.Window, "window")
	if err != nil {
		return alertrule.CreateInput{}, err
	}
	duration, err := parseOptionalDuration(req.Duration, "duration")
	if err != nil {
		return alertrule.CreateInput{}, err
	}
	cooldown, err := parseOptionalDuration(req.Cooldown, "cooldown")
	if err != nil {
		return alertrule.CreateInput{}, err
	}
	return alertrule.CreateInput{
		Name:       req.Name,
		DeviceID:   req.DeviceID,
		DeviceTag:  req.DeviceTag,
		MetricName: req.MetricName,
		Operator:   alert.Operator(req.Operator),
		Threshold:  req.Threshold,
		Window:     window,
		Duration:   duration,
		Level:      alert.Level(req.Level),
		Cooldown:   cooldown,
		Channels:   req.Channels,
	}, nil
}

func parseDuration(value, field string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return 0, application.ValidationError{Field: field, Message: "must be a positive duration such as 5m"}
	}
	return d, nil
}

func parseOptionalDuration(value, field string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil || d < 0 {
		return 0, application.ValidationError{Field: field, Message: "must be a non-negative duration such as 30s"}
	}
	return d, nil
}
