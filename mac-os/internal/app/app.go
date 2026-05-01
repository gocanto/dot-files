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
	stdout    io.Writer
	stderr    io.Writer
	stdin     io.Reader
	runner    command.Runner
	tuiRunner func(io.Reader, io.Writer, []tui.Workflow) (tui.Result, error)
}

type options struct {
	dryRun       bool
	yes          bool
	encrypt      bool
	apps         bool
	archiveRoot  string
	archivePath  string
	configPath   string
	secretsPath  string
	secretTarget string
	opVault      string
	opItem       string
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

	if args[0] != "secrets" && args[0] != "tui" {
		if err := a.requireSudo(); err != nil {
			fmt.Fprintf(a.stderr, "sudo access required: %v\n", err)

			return 1
		}
	}

	switch args[0] {
	case "bootstrap":
		return a.bootstrap(args[1:])
	case "adopt":
		return a.adopt(args[1:])
	case "capture":
		return a.capture(args[1:])
	case "restore":
		return a.restore(args[1:])
	case "doctor":
		return a.doctor(args[1:])
	case "brewfile":
		return a.brewfile(args[1:])
	case "macos":
		return a.macos(args[1:])
	case "secrets":
		return a.secrets(args[1:])
	case "tui":
		return a.tui(args[1:])
	case "help", "-h", "--help":
		a.usage()

		return 0
	default:
		fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.usage()

		return 2
	}
}
