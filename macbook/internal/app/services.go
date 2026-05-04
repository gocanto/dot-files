package app

import (
	"fmt"
	"os"
	"path/filepath"

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
	return stowService{home: a.home, repo: a.repo, stdout: a.stdout, runner: a.runner}.Apply(opts.dryRun)
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
	archivePath := opts.archivePath

	if opts.useLatestArchive && archivePath == "" {
		archiveRoot := opts.archiveRoot

		if archiveRoot == "" {
			archiveRoot = archive.DefaultLocalRoot(a.home)
		}

		latest, ok, err := archive.LatestSnapshot(archiveRoot)

		if err != nil {
			return err
		}

		if !ok {
			fmt.Fprintf(a.stdout, "skipped: no local app settings snapshot found under %s\n", archiveRoot)

			return nil
		}

		archivePath = latest
		fmt.Fprintf(a.stdout, "using latest local app settings snapshot: %s\n", archivePath)
	}

	return a.apps().RestoreConfigs(apps.Options{DryRun: opts.dryRun, Apps: opts.apps, ArchivePath: archivePath, ConfigPath: opts.configPath})
}

func (a app) updateInstalledAppList(opts options) error {
	return a.apps().GenerateInstalledList(apps.Options{DryRun: opts.dryRun, ConfigPath: opts.configPath, GeneratedPath: opts.generatedPath})
}

func (a app) runDoctor(options) error {
	return doctor.Service{GOOS: a.goos, GOARCH: a.goarch, Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.Run(a.settings.OPVault, a.settings.OPItem, a.settings.SecretsConfigPath)
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
	svc := secrets.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}

	secretOpts := secrets.Options{
		DryRun:      opts.dryRun,
		SecretsPath: opts.secretsPath,
		OPVault:     opts.opVault,
		OPItem:      opts.opItem,
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
