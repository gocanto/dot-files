package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidatesModesAndPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apps.yaml")
	content := []byte(`
apps:
  - name: Ghostty
    bundle_id: com.mitchellh.ghostty
    install_method: brew
    package: ghostty
    config_mode: auto
    config_paths:
      - source: ~/.config/ghostty/config
        target: apps/ghostty/config
`)

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := (Loader{Home: "/Users/gus", Repo: dir}).Load("")

	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Apps) != 1 {
		t.Fatalf("loaded %d apps, want 1", len(cfg.Apps))
	}

	if got := cfg.Apps[0].Package; got != "ghostty" {
		t.Fatalf("package = %q, want ghostty", got)
	}
}

func TestLoadRejectsInvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apps.yaml")
	content := []byte(`
apps:
  - name: Broken
    install_method: curl
    config_mode: auto
    config_paths:
      - source: ~/.config/broken
        target: apps/broken
`)

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := (Loader{Home: "/Users/gus", Repo: dir}).Load("")

	if err == nil {
		t.Fatal("expected invalid install mode error")
	}

	if !strings.Contains(err.Error(), "install_method") {
		t.Fatalf("error = %v, want install_method validation", err)
	}
}

func TestCapturePlanSkipsManualConfig(t *testing.T) {
	cfg := Config{Apps: []ManagedApp{
		{
			Name:          "Ghostty",
			InstallMethod: "brew",
			Package:       "ghostty",
			ConfigMode:    "auto",
			ConfigPaths: []ConfigPath{
				{Source: "~/.config/ghostty/config", Target: "apps/ghostty/config"},
			},
		},
		{
			Name:          "Slack",
			InstallMethod: "mas",
			Package:       "803453959",
			ConfigMode:    "manual",
			ConfigPaths: []ConfigPath{
				{Source: "~/Library/Application Support/Slack", Target: "apps/slack"},
			},
		},
	}}

	got := CapturePlan(cfg)

	if len(got) != 1 {
		t.Fatalf("CapturePlan returned %d items, want 1", len(got))
	}

	if got[0].Target != "apps/ghostty/config" {
		t.Fatalf("target = %q", got[0].Target)
	}
}
