package apps

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

func TestRestoreDryRunShowsAppConfigPlan(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "apps.yaml")
	archive := filepath.Join(dir, "archive")
	source := filepath.Join(archive, "apps/ghostty/config")
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

	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(source, []byte("font-size = 16\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	s := Service{Home: "/Users/gus", Repo: dir, Stdout: &stdout, Runner: stubRunner{}}

	if err := s.RestoreConfigs(Options{DryRun: true, Apps: true, ArchivePath: archive}); err != nil {
		t.Fatal(err)
	}

	got := stdout.String()

	if !strings.Contains(got, "would restore app config:") {
		t.Fatalf("dry-run output missing restore plan\n%s", got)
	}

	if !strings.Contains(got, "/Users/gus/.config/ghostty/config") {
		t.Fatalf("dry-run output missing expanded target\n%s", got)
	}
}
