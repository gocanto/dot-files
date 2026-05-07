package services

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/gocanto/dot-files/internal/command"
)

type onePasswordSession struct {
	stdout   io.Writer
	runner   command.Runner
	lookPath func(string) (string, error)
}

type OpVault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OpItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ErrOpUnavailable signals that the 1Password CLI is missing or cannot access
// the configured 1Password account.
// Callers triggered from HTTP must surface this as a 503 rather than attempting
// an interactive `op signin` (which has no terminal to attach to).
type ErrOpUnavailable struct {
	Reason string
}

func (s Service) EnsureOpSession() error {
	return onePasswordSession{stdout: s.Stdout, runner: s.Runner}.Ensure()
}

func (s Service) ListOpVaults() ([]OpVault, error) {
	return onePasswordSession{stdout: s.Stdout, runner: s.Runner}.ListVaults()
}

func (s Service) ListOpItems(vault string) ([]OpItem, error) {
	return onePasswordSession{stdout: s.Stdout, runner: s.Runner}.ListItems(vault)
}

func (e ErrOpUnavailable) Error() string {
	return e.Reason
}

func (s onePasswordSession) ensureInstalled() error {
	lookPath := s.lookPath

	if lookPath == nil {
		lookPath = exec.LookPath
	}

	if _, err := lookPath("op"); err != nil {
		return ErrOpUnavailable{Reason: "1Password CLI (op) not found in PATH; run the Install Homebrew packages workflow or `brew install --cask 1password 1password-cli`, then retry"}
	}

	return nil
}

func (s onePasswordSession) ListVaults() ([]OpVault, error) {
	if err := s.ensureInstalled(); err != nil {
		return nil, err
	}

	out, err := s.runner.Run("op", "vault", "list", "--format=json")

	if err != nil {
		return nil, opAccessError("list 1Password vaults", out, err)
	}

	vaults := []OpVault{}

	if err := json.Unmarshal(out, &vaults); err != nil {
		return nil, fmt.Errorf("parse 1Password vault list JSON: %w", err)
	}

	return vaults, nil
}

func (s onePasswordSession) ListItems(vault string) ([]OpItem, error) {
	if strings.TrimSpace(vault) == "" {
		return nil, fmt.Errorf("vault name is required")
	}

	if err := s.ensureInstalled(); err != nil {
		return nil, err
	}

	out, err := s.runner.Run("op", "item", "list", "--vault="+vault, "--format=json")

	if err != nil {
		return nil, opAccessError(fmt.Sprintf("list 1Password items in vault %q", vault), out, err)
	}

	items := []OpItem{}

	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("parse 1Password item list JSON: %w", err)
	}

	return items, nil
}

func (s onePasswordSession) Ensure() error {
	if err := s.ensureInstalled(); err != nil {
		return err
	}

	if _, err := s.ListVaults(); err == nil {
		fmt.Fprintln(s.stdout, "1Password CLI access is active")

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

	if _, err := s.ListVaults(); err != nil {
		return fmt.Errorf("op signin completed but session is still inactive: %w", err)
	}

	return nil
}

func opAccessError(action string, out []byte, err error) error {
	detail := strings.TrimSpace(command.FirstLine(out))

	if detail == "" {
		detail = err.Error()
	}

	return ErrOpUnavailable{Reason: fmt.Sprintf("1Password CLI cannot %s: %s. Open 1Password, enable Developer > Integrate with 1Password CLI, sign in, then retry", action, detail)}
}
