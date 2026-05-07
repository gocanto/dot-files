package app

import (
	"errors"
	"fmt"
	"net/http"
)

func (s httpServer) listOpVaults(w http.ResponseWriter, _ *http.Request) {
	vaults, err := s.app.listOpVaults()

	if err != nil {
		writeOpError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"vaults": vaults})
}

func (s httpServer) listOpItems(w http.ResponseWriter, r *http.Request) {
	vault := r.URL.Query().Get("vault")

	if vault == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("query parameter 'vault' is required"))

		return
	}

	items, err := s.app.listOpItems(vault)

	if err != nil {
		writeOpError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func writeOpError(w http.ResponseWriter, err error) {
	var unavailable ErrOpUnavailable

	if errors.As(err, &unavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": unavailable.Reason,
			"code":  "op_unavailable",
		})

		return
	}

	writeError(w, http.StatusInternalServerError, err)
}
