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
        target: ghostty/config
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

	if !strings.Contains(got, "would capture app config: ~/.config/ghostty/config -> ghostty/config") {
		t.Fatalf("dry-run output missing app config plan\n%s", got)
	}
}

func TestLatestLocalSnapshotSelectsNewestTimestampedDirectory(t *testing.T) {
	home := t.TempDir()
	root := DefaultLocalRoot(home)

	for _, dir := range []string{
		filepath.Join(root, "20260102-030405"),
		filepath.Join(root, "20260103-030405"),
		filepath.Join(root, "notes"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "20260104-030405.tar.gz.age"), []byte("encrypted\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok, err := LatestLocalSnapshot(home)

	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected latest snapshot")
	}

	want := filepath.Join(root, "20260103-030405")

	if got != want {
		t.Fatalf("LatestLocalSnapshot() = %q, want %q", got, want)
	}
}

func TestLatestLocalSnapshotMissingRootSkips(t *testing.T) {
	got, ok, err := LatestLocalSnapshot(t.TempDir())

	if err != nil {
		t.Fatal(err)
	}

	if ok || got != "" {
		t.Fatalf("LatestLocalSnapshot() = %q, %v; want empty false", got, ok)
	}
}
