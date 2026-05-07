package setting

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/dot-files/internal/template/appconfig"
	"github.com/gocanto/dot-files/internal/template/secrets"
)

func ValidateRuntimeSettings(home, fallbackRepo string, candidate RuntimeSettings) Validation {
	resolved := candidate.withDefaults(home, fallbackRepo)
	checks := []Check{}

	addCheck := func(key, label, path string, err error) {
		check := Check{Key: key, Label: label, Path: path, Status: CheckOK, Message: "ok"}

		if err != nil {
			check.Status = CheckError
			check.Message = err.Error()
		}

		checks = append(checks, check)
	}

	root, err := ValidateRepoRoot(resolved.RepoRoot)

	if err == nil {
		resolved.RepoRoot = root
		resolved = resolved.withDefaults(home, root)
	}

	addCheck("repo_root", "Repository root", resolved.RepoRoot, err)
	addCheck("stow", "Stow directory", filepath.Join(resolved.RepoRoot, "stow"), dirExists(filepath.Join(resolved.RepoRoot, "stow")))

	_, appsErr := appconfig.Loader{Home: home, Repo: resolved.RepoRoot}.Load(resolved.AppsConfigPath)
	addCheck("apps_config_path", "Apps manifest", resolved.AppsConfigPath, appsErr)

	_, secretsErr := secrets.Service{Home: home, Repo: resolved.RepoRoot}.Load(resolved.SecretsConfigPath)
	addCheck("secrets_config_path", "Secrets manifest", resolved.SecretsConfigPath, secretsErr)

	addCheck("generated_apps_path", "Generated apps output", resolved.GeneratedAppsPath, parentWritableOrCreatable(resolved.GeneratedAppsPath))
	addCheck("archive_root", "Archive root", resolved.ArchiveRoot, pathNotFile(resolved.ArchiveRoot))
	addCheck("workflow_db_path", "Workflow SQLite database", resolved.WorkflowDBPath, sqlitePathValid(resolved.WorkflowDBPath))
	addCheck("private_gitconfig_path", "Private Git config", secrets.Service{Home: home, Repo: resolved.RepoRoot}.PrivateGitconfigPath(), nil)

	valid := true

	for _, check := range checks {
		if check.Status != CheckOK {
			valid = false

			break
		}
	}

	return Validation{Settings: resolved, Checks: checks, Valid: valid}
}

func ValidateRepoRoot(root string) (string, error) {
	root = strings.TrimSpace(root)

	if root == "" {
		return "", errors.New("path is required")
	}

	abs, err := filepath.Abs(root)

	if err != nil {
		return root, err
	}

	if !HasRepoMarkers(abs) {
		return abs, fmt.Errorf("missing repo markers: expected %s and %s", filepath.Join(abs, "stow"), filepath.Join(abs, "go.mod"))
	}

	return abs, nil
}

func HasRepoMarkers(dir string) bool {
	if info, err := os.Stat(filepath.Join(dir, "stow")); err != nil || !info.IsDir() {
		return false
	}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}

	return true
}
