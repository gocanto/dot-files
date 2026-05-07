package app

import (
	"path/filepath"
	"strings"
)

func resolvePath(home, repo, path string) string {
	path = strings.TrimSpace(path)

	if path == "~" {
		return filepath.Clean(home)
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	return filepath.Join(repo, path)
}
