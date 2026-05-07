package turbocache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gocanto/dot-files/tools/internal/runner"
)

type Tool struct {
	RootDir string
	Runner  runner.CommandRunner
}

func (t Tool) Run(ctx context.Context, args []string) int {
	commandRunner := t.Runner

	if commandRunner == nil {
		commandRunner = runner.ExecRunner{}
	}

	err := commandRunner.Run(ctx, runner.CommandSpec{
		Cwd:  t.RootDir,
		Name: "pnpm",
		Args: append([]string{"exec", "turbo"}, args...),
	})

	moveErr := MovePackageLogs(t.RootDir)

	if moveErr != nil {
		fmt.Fprintf(os.Stderr, "Failed to move Turbo logs into storage/.cache: %s\n", moveErr)
	}

	if err != nil {

		if exitErr, ok := errors.AsType[*runner.ExitError](err); ok {
			if moveErr != nil {
				return 1
			}

			return exitErr.ExitCode
		}

		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	if moveErr != nil {
		return 1
	}

	return 0
}

func MovePackageLogs(rootDir string) error {
	packageDirs, err := PackageDirs(rootDir)

	if err != nil {
		return err
	}

	turboLogCacheDir := filepath.Join(rootDir, "storage", ".cache", "turbo-logs")

	for _, packageDir := range packageDirs {
		turboDir := filepath.Join(packageDir, ".turbo")
		info, err := os.Lstat(turboDir)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return fmt.Errorf("inspect %s: %w", turboDir, err)
		}

		packageName := filepath.Base(packageDir)
		destinationDir := filepath.Join(turboLogCacheDir, packageName)

		if info.Mode()&os.ModeSymlink != 0 {
			realPath, err := filepath.EvalSymlinks(turboDir)

			if err != nil {
				if removeErr := os.Remove(turboDir); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
					return fmt.Errorf("remove broken symlink %s: %w", turboDir, removeErr)
				}

				continue
			}

			realInfo, err := os.Stat(realPath)

			if err != nil {
				return fmt.Errorf("stat symlink target %s: %w", realPath, err)
			}

			if realInfo.IsDir() && !samePath(realPath, destinationDir) {
				if err := MoveDirectoryContents(realPath, destinationDir); err != nil {
					return err
				}
			}

			if err := os.Remove(turboDir); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove symlink %s: %w", turboDir, err)
			}

			continue
		}

		if info.IsDir() {
			if err := MoveDirectoryContents(turboDir, destinationDir); err != nil {
				return err
			}

			if err := os.RemoveAll(turboDir); err != nil {
				return fmt.Errorf("remove %s: %w", turboDir, err)
			}
		}
	}

	return nil
}

func PackageDirs(rootDir string) ([]string, error) {
	packagesDir := filepath.Join(rootDir, "packages")
	entries, err := os.ReadDir(packagesDir)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("read packages directory: %w", err)
	}

	var dirs []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packageDir := filepath.Join(packagesDir, entry.Name())
		info, err := os.Stat(filepath.Join(packageDir, "package.json"))

		if err == nil && !info.IsDir() {
			dirs = append(dirs, packageDir)

			continue
		}

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect package manifest %s: %w", packageDir, err)
		}
	}

	return dirs, nil
}

func MoveDirectoryContents(sourceDir string, destinationDir string) error {
	if err := os.MkdirAll(destinationDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", destinationDir, err)
	}

	entries, err := os.ReadDir(sourceDir)

	if err != nil {
		return fmt.Errorf("read %s: %w", sourceDir, err)
	}

	for _, entry := range entries {
		from := filepath.Join(sourceDir, entry.Name())
		to := filepath.Join(destinationDir, entry.Name())

		if err := os.RemoveAll(to); err != nil {
			return fmt.Errorf("remove %s: %w", to, err)
		}

		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("move %s to %s: %w", from, to, err)
		}
	}

	return nil
}

func samePath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)

	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}

	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}
