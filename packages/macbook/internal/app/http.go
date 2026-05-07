package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gocanto/dot-files/internal/app/setting"
	"github.com/gocanto/dot-files/internal/httpx"
	"github.com/gocanto/dot-files/internal/storage"
)

func (a app) serveHTTP(args []string) int {
	fs := flag.NewFlagSet("serve-http", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

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
		fmt.Fprintln(a.stderr, "missing --socket")

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
	validation := setting.ValidateRuntimeSettings(a.home, a.repo, settings)

	if !validation.Valid {
		for _, check := range validation.Checks {
			if check.Status == setting.CheckError {
				fmt.Fprintf(a.stderr, "invalid settings: %s: %s\n", check.Label, check.Message)
			}
		}

		return 2
	}

	a.settings = validation.Settings
	a.repo = validation.Settings.RepoRoot

	store, err := storage.Open(context.Background(), a.settings.WorkflowDBPath)

	if err != nil {
		fmt.Fprintf(a.stderr, "open workflow log database: %v\n", err)

		return 1
	}

	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintf(a.stderr, "close workflow log database: %v\n", err)
		}
	}()

	a.store = store

	if err := os.Remove(*socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(a.stderr, "remove stale http socket: %v\n", err)

		return 1
	}

	listener, err := net.Listen("unix", *socketPath)

	if err != nil {
		fmt.Fprintf(a.stderr, "listen on http socket: %v\n", err)

		return 1
	}

	defer func() {
		if err := listener.Close(); err != nil {
			fmt.Fprintf(a.stderr, "close http listener: %v\n", err)
		}

		if err := os.Remove(*socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(a.stderr, "remove http socket: %v\n", err)
		}
	}()

	server := &http.Server{Handler: httpx.NewServerHandler(httpx.ServerHandlerConfig{
		Mux:           httpServer{app: a}.buildMux(),
		SafeQueryKeys: []string{"limit"},
	})}

	if err := httpx.RunServer(*socketPath, listener, server); err != nil {
		fmt.Fprintf(a.stderr, "serve http: %v\n", err)

		return 1
	}

	return 0
}
