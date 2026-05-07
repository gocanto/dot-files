package app

import "net/http"

type httpServer struct {
	app app
}

func (s httpServer) buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthz", s.healthz)
	mux.HandleFunc("GET /v1/workflows", s.listWorkflows)
	mux.HandleFunc("POST /v1/workflows/run", s.runWorkflow)
	mux.HandleFunc("GET /v1/template-files", s.listTemplateFiles)
	mux.HandleFunc("GET /v1/template-files/content", s.readTemplateFile)
	mux.HandleFunc("PUT /v1/template-files/content", s.saveTemplateFile)
	mux.HandleFunc("GET /v1/runs", s.listRuns)
	mux.HandleFunc("GET /v1/runs/{id}/log", s.runLog)
	mux.HandleFunc("GET /v1/settings", s.getSettings)
	mux.HandleFunc("POST /v1/settings/validate", s.validateSettings)
	mux.HandleFunc("GET /v1/preferences", s.getPreferences)
	mux.HandleFunc("POST /v1/preferences", s.savePreferences)
	mux.HandleFunc("GET /v1/onepassword/vaults", s.listOpVaults)
	mux.HandleFunc("GET /v1/onepassword/items", s.listOpItems)

	return mux
}

func (s httpServer) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
