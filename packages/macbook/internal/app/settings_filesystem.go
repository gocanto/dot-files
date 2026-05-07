package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
	existing := parent

	for {
		info, err := os.Stat(existing)

		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", existing)
			}

			return writableDir(existing)
		}

		if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		next := filepath.Dir(existing)

		if next == existing {
			return fmt.Errorf("no existing parent directory for %s", parent)
		}

		existing = next
	}
}

func writableDir(path string) error {
	probe, err := os.CreateTemp(path, ".write-check-*")

	if err != nil {
		return fmt.Errorf("%s is not writable: %w", path, err)
	}

	name := probe.Name()

	if err := probe.Close(); err != nil {
		_ = os.Remove(name)

		return err
	}

	return os.Remove(name)
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
