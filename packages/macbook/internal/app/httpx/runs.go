package httpx

import (
	"fmt"
	"net/http"
	"strconv"
)

func (s Server) listRuns(w http.ResponseWriter, r *http.Request) {
	limit := int64(0)

	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)

		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err))

			return
		}

		limit = parsed
	}

	store, closeStore, err := s.WorkflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	runs, err := store.ListRuns(r.Context(), limit)

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("list workflow runs: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s Server) runLog(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	store, closeStore, err := s.WorkflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	log, err := store.RunLog(r.Context(), runID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("read workflow run log: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, log)
}
