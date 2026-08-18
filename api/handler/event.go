package handler

import (
	"net/http"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/internal/application/event"
	"github.com/example/telemetry-alert/internal/domain/alert"
)

type EventHandler struct {
	service *event.Service
}

func NewEventHandler(service *event.Service) *EventHandler {
	return &EventHandler{service: service}
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
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
	limit, offset := parseLimitOffset(r, 100, 500)
	list, err := h.service.List(r.Context(), pathTenant(r), event.Filter{
		DeviceID:  r.URL.Query().Get("device_id"),
		RuleID:    r.URL.Query().Get("rule_id"),
		EventType: alert.EventType(r.URL.Query().Get("event_type")),
		Start:     start,
		End:       end,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": list, "meta": api.PageMeta{Limit: limit, Offset: offset, Count: len(list)}})
}
