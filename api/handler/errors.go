package handler

import (
	"net/http"

	"github.com/example/telemetry-alert/api"
)

func writeError(w http.ResponseWriter, err error) {
	api.WriteErrorFrom(w, err)
}

func decodeJSON(r *http.Request, dst any) error {
	return api.DecodeJSON(r, dst)
}
