package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findRepoRoot(start string) string {
	root, err := resolveRepoRoot(start, "")

	if err != nil {
		return start
	}

	return root
}

func resolveRepoRoot(start, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		root, err := validateRepoRoot(explicit)

		if err != nil {
			return root, err
		}

		return root, nil
	}

	if root, ok := walkForRepoRoot(start); ok {
		return root, nil
	}

	exe, err := os.Executable()

	if err == nil {
		if root, ok := walkForRepoRoot(filepath.Dir(exe)); ok {
			return root, nil
		}
	}

	return start, fmt.Errorf("could not find repo root from %s", start)
}

func walkForRepoRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)

	if err != nil {
		return start, false
	}

	for {
		if hasRepoMarkers(dir) {
			return dir, true
		}

		macOSDir := filepath.Join(dir, "packages", "macbook")

		if hasRepoMarkers(macOSDir) {
			return macOSDir, true
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return start, false
		}

		dir = parent
	}
}

func hasRepoMarkers(dir string) bool {
	if info, err := os.Stat(filepath.Join(dir, "stow")); err != nil || !info.IsDir() {
		return false
	}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}

	return true
}
