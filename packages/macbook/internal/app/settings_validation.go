package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/template/appconfig"
	"github.com/gocanto/mac-os/internal/template/secrets"
)

func validateRuntimeSettings(home, fallbackRepo string, candidate runtimeSettings) settingsValidation {
	resolved := candidate.withDefaults(home, fallbackRepo)
	checks := []settingsCheck{}

	addCheck := func(key, label, path string, err error) {
		check := settingsCheck{Key: key, Label: label, Path: path, Status: checkOK, Message: "ok"}

		if err != nil {
			check.Status = checkError
			check.Message = err.Error()
		}

		checks = append(checks, check)
	}

	root, err := validateRepoRoot(resolved.RepoRoot)

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
		if check.Status != checkOK {
			valid = false

			break
		}
	}

	return settingsValidation{Settings: resolved, Checks: checks, Valid: valid}
}

func validateRepoRoot(root string) (string, error) {
	root = strings.TrimSpace(root)

	if root == "" {
		return "", errors.New("path is required")
	}

	abs, err := filepath.Abs(root)

	if err != nil {
		return root, err
	}

	if !hasRepoMarkers(abs) {
		return abs, fmt.Errorf("missing repo markers: expected %s and %s", filepath.Join(abs, "stow"), filepath.Join(abs, "go.mod"))
	}

	return abs, nil
}
