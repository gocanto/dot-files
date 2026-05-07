package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gocanto/dot-files/internal/command"
)

type stubRunner struct {
	outputs map[string][]byte
	errors  map[string]error
	calls   *[]string
}

func (r stubRunner) Run(name string, args ...string) ([]byte, error) {
	key := command.ShellQuote(append([]string{name}, args...))

	if r.calls != nil {
		*r.calls = append(*r.calls, key)
	}

	return r.outputs[key], r.errors[key]
}

func writeSettingsRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()

	for _, dir := range []string{
		filepath.Join(repo, "stow"),
		filepath.Join(repo, "stow", "git", ".config", "git"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(repo, "apps.yaml"), []byte(`
apps:
  - name: Ghostty
    install_method: brew
    package: ghostty
    config_mode: manual
`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(repo, "secrets.yaml"), []byte(`
secrets:
  - name: gitconfig
    op_field: gitconfig_plaintext
    plaintext_path: stow/git/.config/git/private.gitconfig
    mode: plaintext
`), 0o600); err != nil {
		t.Fatal(err)
	}

	return repo
}
