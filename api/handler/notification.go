package handler

import (
	"net/http"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/internal/application/notification"
)

type NotificationHandler struct {
	dispatcher *notification.Dispatcher
}

func NewNotificationHandler(dispatcher *notification.Dispatcher) *NotificationHandler {
	return &NotificationHandler{dispatcher: dispatcher}
}

func (h *NotificationHandler) ListFailures(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 100, 200)
	list, err := h.dispatcher.ListFailures(r.Context(), pathTenant(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": list, "meta": api.PageMeta{Limit: limit, Offset: offset, Count: len(list)}})
}
