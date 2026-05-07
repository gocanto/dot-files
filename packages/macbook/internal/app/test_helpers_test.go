package app

import (
	"os"
	"path/filepath"
	"testing"
)

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
