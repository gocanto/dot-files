package app

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/apps"
	"github.com/gocanto/mac-os/internal/archive"
	"github.com/gocanto/mac-os/internal/brewfile"
	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/doctor"
	"github.com/gocanto/mac-os/internal/dotfiles"
	"github.com/gocanto/mac-os/internal/githubsetup"
	"github.com/gocanto/mac-os/internal/macosdefaults"
	"github.com/gocanto/mac-os/internal/secrets"
)

// findStaleStowLinks walks each stow package's source tree and reports
// existing $HOME symlinks whose target lies outside the current stowDir
// (e.g. a leftover link from a prior run out of ~/Downloads/dot-files-main).
// Stow itself silently skips these and the user keeps editing files in the
// active repo without ever changing what $HOME sees, so we surface them
// loudly with the path each link currently points to.

type staleStowLink struct {
	home     string
	pointsTo string
}

func (a app) ensurePrerequisites(opts options) error {
	return doctor.Service{GOOS: a.goos, GOARCH: a.goarch, Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.EnsurePrerequisites(opts.dryRun)
}

func (a app) applyHomebrewBundle(opts options) error {
	brewfilePath := filepath.Join(os.TempDir(), "mac-os-Brewfile")

	if err := os.WriteFile(brewfilePath, []byte(brewfile.Content()), 0o644); err != nil {
		return fmt.Errorf("write generated Brewfile to %s: %w", brewfilePath, err)
	}

	cmd := []string{"brew", "bundle", "--verbose", "--file", brewfilePath}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	logPath := filepath.Join(os.TempDir(), "mac-os-homebrew-bundle.log")
	fmt.Fprintf(a.stdout, "logging full output to %s\n", logPath)

	out, runErr := a.runner.Run(cmd[0], cmd[1:]...)

	if writeErr := os.WriteFile(logPath, out, 0o644); writeErr != nil {
		fmt.Fprintf(a.stdout, "warning: could not write log file: %v\n", writeErr)
	}

	if len(out) > 0 {
		fmt.Fprint(a.stdout, string(out))
	}

	if runErr != nil {
		return fmt.Errorf("brew bundle failed (full log: %s): %w", logPath, runErr)
	}

	return nil
}

func (a app) applyAppStoreApps(opts options) error {
	return a.apps().ApplyAppStore(apps.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
}

func (a app) reportManualApps(opts options) error {
	return a.apps().ReportManual(apps.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
}

func (a app) setupGitHub(opts options) error {
	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would validate 1Password CLI session: op whoami (and op signin if needed)\n")
	} else if err := a.ensureOpSession(); err != nil {
		return err
	}

	return githubsetup.Service{
		Home:   a.home,
		Repo:   a.repo,
		Stdin:  a.stdin,
		Stdout: a.stdout,
		Runner: a.runner,
	}.Setup(githubsetup.Options{
		DryRun:  opts.dryRun,
		OPVault: opts.opVault,
		OPItem:  opts.opItem,
	})
}

func (a app) applyStow(opts options) error {
	stowDir := filepath.Join(a.repo, "stow")

	if _, err := os.Stat(stowDir); err != nil {
		return fmt.Errorf("missing stow directory at %s", stowDir)
	}

	entries, err := os.ReadDir(stowDir)

	if err != nil {
		return err
	}

	stale, err := findStaleStowLinks(a.home, stowDir, entries)

	if err != nil {
		return fmt.Errorf("scan for stale stow links under %s: %w", a.home, err)
	}

	if len(stale) > 0 {
		printStaleStowLinks(a.stdout, a.home, stowDir, stale)

		return fmt.Errorf("found %d stale stow link(s) pointing outside %s; remove them and rerun", len(stale), stowDir)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		cmd := []string{"stow", "--dir", stowDir, "--target", a.home, "--verbose", entry.Name()}

		if opts.dryRun {
			cmd = append(cmd, "--no")
			fmt.Fprintf(a.stdout, "would run: %s\n", command.ShellQuote(cmd))

			continue
		}

		out, err := a.runner.Run(cmd[0], cmd[1:]...)
		fmt.Fprint(a.stdout, string(out))

		if err != nil {
			return err
		}
	}

	return nil
}

func (a app) adoptDotfiles(opts options) error {
	return dotfiles.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout}.Adopt(opts.dryRun)
}

func (a app) ensureOhMyZsh(opts options) error {
	marker := filepath.Join(a.home, ".oh-my-zsh", "oh-my-zsh.sh")

	if _, err := os.Stat(marker); err == nil {
		fmt.Fprintln(a.stdout, "oh-my-zsh found")

		return nil
	}

	script := `set -e
RUNZSH=no CHSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
`
	cmd := []string{"sh", "-c", script}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	out, err := a.runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(a.stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("install oh-my-zsh: %w", err)
	}

	return nil
}

func (a app) applyMacOSDefaults(opts options) error {
	return macosdefaults.Service{Runner: a.runner, Stdout: a.stdout, Stderr: a.stderr}.Apply(opts.dryRun)
}

func (a app) captureArchive(opts options) error {
	return archive.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Stderr: a.stderr, Runner: a.runner}.Capture(archive.Options{
		DryRun:      opts.dryRun,
		Encrypt:     opts.encrypt,
		Apps:        opts.apps,
		ArchiveRoot: opts.archiveRoot,
		ConfigPath:  opts.configPath,
		OPVault:     opts.opVault,
		OPItem:      opts.opItem,
	})
}

