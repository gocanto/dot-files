package app

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
)

type onePasswordSession struct {
	stdout   io.Writer
	runner   command.Runner
	lookPath func(string) (string, error)
}

func (a app) ensureOpSession() error {
	return onePasswordSession{stdout: a.stdout, runner: a.runner}.Ensure()
}

func (s onePasswordSession) Ensure() error {
	lookPath := s.lookPath

	if lookPath == nil {
		lookPath = exec.LookPath
	}

	if _, err := lookPath("op"); err != nil {
		return fmt.Errorf("1Password CLI (op) not found in PATH; install it (brew install 1password-cli) and rerun")
	}

	if _, err := s.runner.Run("op", "whoami"); err == nil {
		fmt.Fprintln(s.stdout, "1Password CLI session is active")

		return nil
	}

	out, err := s.runner.Run("op", "account", "list")

	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return fmt.Errorf("no 1Password account configured for the CLI; either enable 'Integrate with 1Password CLI' in the 1Password app's Developer settings or run 'op account add', then rerun")
	}

	fmt.Fprintln(s.stdout, "1Password CLI is not signed in; running: op signin")

	if err := command.RunInteractive(s.runner, s.stdout, "op", "signin"); err != nil {
		return fmt.Errorf("op signin failed: %w", err)
	}

	if _, err := s.runner.Run("op", "whoami"); err != nil {
		return fmt.Errorf("op signin completed but session is still inactive: %w", err)
	}

	return nil
}
