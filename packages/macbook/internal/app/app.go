package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/storage"
)

type app struct {
	home     string
	repo     string
	settings runtimeSettings
	goos     string
	goarch   string
	stdout   io.Writer
	stderr   io.Writer
	stdin    io.Reader
	runner   command.Runner
	store    *storage.Store
}

type options struct {
	dryRun            bool
	encrypt           bool
	apps              bool
	allowMasUninstall bool
	archiveRoot       string
	archivePath       string
	useLatestArchive  bool
	configPath        string
	generatedPath     string
	secretsPath       string
	opVault           string
	opItem            string
}

func (a app) workflowStore(ctx context.Context) (*storage.Store, func(), error) {
	if a.store != nil {
		return a.store, func() {}, nil
	}

	store, err := storage.Open(ctx, a.settings.WorkflowDBPath)

	if err != nil {
		return nil, nil, err
	}

	return store, func() { _ = store.Close() }, nil
}

const (
	defaultOPVault = "Private"
	defaultOPItem  = "Mac Migration Archive"
)

func Run(args []string) int {
	home, err := os.UserHomeDir()

	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find home directory: %v\n", err)

		return 1
	}

	repo, err := os.Getwd()

	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find working directory: %v\n", err)

		return 1
	}

	a := newApp(home, findRepoRoot(repo), os.Stdin, os.Stdout, os.Stderr, command.RealRunner{})

	return a.run(args)
}

func newApp(home, repo string, stdin io.Reader, stdout, stderr io.Writer, runner command.Runner) app {
	return app{
		home:     home,
		repo:     repo,
		settings: defaultRuntimeSettings(home, repo),
		goos:     runtime.GOOS,
		goarch:   runtime.GOARCH,
		stdout:   stdout,
		stderr:   stderr,
		stdin:    stdin,
		runner:   runner,
	}
}

func (a app) run(args []string) int {
	if len(args) == 0 {
		a.usage()

		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		a.usage()

		return 0
	case "serve-http":
		return a.serveHTTP(args[1:])
	case "list-workflows":
		return a.listWorkflows()
	case "run-workflow":
		return a.runWorkflowCLI(args[1:])
	default:
		fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.usage()

		return 2
	}
}
