package archive

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubRunner struct{}

func (stubRunner) Run(string, ...string) ([]byte, error) {
	return nil, nil
}

func TestCaptureDryRunShowsEncryptionPlan(t *testing.T) {
	var stdout bytes.Buffer
	s := Service{
		Home:   "/Users/gus",
		Repo:   "/repo",
		Stdout: &stdout,
		Runner: stubRunner{},
	}

	if err := s.Capture(Options{DryRun: true, Encrypt: true, OPVault: "Private", OPItem: "Mac Migration Archive"}); err != nil {
		t.Fatal(err)
	}

	got := stdout.String()

	for _, want := range []string{
		"would read 1Password item: Private/Mac Migration Archive",
		"would encrypt archive with Age recipient from 1Password",
		"would update 1Password latest_archive metadata",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry-run output missing %q\n%s", want, got)
		}
	}
}

func TestCaptureDryRunShowsAppConfigPlan(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "apps.yaml")
	content := []byte(`
apps:
  - name: Ghostty
    install_method: brew
    package: ghostty
    config_mode: auto
    config_paths:
      - source: ~/.config/ghostty/config
        target: apps/ghostty/config
`)

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	s := Service{
		Home:   "/Users/gus",
		Repo:   dir,
		Stdout: &stdout,
		Runner: stubRunner{},
	}

	if err := s.Capture(Options{DryRun: true, Apps: true}); err != nil {
		t.Fatal(err)
	}

	got := stdout.String()

	if !strings.Contains(got, "would capture app config: ~/.config/ghostty/config -> apps/ghostty/config") {
		t.Fatalf("dry-run output missing app config plan\n%s", got)
	}
}
