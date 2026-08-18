package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/example/telemetry-alert/internal/application"
)

type errorBody struct {
	Error   string `json:"error"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type pageMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

// WriteJSON writes an arbitrary value as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	writeJSON(w, status, value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	body := errorBody{Error: "internal_error", Message: "unexpected server error"}
	var ve application.ValidationError
	switch {
	case errors.Is(err, application.ErrNotFound):
		status = http.StatusNotFound
		body = errorBody{Error: "not_found", Message: err.Error()}
	case errors.Is(err, application.ErrConflict):
		status = http.StatusConflict
		body = errorBody{Error: "conflict", Message: err.Error()}
	case errors.Is(err, application.ErrInvalidInput):
		status = http.StatusBadRequest
		body = errorBody{Error: "invalid_input", Message: err.Error()}
	case errors.Is(err, application.ErrUnauthorized):
		status = http.StatusUnauthorized
		body = errorBody{Error: "unauthorized", Message: "invalid or missing access token"}
	case errors.Is(err, application.ErrForbidden):
		status = http.StatusForbidden
		body = errorBody{Error: "forbidden", Message: err.Error()}
	case errors.Is(err, application.ErrRateLimited):
		status = http.StatusTooManyRequests
		body = errorBody{Error: "rate_limited", Message: err.Error()}
	case errors.Is(err, application.ErrDeviceInactive):
		status = http.StatusForbidden
		body = errorBody{Error: "device_inactive", Message: err.Error()}
	case errors.As(err, &ve):
		status = http.StatusBadRequest
		body = errorBody{Error: "invalid_input", Field: ve.Field, Message: ve.Message}
	}
	writeJSON(w, status, body)
}

// WriteErrorFrom maps a domain/service error to an HTTP response.
func WriteErrorFrom(w http.ResponseWriter, err error) {
	writeError(w, err)
}

// WriteError writes a standard JSON error response.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: code, Message: message})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return application.ValidationError{Field: "body", Message: "invalid JSON body"}
	}
	return nil
}

// DecodeJSON decodes a JSON request body while rejecting unknown fields.
func DecodeJSON(r *http.Request, dst any) error {
	return decodeJSON(r, dst)
}

// WriteError is an alias for the standard JSON error response helper.
type PageMeta = pageMeta
