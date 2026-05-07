package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gocanto/mac-os/internal/storage"
)

type savePreferencesRequest struct {
	Theme string `json:"theme"`
}

func (s httpServer) getPreferences(w http.ResponseWriter, r *http.Request) {
	store, closeStore, err := s.app.workflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	prefs, err := store.GetUserPreferences(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("read user preferences: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, prefs)
}

func (s httpServer) savePreferences(w http.ResponseWriter, r *http.Request) {
	var req savePreferencesRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	store, closeStore, err := s.app.workflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	prefs, err := store.SaveUserPreferences(r.Context(), storage.UserPreferences{Theme: req.Theme})

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save user preferences: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, prefs)
}
