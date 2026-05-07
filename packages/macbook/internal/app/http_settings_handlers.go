package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type validateSettingsRequest struct {
	Settings runtimeSettings `json:"settings"`
}

func (s httpServer) getSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, validateRuntimeSettings(s.app.home, s.app.repo, s.app.settings))
}

func (s httpServer) validateSettings(w http.ResponseWriter, r *http.Request) {
	var req validateSettingsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, validateRuntimeSettings(s.app.home, s.app.repo, req.Settings))
}