func (a app) restoreAppConfigs(opts options) error {
	return a.apps().RestoreConfigs(apps.Options{DryRun: opts.dryRun, Apps: opts.apps, ArchivePath: opts.archivePath, ConfigPath: opts.configPath})
}

func (a app) runDoctor(options) error {
	return doctor.Service{GOOS: a.goos, GOARCH: a.goarch, Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.Run(defaultOPVault, defaultOPItem)
}

func (a app) openEraseAssistant(dryRun bool) error {
	fmt.Fprintln(a.stdout, "Erase first selected.")
	fmt.Fprintln(a.stdout, "Use Apple's Erase Assistant: System Settings > General > Transfer or Reset > Erase All Content and Settings.")
	fmt.Fprintln(a.stdout, "Factory install will stop now. Run this tool again after the Mac returns to setup or after you decide to proceed without erasing.")

	sudoCmd := []string{"sudo", "-v"}
	cmd := []string{"open", "x-apple.systempreferences:com.apple.Transfer-Reset-Settings.extension"}

	if dryRun {
		fmt.Fprintf(a.stdout, "would validate administrator access: %s\n", command.ShellQuote(sudoCmd))
		fmt.Fprintf(a.stdout, "would open reset settings: %s\n", command.ShellQuote(cmd))

		return nil
	}

	if a.goos != "darwin" {
		fmt.Fprintf(a.stdout, "skipped opening reset settings: current OS is %s\n", a.goos)

		return nil
	}

	fmt.Fprintf(a.stdout, "validating administrator access: %s\n", command.ShellQuote(sudoCmd))

	if err := command.RunInteractive(a.runner, a.stdout, sudoCmd[0], sudoCmd[1:]...); err != nil {
		return fmt.Errorf("validate administrator access: %w", err)
	}

	fmt.Fprintf(a.stdout, "opening reset settings: %s\n", command.ShellQuote(cmd))

	out, err := a.runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(a.stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("open Erase Assistant settings: %w", err)
	}

	return nil
}

func (a app) apps() apps.Service {
	return apps.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}
}

func (a app) restorePrivateSecrets(opts options) error {
	svc := secrets.Service{Repo: a.repo, Stdout: a.stdout, Runner: a.runner}

	secretOpts := secrets.Options{
		DryRun:  opts.dryRun,
		OPVault: opts.opVault,
		OPItem:  opts.opItem,
	}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would validate 1Password CLI session: op whoami (and op signin if needed)\n")
		fmt.Fprintf(a.stdout, "would decrypt private secrets from 1Password item %q in vault %q\n", opts.opItem, opts.opVault)

		return svc.Decrypt(secretOpts)
	}

	if err := a.ensureOpSession(); err != nil {
		return err
	}

	return svc.Decrypt(secretOpts)
}

func (a app) ensureOpSession() error {
	if _, err := exec.LookPath("op"); err != nil {
		return fmt.Errorf("1Password CLI (op) not found in PATH; install it (brew install 1password-cli) and rerun")
	}

	if _, err := a.runner.Run("op", "whoami"); err == nil {
		fmt.Fprintln(a.stdout, "1Password CLI session is active")

		return nil
	}

	out, err := a.runner.Run("op", "account", "list")

	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return fmt.Errorf("no 1Password account configured for the CLI; either enable 'Integrate with 1Password CLI' in the 1Password app's Developer settings or run 'op account add', then rerun")
	}

	fmt.Fprintln(a.stdout, "1Password CLI is not signed in; running: op signin")

	if err := command.RunInteractive(a.runner, a.stdout, "op", "signin"); err != nil {
		return fmt.Errorf("op signin failed: %w", err)
	}

	if _, err := a.runner.Run("op", "whoami"); err != nil {
		return fmt.Errorf("op signin completed but session is still inactive: %w", err)
	}

	return nil
}

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
