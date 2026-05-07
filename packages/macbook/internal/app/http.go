package app

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
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

	settings := runtimeSettings{
		RepoRoot:          *repoRoot,
		AppsConfigPath:    *appsConfigPath,
		SecretsConfigPath: *secretsConfigPath,
		GeneratedAppsPath: *generatedAppsPath,
		ArchiveRoot:       *archiveRoot,
		WorkflowDBPath:    *workflowDBPath,
		OPVault:           *opVault,
		OPItem:            *opItem,
	}
	validation := validateRuntimeSettings(a.home, a.repo, settings)

	if !validation.Valid {
		for _, check := range validation.Checks {
			if check.Status == checkError {
				fmt.Fprintf(a.stderr, "invalid settings: %s: %s\n", check.Label, check.Message)
			}
		}

		return 2
	}

	a.settings = validation.Settings
	a.repo = validation.Settings.RepoRoot

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
		_ = listener.Close()
		_ = os.Remove(*socketPath)
	}()

	server := &http.Server{Handler: httpServer{app: a}.buildMux()}

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(a.stderr, "serve http: %v\n", err)

		return 1
	}

	return 0
}
