package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/snapshot"
	"github.com/gocanto/mac-os/internal/storage"
	"github.com/gocanto/mac-os/internal/template/appconfig"
	"github.com/gocanto/mac-os/internal/template/secrets"
)

type runtimeSettings struct {
	RepoRoot          string `json:"repoRoot"`
	AppsConfigPath    string `json:"appsConfigPath"`
	SecretsConfigPath string `json:"secretsConfigPath"`
	GeneratedAppsPath string `json:"generatedAppsPath"`
	ArchiveRoot       string `json:"archiveRoot"`
	WorkflowDBPath    string `json:"workflowDbPath"`
	OPVault           string `json:"opVault"`
	OPItem            string `json:"opItem"`
}

type settingsCheck struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Path    string `json:"path"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type settingsValidation struct {
	Settings runtimeSettings `json:"settings"`
	Checks   []settingsCheck `json:"checks"`
	Valid    bool            `json:"valid"`
}

const (
	checkOK    = "ok"
	checkError = "error"
)

func defaultRuntimeSettings(home, repo string) runtimeSettings {
	return runtimeSettings{
		RepoRoot:          repo,
		AppsConfigPath:    filepath.Join(repo, "apps.yaml"),
		SecretsConfigPath: filepath.Join(repo, "secrets.yaml"),
		GeneratedAppsPath: filepath.Join(repo, "apps.generated.yaml"),
		ArchiveRoot:       snapshot.DefaultLocalRoot(home),
		WorkflowDBPath:    storage.DefaultPath(home),
		OPVault:           defaultOPVault,
		OPItem:            defaultOPItem,
	}
}

func (s runtimeSettings) withDefaults(home, fallbackRepo string) runtimeSettings {
	if strings.TrimSpace(s.RepoRoot) == "" {
		s.RepoRoot = fallbackRepo
	}

	repo := resolvePath(home, fallbackRepo, s.RepoRoot)
	s.RepoRoot = repo

	defaults := defaultRuntimeSettings(home, repo)

	if strings.TrimSpace(s.AppsConfigPath) == "" {
		s.AppsConfigPath = defaults.AppsConfigPath
	}

	if strings.TrimSpace(s.SecretsConfigPath) == "" {
		s.SecretsConfigPath = defaults.SecretsConfigPath
	}

	if strings.TrimSpace(s.GeneratedAppsPath) == "" {
		s.GeneratedAppsPath = defaults.GeneratedAppsPath
	}

	if strings.TrimSpace(s.ArchiveRoot) == "" {
		s.ArchiveRoot = defaults.ArchiveRoot
	}

	if strings.TrimSpace(s.WorkflowDBPath) == "" {
		s.WorkflowDBPath = defaults.WorkflowDBPath
	}

	if strings.TrimSpace(s.OPVault) == "" {
		s.OPVault = defaults.OPVault
	}

	if strings.TrimSpace(s.OPItem) == "" {
		s.OPItem = defaults.OPItem
	}

	s.AppsConfigPath = resolvePath(home, repo, s.AppsConfigPath)
	s.SecretsConfigPath = resolvePath(home, repo, s.SecretsConfigPath)
	s.GeneratedAppsPath = resolvePath(home, repo, s.GeneratedAppsPath)
	s.ArchiveRoot = resolvePath(home, repo, s.ArchiveRoot)
	s.WorkflowDBPath = resolvePath(home, repo, s.WorkflowDBPath)

	return s
}

func resolvePath(home, repo, path string) string {
	path = strings.TrimSpace(path)

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	return filepath.Join(repo, path)
}

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

func dirExists(path string) error {
	info, err := os.Stat(path)

	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	return nil
}

func pathNotFile(path string) error {
	info, err := os.Stat(path)

	if err == nil && !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

func parentWritableOrCreatable(path string) error {
	parent := filepath.Dir(path)
	info, err := os.Stat(parent)

	if err == nil && info.IsDir() {
		return nil
	}

	if err == nil && !info.IsDir() {
		return fmt.Errorf("%s is not a directory", parent)
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func sqlitePathValid(path string) error {
	info, err := os.Stat(path)

	if err == nil && info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return parentWritableOrCreatable(path)
}
