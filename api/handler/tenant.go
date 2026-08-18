package handler

import (
	"net/http"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/internal/application/tenant"
)

type TenantHandler struct {
	service *tenant.Service
}

func NewTenantHandler(service *tenant.Service) *TenantHandler {
	return &TenantHandler{service: service}
}

type tenantRequest struct {
	Name string `json:"name"`
}

func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req tenantRequest
	if err := decodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	t, err := h.service.Create(r.Context(), req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, t)
}

func (h *TenantHandler) Get(w http.ResponseWriter, r *http.Request) {
	t, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, t)
}

func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 100, 200)
	list, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"items": list, "meta": api.PageMeta{Limit: limit, Offset: offset, Count: len(list)}})
}

func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req tenantRequest
	if err := decodeJSON(r, &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid_input", err.Error())
		return
	}
	t, err := h.service.Update(r.Context(), r.PathValue("id"), req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, t)
}
