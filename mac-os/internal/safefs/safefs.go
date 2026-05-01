package safefs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Item struct {
	Source string
	Target string
}

func WriteFile(path string, content []byte, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, content, perm)
}

func CopyFile(source, target string) error {
	data, err := os.ReadFile(source)

	if err != nil {
		return err
	}

	return WriteFile(target, data, 0o600)
}

func CopyDirSafe(source, target string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(source, path)

		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		if ShouldSkipSensitive(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		dst := filepath.Join(target, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0o700)
		}

		return CopyFile(path, dst)
	})
}

func CopySanitizedFile(source, target, home string) error {
	if ShouldSkipSensitive(source) {
		return nil
	}

	data, err := os.ReadFile(source)

	if err != nil {
		return err
	}

	data = SanitizeDotfile(source, home, data)

	return WriteFile(target, data, 0o600)
}

func CopyPlanItem(root, home string, item Item) error {
	source := ExpandHome(item.Source, home)

	info, err := os.Stat(source)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	target := filepath.Join(root, item.Target)

	if info.IsDir() {
		return CopyDirSafe(source, target)
	}

	return CopySanitizedFile(source, target, home)
}

func ShouldSkipSensitive(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	base := strings.ToLower(filepath.Base(path))

	if strings.Contains(lower, "/.ssh/id_") && !strings.HasSuffix(lower, ".pub") {
		return true
	}

	patterns := []string{
		".zsh_history",
		".bash_history",
		".mysql_history",
		".gnupg",
		"auth.json",
		"hosts.yml",
		"ngrok.yml",
		"cache",
		"session",
		"sessions",
		"file-history",
		"state.vscdb",
		"storage.json",
		"cookies",
		"login data",
		"machineid",
		"token",
		"secret",
		"private",
		"keyring",
		"keychain",
		"docker.raw",
		"database",
		"library/application support/google/chrome",
		"library/application support/bravesoftware",
	}

	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) || strings.Contains(base, pattern) {
			return true
		}
	}

	return false
}

func SanitizeDotfile(path, home string, data []byte) []byte {
	content := string(data)

	if home != "" {
		content = strings.ReplaceAll(content, home, "$HOME")
	}

	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		lower := strings.ToLower(line)

		if strings.Contains(lower, "machineid") ||
			strings.Contains(lower, "machine_id") ||
			strings.Contains(lower, "installation_id") ||
			strings.Contains(lower, "api_key") ||
			strings.Contains(lower, "apikey") ||
			strings.Contains(lower, "access_token") ||
			strings.Contains(lower, "refresh_token") ||
			strings.Contains(lower, "secret=") ||
			strings.Contains(lower, "token=") {
			kept = append(kept, "# redacted machine-specific or secret-like setting")

			continue
		}

		kept = append(kept, line)
	}

	out := strings.Join(kept, "\n")

	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	return []byte(out)
}

func ExpandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	return path
}
