package app

import (
	"fmt"
	"io"
	"os"
	"runtime"

	apphttpx "github.com/gocanto/dot-files/internal/app/httpx"
	"github.com/gocanto/dot-files/internal/app/service"
	"github.com/gocanto/dot-files/internal/app/setting"
	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/domain"
)

type app struct {
	home     string
	repo     string
	settings setting.RuntimeSettings
	goos     string
	goarch   string
	stdout   io.Writer
	stderr   io.Writer
	stdin    io.Reader
	runner   command.Runner
}

type options = service.Options

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
		settings: setting.DefaultRuntimeSettings(home, repo),
		goos:     runtime.GOOS,
		goarch:   runtime.GOARCH,
		stdout:   stdout,
		stderr:   stderr,
		stdin:    stdin,
		runner:   runner,
	}
}

func (a app) service() service.Service {
	return service.Service{
		Home:     a.home,
		Repo:     a.repo,
		GOOS:     a.goos,
		GOARCH:   a.goarch,
		Stdin:    a.stdin,
		Stdout:   a.stdout,
		Stderr:   a.stderr,
		Runner:   a.runner,
		Settings: a.settings,
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
		return apphttpx.Serve(args[1:], apphttpx.ServeConfig{
			Home:      a.home,
			Repo:      a.repo,
			Stderr:    a.stderr,
			Service:   a.httpService,
			Workflows: a.httpWorkflows,
		})
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

func (a app) httpService(settings setting.RuntimeSettings) service.Service {
	a.settings = settings
	a.repo = settings.RepoRoot

	return a.service()
}

func (a app) httpWorkflows(settings setting.RuntimeSettings) []domain.Workflow {
	a.settings = settings
	a.repo = settings.RepoRoot

	return a.workflows()
}
