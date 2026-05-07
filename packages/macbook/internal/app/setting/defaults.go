package setting

import (
	"path/filepath"
	"strings"

	"github.com/gocanto/dot-files/internal/snapshot"
	"github.com/gocanto/dot-files/internal/storage"
)

const (
	DefaultOPVault = "Private"
	DefaultOPItem  = "Mac Migration Archive"
)

func DefaultRuntimeSettings(home, repo string) RuntimeSettings {
	return RuntimeSettings{
		RepoRoot:          repo,
		AppsConfigPath:    filepath.Join(repo, "apps.yaml"),
		SecretsConfigPath: filepath.Join(repo, "secrets.yaml"),
		GeneratedAppsPath: filepath.Join(repo, "apps.generated.yaml"),
		ArchiveRoot:       snapshot.DefaultLocalRoot(home),
		WorkflowDBPath:    storage.DefaultPath(home),
		OPVault:           DefaultOPVault,
		OPItem:            DefaultOPItem,
	}
}

func (s RuntimeSettings) withDefaults(home, fallbackRepo string) RuntimeSettings {
	if strings.TrimSpace(s.RepoRoot) == "" {
		s.RepoRoot = fallbackRepo
	}

	repo := resolvePath(home, fallbackRepo, s.RepoRoot)
	s.RepoRoot = repo

	defaults := DefaultRuntimeSettings(home, repo)

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
