package app

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s httpServer) listTemplateFiles(w http.ResponseWriter, _ *http.Request) {
	files, err := s.app.listTemplateFiles()

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s httpServer) readTemplateFile(w http.ResponseWriter, r *http.Request) {
	content, err := s.app.readTemplateFile(r.URL.Query().Get("path"))

	if err != nil {
		writeError(w, http.StatusBadRequest, err)

		return
	}

	writeJSON(w, http.StatusOK, content)
}

func (s httpServer) saveTemplateFile(w http.ResponseWriter, r *http.Request) {
	var req saveTemplateFileRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	content, err := s.app.saveTemplateFile(req.Path, req.Content)

	if err != nil {
		writeError(w, http.StatusBadRequest, err)

		return
	}

	writeJSON(w, http.StatusOK, content)
}
