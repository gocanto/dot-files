package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/converge/appstore"
	"github.com/gocanto/mac-os/internal/converge/brew"
	"github.com/gocanto/mac-os/internal/converge/github"
	convergemacos "github.com/gocanto/mac-os/internal/converge/macos"
	currentapps "github.com/gocanto/mac-os/internal/currentstate/apps"
	"github.com/gocanto/mac-os/internal/currentstate/doctor"
	"github.com/gocanto/mac-os/internal/safefs"
	"github.com/gocanto/mac-os/internal/snapshot"
	"github.com/gocanto/mac-os/internal/template/appconfig"
	"github.com/gocanto/mac-os/internal/template/brewfile"
	"github.com/gocanto/mac-os/internal/template/dotfiles"
	templatemacos "github.com/gocanto/mac-os/internal/template/macos"
	"github.com/gocanto/mac-os/internal/template/secrets"
)

type untrackedReport struct {
	Formulae []string
	Casks    []string
	AppStore []appstore.AppStoreApp
}

func (a app) ensurePrerequisites(opts options) error {
	return doctor.Service{GOOS: a.goos, GOARCH: a.goarch, Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.EnsurePrerequisites(opts.dryRun)
}

func (a app) applyHomebrewBundle(opts options) error {
	brewfileFile, err := os.CreateTemp("", "mac-os-*.Brewfile")

	if err != nil {
		return fmt.Errorf("create temporary Brewfile: %w", err)
	}

	brewfilePath := brewfileFile.Name()
	defer os.Remove(brewfilePath)

	if _, err := brewfileFile.Write([]byte(brewfile.Content())); err != nil {
		return fmt.Errorf("write generated Brewfile to %s: %w", brewfilePath, err)
	}

	if err := brewfileFile.Close(); err != nil {
		return fmt.Errorf("close generated Brewfile %s: %w", brewfilePath, err)
	}

	cmd := []string{"brew", "bundle", "--verbose", "--file", brewfilePath}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	logFile, err := os.CreateTemp("", "mac-os-homebrew-bundle-*.log")

	if err != nil {
		return fmt.Errorf("create Homebrew bundle log: %w", err)
	}

	logPath := logFile.Name()
	defer logFile.Close()

	fmt.Fprintf(a.stdout, "logging full output to %s\n", logPath)

	out, runErr := a.runner.Run(cmd[0], cmd[1:]...)

	if _, writeErr := logFile.Write(out); writeErr != nil {
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
	return a.appstore().ApplyAppStore(appstore.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
}

func (a app) reportManualApps(opts options) error {
	return a.appstore().ReportManual(appstore.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
}

func (a app) setupGitHub(opts options) error {
	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would validate 1Password CLI access: op vault list --format=json (and op signin if needed)\n")
	} else if err := a.ensureOpSession(); err != nil {
		return err
	}

	return github.Service{
		Home:   a.home,
		Repo:   a.repo,
		Stdin:  a.stdin,
		Stdout: a.stdout,
		Runner: a.runner,
	}.Setup(github.Options{
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
	return convergemacos.Service{Runner: a.runner, Stdout: a.stdout, Stderr: a.stderr}.Apply(opts.dryRun)
}

func (a app) captureArchive(opts options) error {
	return snapshot.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Stderr: a.stderr, Runner: a.runner}.Capture(snapshot.Options{
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

		if archiveRoot == "~" {
			archiveRoot = a.home
		}

		if strings.HasPrefix(archiveRoot, "~/") {
			archiveRoot = filepath.Join(a.home, strings.TrimPrefix(archiveRoot, "~/"))
		}

		if archiveRoot == "" {
			archiveRoot = snapshot.DefaultLocalRoot(a.home)
		}

		latest, ok, err := snapshot.LatestSnapshot(archiveRoot)

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

	return a.appstore().RestoreConfigs(appstore.Options{DryRun: opts.dryRun, Apps: opts.apps, ArchivePath: archivePath, ConfigPath: opts.configPath})
}

func (a app) updateInstalledAppList(opts options) error {
	return a.currentApps().GenerateInstalledList(currentapps.Options{DryRun: opts.dryRun, ConfigPath: opts.configPath, GeneratedPath: opts.generatedPath})
}

func (a app) writeDotfileCandidates(opts options) error {
	dest := filepath.Join(a.repo, ".template-candidates", "current-mac")
	fmt.Fprintf(a.stdout, "dotfile review candidate destination: %s\n", dest)

	for _, item := range dotfiles.CapturePlan() {
		if opts.dryRun {
			fmt.Fprintf(a.stdout, "would write dotfile candidate: %s -> %s\n", item.Source, filepath.Join(dest, item.Target))

			continue
		}

		if err := safefs.CopyPlanItem(dest, a.home, item); err != nil {
			return fmt.Errorf("write dotfile candidate %s: %w", item.Target, err)
		}

		fmt.Fprintf(a.stdout, "wrote dotfile candidate: %s\n", filepath.Join(dest, item.Target))
	}

	return nil
}

func (a app) runDoctor(options) error {
	return doctor.Service{GOOS: a.goos, GOARCH: a.goarch, Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.Run(a.settings.OPVault, a.settings.OPItem, a.settings.SecretsConfigPath)
}

func (a app) openEraseAssistant(dryRun bool) error {
	fmt.Fprintln(a.stdout, "Erase first selected.")
	fmt.Fprintln(a.stdout, "Use Apple's Erase Assistant: System Settings > General > Transfer or Reset > Erase All Content and Settings.")
	fmt.Fprintln(a.stdout, "Factory install will stop now. Run this tool again after the Mac returns to setup or after you decide to proceed without erasing.")

	cmd := []string{"open", "x-apple.systempreferences:com.apple.Transfer-Reset-Settings.extension"}

	if dryRun {
		fmt.Fprintf(a.stdout, "would open reset settings: %s\n", command.ShellQuote(cmd))

		return nil
	}

	if a.goos != "darwin" {
		fmt.Fprintf(a.stdout, "skipped opening reset settings: current OS is %s\n", a.goos)

		return nil
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

func (a app) appstore() appstore.Service {
	return appstore.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}
}

func (a app) currentApps() currentapps.Service {
	return currentapps.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}
}

func (a app) previewTemplateBrew(_ options) error {
	fmt.Fprintln(a.stdout, "# Tracked Homebrew bundle")
	fmt.Fprint(a.stdout, brewfile.Content())

	return nil
}

func (a app) previewTemplateApps(opts options) error {
	loader := appconfig.Loader{Home: a.home, Repo: a.repo}
	cfg, err := loader.Load(opts.configPath)

	if err != nil {
		return err
	}

	groups := map[string][]string{}

	for _, app := range cfg.Apps {
		groups[app.InstallMethod] = append(groups[app.InstallMethod], app.Name)
	}

	keys := make([]string, 0, len(groups))

	for key := range groups {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	fmt.Fprintln(a.stdout, "# Tracked apps from apps.yaml")

	for _, method := range keys {
		names := groups[method]

		sort.Strings(names)

		fmt.Fprintf(a.stdout, "[%s] (%d)\n", method, len(names))

		for _, name := range names {
			fmt.Fprintf(a.stdout, "  - %s\n", name)
		}
	}

	return nil
}

func (a app) previewTemplateMacOS(_ options) error {
	fmt.Fprintln(a.stdout, "# Tracked macOS settings")

	for _, setting := range templatemacos.Settings() {
		fmt.Fprintf(a.stdout, "  %s %s %s\n", setting.Domain, setting.Key, command.ShellQuote(setting.Args))
	}

	return nil
}

func (a app) previewTemplateDotfiles(_ options) error {
	stowDir := filepath.Join(a.repo, "stow")

	fmt.Fprintf(a.stdout, "reading stow directory: %s\n", stowDir)

	entries, err := os.ReadDir(stowDir)

	if err != nil {
		return fmt.Errorf("read stow dir %s: %w", stowDir, err)
	}

	fmt.Fprintf(a.stdout, "found %d stow %s\n", len(entries), plural(len(entries), "entry", "entries"))
	fmt.Fprintln(a.stdout, "# Tracked dotfile bundles under stow/")

	bundles := 0

	for _, entry := range entries {
		fmt.Fprintf(a.stdout, "checking stow entry: %s\n", entry.Name())

		if !entry.IsDir() {
			fmt.Fprintf(a.stdout, "  skipped non-directory: %s\n", entry.Name())

			continue
		}

		fmt.Fprintf(a.stdout, "  - %s\n", entry.Name())
		bundles++
	}

	if bundles == 0 {
		fmt.Fprintln(a.stdout, "  (no dotfile bundle directories found)")
	}

	fmt.Fprintf(a.stdout, "done: found %d dotfile %s\n", bundles, plural(bundles, "bundle", "bundles"))

	return nil
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}

func (a app) validateTemplate(opts options) error {
	loader := appconfig.Loader{Home: a.home, Repo: a.repo}

	if _, err := loader.Load(opts.configPath); err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "ok: apps.yaml at %s parses and validates\n", loader.Path(opts.configPath))

	secretsSvc := secrets.Service{Home: a.home, Repo: a.repo}

	if _, err := secretsSvc.Load(opts.secretsPath); err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "ok: secrets.yaml at %s parses and validates\n", secretsSvc.ConfigPath(opts.secretsPath))

	stowDir := filepath.Join(a.repo, "stow")
	info, err := os.Stat(stowDir)

	if err != nil {
		return fmt.Errorf("stow dir %s: %w", stowDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("stow path %s is not a directory", stowDir)
	}

	fmt.Fprintf(a.stdout, "ok: stow dir %s exists\n", stowDir)
	fmt.Fprintf(a.stdout, "ok: tracked Brewfile lists %d formulae and %d casks\n",
		len(brewfile.TrackedFormulae()), len(brewfile.TrackedCasks()))
	fmt.Fprintf(a.stdout, "ok: tracked macOS settings cover %d entries across %d domains\n",
		len(templatemacos.Settings()), len(templatemacos.Domains()))

	return nil
}

func (a app) inspectCurrentBrew(_ options) error {
	svc := brew.Service{Stdout: a.stdout, Runner: a.runner}
	formulae, err := svc.InstalledFormulae()

	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "# Installed Homebrew formulae (%d)\n", len(formulae))

	for _, name := range formulae {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	casks, err := svc.InstalledCasks()

	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "# Installed Homebrew casks (%d)\n", len(casks))

	for _, name := range casks {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	return nil
}

func (a app) inspectCurrentMacOS(_ options) error {
	fmt.Fprintln(a.stdout, "# Tracked macOS defaults domains (current values)")

	for _, domain := range templatemacos.Domains() {
		out, err := a.runner.Run("defaults", "read", domain)

		if err != nil {
			fmt.Fprintf(a.stdout, "  %s: read failed: %v\n", domain, err)

			continue
		}

		fmt.Fprintf(a.stdout, "## %s\n", domain)
		fmt.Fprintln(a.stdout, command.FirstLine(out))
	}

	return nil
}

func (a app) scanUntracked(opts options) (untrackedReport, error) {
	report := untrackedReport{}
	svc := brew.Service{Stdout: a.stdout, Runner: a.runner}

	installedFormulae, err := svc.InstalledFormulae()

	if err != nil {
		return report, err
	}

	report.Formulae = brew.Untracked(installedFormulae, brewfile.TrackedFormulae())

	installedCasks, err := svc.InstalledCasks()

	if err != nil {
		return report, err
	}

	report.Casks = brew.Untracked(installedCasks, brewfile.TrackedCasks())

	if opts.apps {
		appStore, err := a.appstore().UntrackedAppStore(appstore.Options{ConfigPath: opts.configPath, Apps: true})

		if err != nil {
			return report, err
		}

		report.AppStore = appStore
	}

	return report, nil
}

func (a app) reportUntracked(opts options) error {
	report, err := a.scanUntracked(opts)

	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "Untracked Homebrew formulae (%d):\n", len(report.Formulae))

	for _, name := range report.Formulae {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	fmt.Fprintf(a.stdout, "Untracked Homebrew casks (%d):\n", len(report.Casks))

	for _, name := range report.Casks {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	fmt.Fprintf(a.stdout, "Untracked App Store apps (%d):\n", len(report.AppStore))

	for _, app := range report.AppStore {
		fmt.Fprintf(a.stdout, "  - %s (%s)\n", app.Name, app.ID)
	}

	return nil
}

func (a app) removeUntrackedBrew(opts options) error {
	report, err := a.scanUntracked(opts)

	if err != nil {
		return err
	}

	svc := brew.Service{Stdout: a.stdout, Runner: a.runner}

	for _, name := range report.Formulae {
		if err := svc.Uninstall(brew.KindFormula, name, opts.dryRun); err != nil {
			return err
		}
	}

	for _, name := range report.Casks {
		if err := svc.Uninstall(brew.KindCask, name, opts.dryRun); err != nil {
			return err
		}
	}

	return nil
}

func (a app) removeUntrackedAppStore(opts options) error {
	if !opts.apps {
		fmt.Fprintln(a.stdout, "skipped: run with --apps to inspect Mac App Store removals")

		return nil
	}

	report, err := a.scanUntracked(opts)

	if err != nil {
		return err
	}

	if len(report.AppStore) == 0 {
		fmt.Fprintln(a.stdout, "no untracked App Store apps detected")

		return nil
	}

	if !opts.allowMasUninstall {
		fmt.Fprintln(a.stdout, "mas uninstall is gated off. Open Finder or App Store and remove these manually:")

		for _, app := range report.AppStore {
			fmt.Fprintf(a.stdout, "  - %s (%s)\n", app.Name, app.ID)
		}

		return nil
	}

	manualCleanup := []appstore.AppStoreApp{}

	for _, app := range report.AppStore {
		if err := a.appstore().UninstallAppStore(app, opts.dryRun); err != nil {
			fmt.Fprintf(a.stdout, "warning: %v\n", err)
			manualCleanup = append(manualCleanup, app)
		}
	}

	if len(manualCleanup) > 0 {
		fmt.Fprintf(a.stdout, "manual cleanup needed for %d App Store apps:\n", len(manualCleanup))

		for _, app := range manualCleanup {
			fmt.Fprintf(a.stdout, "  - %s (%s)\n", app.Name, app.ID)
		}
	}

	return nil
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
		fmt.Fprintf(a.stdout, "would validate 1Password CLI access: op vault list --format=json (and op signin if needed)\n")
		fmt.Fprintf(a.stdout, "would decrypt private secrets from 1Password item %q in vault %q\n", opts.opItem, opts.opVault)

		return svc.Decrypt(secretOpts)
	}

	if err := a.ensureOpSession(); err != nil {
		return err
	}

	return svc.Decrypt(secretOpts)
}
