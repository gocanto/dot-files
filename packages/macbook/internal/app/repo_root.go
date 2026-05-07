package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/dot-files/internal/app/setting"
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
		return setting.ValidateRepoRoot(explicit)
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
		if setting.HasRepoMarkers(dir) {
			return dir, true
		}

		macOSDir := filepath.Join(dir, "packages", "macbook")

		if setting.HasRepoMarkers(macOSDir) {
			return macOSDir, true
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return start, false
		}

		dir = parent
	}
}
