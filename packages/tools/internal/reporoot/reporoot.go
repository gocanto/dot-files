package reporoot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Find(start string) (string, error) {
	if start == "" {
		var err error

		start, err = os.Getwd()

		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
	}

	dir, err := filepath.Abs(start)

	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	for {
		if hasFile(dir, "pnpm-workspace.yaml") && hasFile(dir, "turbo.json") && hasDir(dir, "packages") {
			return dir, nil
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return "", errors.New("repository root not found")
		}

		dir = parent
	}
}

func hasFile(dir string, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))

	return err == nil && !info.IsDir()
}

func hasDir(dir string, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))

	return err == nil && info.IsDir()
}
