package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRepoRootAcceptsExplicitValidRoot(t *testing.T) {
	repo := writeSettingsRepo(t)
	nested := filepath.Join(t.TempDir(), "elsewhere")

	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := resolveRepoRoot(nested, repo)

	if err != nil {
		t.Fatal(err)
	}

	if got != repo {
		t.Fatalf("resolveRepoRoot = %q, want %q", got, repo)
	}
}

func TestResolveRepoRootRejectsExplicitInvalidRoot(t *testing.T) {
	dir := t.TempDir()

	if _, err := resolveRepoRoot(dir, dir); err == nil {
		t.Fatal("expected invalid repo root error")
	}
}

func TestValidateRuntimeSettingsResolvesPathsAndRejectsDirectoryDB(t *testing.T) {
	home := t.TempDir()
	repo := writeSettingsRepo(t)
	dbDir := filepath.Join(home, "dbdir")

	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		t.Fatal(err)
	}

	validation := validateRuntimeSettings(home, repo, runtimeSettings{
		RepoRoot:          repo,
		AppsConfigPath:    "apps.yaml",
		SecretsConfigPath: "~/secrets.yaml",
		GeneratedAppsPath: "generated/apps.yaml",
		ArchiveRoot:       "~/archives",
		WorkflowDBPath:    dbDir,
	})

	if validation.Valid {
		t.Fatalf("expected invalid settings, got %#v", validation)
	}

	if validation.Settings.AppsConfigPath != filepath.Join(repo, "apps.yaml") {
		t.Fatalf("apps path = %q", validation.Settings.AppsConfigPath)
	}

	if validation.Settings.SecretsConfigPath != filepath.Join(home, "secrets.yaml") {
		t.Fatalf("secrets path = %q", validation.Settings.SecretsConfigPath)
	}

	if !hasSettingsCheck(validation.Checks, "workflow_db_path", checkError) {
		t.Fatalf("checks = %#v", validation.Checks)
	}
}

func TestValidateRuntimeSettingsAcceptsValidDefaults(t *testing.T) {
	home := t.TempDir()
	repo := writeSettingsRepo(t)
	validation := validateRuntimeSettings(home, repo, runtimeSettings{RepoRoot: repo})

	if !validation.Valid {
		t.Fatalf("settings should be valid: %#v", validation.Checks)
	}

	if validation.Settings.WorkflowDBPath == "" {
		t.Fatal("expected default workflow db path")
	}
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

func hasSettingsCheck(checks []settingsCheck, key, status string) bool {
	for _, check := range checks {
		if check.Key == key && check.Status == status {
			return true
		}
	}

	return false
}
