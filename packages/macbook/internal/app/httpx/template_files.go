package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type saveTemplateFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (s Server) listTemplateFiles(w http.ResponseWriter, _ *http.Request) {
	files, err := s.Service.ListTemplateFiles()

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s Server) readTemplateFile(w http.ResponseWriter, r *http.Request) {
	content, err := s.Service.ReadTemplateFile(r.URL.Query().Get("path"))

	if err != nil {
		writeError(w, http.StatusBadRequest, err)

		return
	}

	writeJSON(w, http.StatusOK, content)
}

func (s Server) saveTemplateFile(w http.ResponseWriter, r *http.Request) {
	var req saveTemplateFileRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	content, err := s.Service.SaveTemplateFile(req.Path, req.Content)

	if err != nil {
		writeError(w, http.StatusBadRequest, err)

		return
	}

	writeJSON(w, http.StatusOK, content)
}
