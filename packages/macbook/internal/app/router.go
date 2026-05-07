package app

import (
	"net/http"

	"github.com/gocanto/dot-files/internal/app/httpx"
)

type httpServer struct {
	app app
}

func (s httpServer) buildMux() *http.ServeMux {
	return s.app.handlerServer().BuildMux()
}

func (a app) handlerServer() httpx.Server {
	return httpx.Server{
		Service:       a.service(),
		Home:          a.home,
		Repo:          a.repo,
		Settings:      a.settings,
		Workflows:     a.workflows,
		WorkflowStore: a.workflowStore,
	}
}
