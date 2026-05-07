package httpx

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gocanto/dot-files/internal/app/service"
)

func (s Server) listOpVaults(w http.ResponseWriter, _ *http.Request) {
	vaults, err := s.Service.ListOpVaults()

	if err != nil {
		writeOpError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"vaults": vaults})
}

func (s Server) listOpItems(w http.ResponseWriter, r *http.Request) {
	vault := r.URL.Query().Get("vault")

	if vault == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("query parameter 'vault' is required"))

		return
	}

	items, err := s.Service.ListOpItems(vault)

	if err != nil {
		writeOpError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func writeOpError(w http.ResponseWriter, err error) {
	var unavailable service.ErrOpUnavailable

	if errors.As(err, &unavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": unavailable.Reason,
			"code":  "op_unavailable",
		})

		return
	}

	writeError(w, http.StatusInternalServerError, err)
}
