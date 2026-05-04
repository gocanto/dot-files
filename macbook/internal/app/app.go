package app

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/tui"
)

type app struct {
	home      string
	repo      string
	goos      string
	goarch    string
	stdout    io.Writer
	stderr    io.Writer
	stdin     io.Reader
	runner    command.Runner
	tuiRunner func(io.Reader, io.Writer, []tui.Workflow) (tui.Result, error)
}

type options struct {
	dryRun           bool
	encrypt          bool
	apps             bool
	archiveRoot      string
	archivePath      string
	useLatestArchive bool
	configPath       string
	generatedPath    string
	opVault          string
	opItem           string
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
		home:      home,
		repo:      repo,
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		stdout:    stdout,
		stderr:    stderr,
		stdin:     stdin,
		runner:    runner,
		tuiRunner: tui.Run,
	}
}

func (a app) run(args []string) int {
	if len(args) == 0 {
		return a.tui(nil)
	}

	switch args[0] {
	case "help", "-h", "--help":
		a.usage()

		return 0
	default:
		fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.usage()

		return 2
	}
}
