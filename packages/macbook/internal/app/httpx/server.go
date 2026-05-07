package httpx

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/gocanto/dot-files/internal/app/service"
	"github.com/gocanto/dot-files/internal/app/setting"
	"github.com/gocanto/dot-files/internal/domain"
	"github.com/gocanto/dot-files/internal/storage"
)

type WorkflowStore func(context.Context) (*storage.Store, func(), error)

type ServiceFactory func(setting.RuntimeSettings) service.Service

type WorkflowFactory func(setting.RuntimeSettings) []domain.Workflow

type ServeConfig struct {
	Home      string
	Repo      string
	Stderr    io.Writer
	Service   ServiceFactory
	Workflows WorkflowFactory
}

type Server struct {
	Service       service.Service
	Home          string
	Repo          string
	Settings      setting.RuntimeSettings
	Workflows     func() []domain.Workflow
	WorkflowStore WorkflowStore
}

func Serve(args []string, cfg ServeConfig) int {
	fs := flag.NewFlagSet("serve-http", flag.ContinueOnError)
	fs.SetOutput(cfg.Stderr)

	socketPath := fs.String("socket", "", "Unix socket path")
	repoRoot := fs.String("repo-root", "", "Repository root")
	appsConfigPath := fs.String("apps-config", "", "Apps manifest path")
	secretsConfigPath := fs.String("secrets-config", "", "Secrets manifest path")
	generatedAppsPath := fs.String("generated-apps", "", "Generated apps output path")
	archiveRoot := fs.String("archive-root", "", "Archive root")
	workflowDBPath := fs.String("workflow-db", "", "Workflow SQLite database path")
	opVault := fs.String("op-vault", "", "1Password vault")
	opItem := fs.String("op-item", "", "1Password item")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *socketPath == "" {
		fmt.Fprintln(cfg.Stderr, "missing --socket")

		return 2
	}

	settings := setting.RuntimeSettings{
		RepoRoot:          *repoRoot,
		AppsConfigPath:    *appsConfigPath,
		SecretsConfigPath: *secretsConfigPath,
		GeneratedAppsPath: *generatedAppsPath,
		ArchiveRoot:       *archiveRoot,
		WorkflowDBPath:    *workflowDBPath,
		OPVault:           *opVault,
		OPItem:            *opItem,
	}
	validation := setting.ValidateRuntimeSettings(cfg.Home, cfg.Repo, settings)

	if !validation.Valid {
		for _, check := range validation.Checks {
			if check.Status == setting.CheckError {
				fmt.Fprintf(cfg.Stderr, "invalid settings: %s: %s\n", check.Label, check.Message)
			}
		}

		return 2
	}

	settings = validation.Settings
	repo := settings.RepoRoot

	store, err := storage.Open(context.Background(), settings.WorkflowDBPath)

	if err != nil {
		fmt.Fprintf(cfg.Stderr, "open workflow log database: %v\n", err)

		return 1
	}

	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintf(cfg.Stderr, "close workflow log database: %v\n", err)
		}
	}()

	if err := os.Remove(*socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(cfg.Stderr, "remove stale http socket: %v\n", err)

		return 1
	}

	listener, err := net.Listen("unix", *socketPath)

	if err != nil {
		fmt.Fprintf(cfg.Stderr, "listen on http socket: %v\n", err)

		return 1
	}

	defer func() {
		if err := listener.Close(); err != nil {
			fmt.Fprintf(cfg.Stderr, "close http listener: %v\n", err)
		}

		if err := os.Remove(*socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(cfg.Stderr, "remove http socket: %v\n", err)
		}
	}()

	appServer := Server{
		Service:   cfg.Service(settings),
		Home:      cfg.Home,
		Repo:      repo,
		Settings:  settings,
		Workflows: func() []domain.Workflow { return cfg.Workflows(settings) },
		WorkflowStore: func(context.Context) (*storage.Store, func(), error) {
			return store, func() {}, nil
		},
	}
	server := &http.Server{Handler: NewServerHandler(ServerHandlerConfig{
		Mux:           appServer.BuildMux(),
		SafeQueryKeys: []string{"limit"},
	})}

	if err := RunServer(*socketPath, listener, server); err != nil {
		fmt.Fprintf(cfg.Stderr, "serve http: %v\n", err)

		return 1
	}

	return 0
}

func (s Server) BuildMux() *http.ServeMux {
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

func (s Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
