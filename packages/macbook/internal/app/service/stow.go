package service

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/dot-files/internal/command"
)

type stowService struct {
	home   string
	repo   string
	stdout io.Writer
	runner command.Runner
}

type staleStowLink struct {
	home     string
	pointsTo string
}

func (s stowService) Apply(dryRun bool) error {
	stowDir := filepath.Join(s.repo, "stow")

	if _, err := os.Stat(stowDir); err != nil {
		return fmt.Errorf("missing stow directory at %s", stowDir)
	}

	entries, err := os.ReadDir(stowDir)

	if err != nil {
		return err
	}

	stale, err := findStaleStowLinks(s.home, stowDir, entries)

	if err != nil {
		return fmt.Errorf("scan for stale stow links under %s: %w", s.home, err)
	}

	if len(stale) > 0 {
		printStaleStowLinks(s.stdout, s.home, stowDir, stale)

		return fmt.Errorf("found %d stale stow link(s) pointing outside %s; remove them and rerun", len(stale), stowDir)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		cmd := []string{"stow", "--dir", stowDir, "--target", s.home, "--verbose", entry.Name()}

		if dryRun {
			cmd = append(cmd, "--no")
			fmt.Fprintf(s.stdout, "would run: %s\n", command.ShellQuote(cmd))

			continue
		}

		out, err := s.runner.Run(cmd[0], cmd[1:]...)
		fmt.Fprint(s.stdout, string(out))

		if err != nil {
			return err
		}
	}

	return nil
}

// findStaleStowLinks walks each stow package's source tree and reports
// existing $HOME symlinks whose target lies outside the current stowDir.
func findStaleStowLinks(home, stowDir string, entries []os.DirEntry) ([]staleStowLink, error) {
	stowDirCanonical, err := filepath.EvalSymlinks(stowDir)

	if err != nil {
		stowDirCanonical = stowDir
	}

	stowPrefix := stowDirCanonical + string(os.PathSeparator)

	var stale []staleStowLink

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkg := filepath.Join(stowDir, entry.Name())
		walkErr := filepath.WalkDir(pkg, func(src string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if src == pkg {
				return nil
			}

			rel, err := filepath.Rel(pkg, src)

			if err != nil {
				return err
			}

			target := filepath.Join(home, rel)
			info, err := os.Lstat(target)

			if err != nil {
				return nil
			}

			if info.Mode()&os.ModeSymlink == 0 {
				return nil
			}

			resolved, err := filepath.EvalSymlinks(target)

			if err != nil {
				return nil
			}

			if resolved == stowDirCanonical || strings.HasPrefix(resolved, stowPrefix) {
				if d.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}

			stale = append(stale, staleStowLink{home: target, pointsTo: resolved})

			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		})

		if walkErr != nil {
			return nil, walkErr
		}
	}

	return stale, nil
}

func printStaleStowLinks(stdout io.Writer, home, stowDir string, links []staleStowLink) {
	fmt.Fprintln(stdout, "stow conflict: existing symlinks point to a different stow tree:")

	for _, link := range links {
		fmt.Fprintf(stdout, "  %s -> %s\n", link.home, link.pointsTo)
	}

	if oldStow := commonStowRoot(links); oldStow != "" && oldStow != stowDir {
		oldRepo := filepath.Dir(oldStow)
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "remove the stale links by unstowing from the old tree:")
		fmt.Fprintf(stdout, "  cd %s && stow --dir stow --target %s --delete <packages>\n", oldRepo, home)
		fmt.Fprintln(stdout, "  (or: rm the symlinks listed above)")
	}

	fmt.Fprintln(stdout, "then rerun this workflow.")
}

func commonStowRoot(links []staleStowLink) string {
	if len(links) == 0 {
		return ""
	}

	root := stowRootOf(links[0].pointsTo)

	if root == "" {
		return ""
	}

	for _, link := range links[1:] {
		if stowRootOf(link.pointsTo) != root {
			return ""
		}
	}

	return root
}

func stowRootOf(path string) string {
	const sep = string(os.PathSeparator)
	idx := strings.LastIndex(path, sep+"stow"+sep)

	if idx < 0 {
		return ""
	}

	return path[:idx+len(sep)+len("stow")]
}
