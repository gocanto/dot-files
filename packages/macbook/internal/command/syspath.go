package command

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// EnsureSystemPath prepends standard macOS developer-tool directories to PATH
// so that GUI-launched processes (which inherit only /usr/bin:/bin:/usr/sbin:/sbin
// from launchd) can locate Homebrew and user-local binaries. No-op on non-darwin.
func EnsureSystemPath() {
	if runtime.GOOS != "darwin" {
		return
	}

	candidates := []string{
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/sbin",
	}

	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, "bin"),
		)
	}

	updated := augmentPath(os.Getenv("PATH"), candidates, isDir)

	if updated == "" {
		return
	}

	os.Setenv("PATH", updated)
}

// augmentPath returns a PATH string with each candidate prepended (preserving
// candidate order) when the directory exists and is not already present.
// Returns "" when no changes are needed.
func augmentPath(current string, candidates []string, exists func(string) bool) string {
	existing := make(map[string]struct{})

	for _, entry := range strings.Split(current, string(os.PathListSeparator)) {
		if entry != "" {
			existing[entry] = struct{}{}
		}
	}

	var prepend []string

	for _, dir := range candidates {
		if _, seen := existing[dir]; seen {
			continue
		}

		if !exists(dir) {
			continue
		}

		prepend = append(prepend, dir)
		existing[dir] = struct{}{}
	}

	if len(prepend) == 0 {
		return ""
	}

	updated := strings.Join(prepend, string(os.PathListSeparator))

	if current != "" {
		updated = updated + string(os.PathListSeparator) + current
	}

	return updated
}

func isDir(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}
