package handler

import (
	"net/http"

	"github.com/example/telemetry-alert/api"
	deviceapp "github.com/example/telemetry-alert/internal/application/device"
	devicedomain "github.com/example/telemetry-alert/internal/domain/device"
)

type DeviceHandler struct {
	service *deviceapp.Service
}

func NewDeviceHandler(service *deviceapp.Service) *DeviceHandler {
	return &DeviceHandler{service: service}
}

type deviceRequest struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	SerialNumber string            `json:"serial_number"`
	Location     string            `json:"location"`
	Tags         map[string]string `json:"tags,omitempty"`
}

type createdDeviceResponse struct {
	Device devicedomain.Device `json:"device"`
	Token  string              `json:"token"`
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req deviceRequest
	if err := decodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	created, err := h.service.Create(r.Context(), pathTenant(r), deviceapp.CreateInput{
		Name: req.Name, Type: req.Type, SerialNumber: req.SerialNumber, Location: req.Location, Tags: req.Tags,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, createdDeviceResponse{Device: created.Device, Token: created.Token})
}

func (h *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	d, err := h.service.Get(r.Context(), pathTenant(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, d)
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 100, 200)
	list, err := h.service.List(r.Context(), pathTenant(r), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": list, "meta": api.PageMeta{Limit: limit, Offset: offset, Count: len(list)}})
}

func (h *DeviceHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req deviceRequest
	if err := decodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	d, err := h.service.Update(r.Context(), pathTenant(r), r.PathValue("id"), deviceapp.CreateInput{
		Name: req.Name, Type: req.Type, SerialNumber: req.SerialNumber, Location: req.Location, Tags: req.Tags,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, d)
}

func (h *DeviceHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status devicedomain.Status `json:"status"`
	}
	if err := decodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	d, err := h.service.SetStatus(r.Context(), pathTenant(r), r.PathValue("id"), req.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, d)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), pathTenant(r), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
