package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gocanto/dot-files/internal/app/setting"
)

type validateSettingsRequest struct {
	Settings setting.RuntimeSettings `json:"settings"`
}

func (s Server) getSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, setting.ValidateRuntimeSettings(s.Home, s.Repo, s.Settings))
}

func (s Server) validateSettings(w http.ResponseWriter, r *http.Request) {
	var req validateSettingsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, setting.ValidateRuntimeSettings(s.Home, s.Repo, req.Settings))
}
