package app

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gocanto/mac-os/internal/apps"
	"github.com/gocanto/mac-os/internal/archive"
	"github.com/gocanto/mac-os/internal/brewfile"
	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/doctor"
	"github.com/gocanto/mac-os/internal/dotfiles"
	"github.com/gocanto/mac-os/internal/macosdefaults"
	"github.com/gocanto/mac-os/internal/secrets"
	"github.com/gocanto/mac-os/internal/tui"
)

type app struct {
	home      string
	repo      string
	goos      string
	stdout    io.Writer
	stderr    io.Writer
	stdin     io.Reader
	runner    command.Runner
	tuiRunner func(io.Reader, io.Writer, []tui.Workflow) (tui.Result, error)
}

type options struct {
	dryRun       bool
	yes          bool
	encrypt      bool
	apps         bool
	archiveRoot  string
	archivePath  string
	configPath   string
	secretsPath  string
	secretTarget string
	opVault      string
	opItem       string
}

const (
	defaultOPVault = "Private"
	defaultOPItem  = "Mac Migration Archive"
)

func Run(args []string) int {
	home, err := os.UserHomeDir()

	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find home directory: %v\n", err)

		return 1
	}

	repo, err := os.Getwd()

	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot find working directory: %v\n", err)

		return 1
	}

	a := newApp(home, findRepoRoot(repo), os.Stdin, os.Stdout, os.Stderr, command.RealRunner{})

	return a.run(args)
}

func newApp(home, repo string, stdin io.Reader, stdout, stderr io.Writer, runner command.Runner) app {
	return app{
		home:      home,
		repo:      repo,
		goos:      runtime.GOOS,
		stdout:    stdout,
		stderr:    stderr,
		stdin:     stdin,
		runner:    runner,
		tuiRunner: tui.Run,
	}
}

func (a app) run(args []string) int {
	if len(args) == 0 {
		return a.tui(nil)
	}

	if args[0] != "secrets" && args[0] != "tui" {
		if err := a.requireSudo(); err != nil {
			fmt.Fprintf(a.stderr, "sudo access required: %v\n", err)

			return 1
		}
	}

	switch args[0] {
	case "bootstrap":
		return a.bootstrap(args[1:])
	case "adopt":
		return a.adopt(args[1:])
	case "capture":
		return a.capture(args[1:])
	case "restore":
		return a.restore(args[1:])
	case "doctor":
		return a.doctor(args[1:])
	case "brewfile":
		return a.brewfile(args[1:])
	case "macos":
		return a.macos(args[1:])
	case "secrets":
		return a.secrets(args[1:])
	case "tui":
		return a.tui(args[1:])
	case "help", "-h", "--help":
		a.usage()

		return 0
	default:
		fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.usage()

		return 2
	}
}

func (a app) usage() {
	fmt.Fprintln(a.stdout, `mac-os manages this machine's dotfiles, developer tools, and macOS settings.

Usage:
  mac-os
  mac-os tui
  mac-os adopt [--dry-run] [--yes]
  mac-os bootstrap [--archive PATH] [--apps] [--config PATH] [--dry-run] [--yes]
  mac-os capture [--apps] [--config PATH] [--archive-root PATH] [--encrypt] [--op-vault VAULT] [--op-item ITEM] [--dry-run] [--yes]
  mac-os restore --archive PATH [--apps] [--config PATH] [--dry-run] [--yes]
  mac-os secrets encrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets decrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets sync [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os doctor
  mac-os brewfile [--write PATH]
  mac-os macos [--dry-run] [--yes]

Commands:
  tui        Open the interactive Bubble Tea workflow dashboard.
  adopt      Import safe current dotfiles into the repo's Stow layout.
  bootstrap  Run prompted phases for tools, dotfiles, macOS defaults, capture, and doctor.
  capture    Save a private settings inventory outside the repo by default.
  restore    Restore allowlisted app configuration from a private archive.
  secrets    Manage encrypted private dotfile overlays with 1Password and Age.
  doctor     Print installed tool versions and missing prerequisites.
  brewfile   Print or write the curated Brewfile for this setup.
  macos      Apply curated macOS defaults only.`)
}

func (a app) bootstrap(args []string) int {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show commands without changing the machine")
	fs.BoolVar(&opts.yes, "yes", false, "run all phases without prompting")
	fs.BoolVar(&opts.apps, "apps", false, "include app install/config phases from apps.yaml")
	fs.StringVar(&opts.archivePath, "archive", "", "restore app configuration from this archive during bootstrap")
	fs.StringVar(&opts.configPath, "config", "", "app restore config path")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	for _, phase := range a.bootstrapPhases(opts) {
		if err := a.confirmAndRun(phase.Name, opts, func() error { return phase.Run(a.stdout) }); err != nil {
			fmt.Fprintf(a.stderr, "%s failed: %v\n", phase.Name, err)

			return 1
		}
	}

	return 0
}

func (a app) adopt(args []string) int {
	fs := flag.NewFlagSet("adopt", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show files without importing them")
	fs.BoolVar(&opts.yes, "yes", false, "import without prompting")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.confirmAndRun("adopt safe dotfiles", opts, func() error { return a.adoptDotfiles(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "adopt failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) capture(args []string) int {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show capture plan without writing files")
	fs.BoolVar(&opts.yes, "yes", false, "capture without prompting")
	fs.BoolVar(&opts.encrypt, "encrypt", false, "package and encrypt the archive with Age using 1Password metadata")
	fs.BoolVar(&opts.apps, "apps", false, "include allowlisted app configuration from apps.yaml")
	fs.StringVar(&opts.archiveRoot, "archive-root", "", "directory where timestamped archives are stored")
	fs.StringVar(&opts.configPath, "config", "", "app restore config path")
	fs.StringVar(&opts.opVault, "op-vault", defaultOPVault, "1Password vault containing archive metadata")
	fs.StringVar(&opts.opItem, "op-item", defaultOPItem, "1Password item containing archive metadata")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.confirmAndRun("private archive capture", opts, func() error { return a.captureArchive(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "capture failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) restore(args []string) int {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show restore plan without writing files")
	fs.BoolVar(&opts.yes, "yes", false, "restore without prompting")
	fs.BoolVar(&opts.apps, "apps", false, "restore allowlisted app configuration from apps.yaml")
	fs.StringVar(&opts.archivePath, "archive", "", "archive directory to restore from")
	fs.StringVar(&opts.configPath, "config", "", "app restore config path")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if opts.archivePath == "" {
		fmt.Fprintln(a.stderr, "restore requires --archive PATH")

		return 2
	}

	if !opts.apps {
		fmt.Fprintln(a.stderr, "restore currently requires --apps")

		return 2
	}

	if err := a.confirmAndRun("app config restore", opts, func() error { return a.restoreAppConfigs(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "restore failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) doctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.runDoctor(options{}); err != nil {
		fmt.Fprintf(a.stderr, "doctor failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) brewfile(args []string) int {
	fs := flag.NewFlagSet("brewfile", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	writePath := fs.String("write", "", "write Brewfile to this path")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	content := brewfile.Content()

	if *writePath == "" {
		fmt.Fprint(a.stdout, content)

		return 0
	}

	if err := os.WriteFile(*writePath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(a.stderr, "write Brewfile: %v\n", err)

		return 1
	}

	fmt.Fprintf(a.stdout, "wrote %s\n", *writePath)

	return 0
}

func (a app) macos(args []string) int {
	fs := flag.NewFlagSet("macos", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show defaults without applying")
	fs.BoolVar(&opts.yes, "yes", false, "apply without prompting")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.confirmAndRun("macOS defaults", opts, func() error { return a.applyMacOSDefaults(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "macOS defaults failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) secrets(args []string) int {
	if len(args) == 0 {
		a.secretsUsage()

		return 0
	}

	fs := flag.NewFlagSet("secrets "+args[0], flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show secret workflow without writing files")
	fs.StringVar(&opts.secretTarget, "target", "", "secret target name from secrets.yaml")
	fs.StringVar(&opts.secretsPath, "secrets-config", "", "secret manifest config path")
	fs.StringVar(&opts.opVault, "op-vault", defaultOPVault, "1Password vault containing secret metadata")
	fs.StringVar(&opts.opItem, "op-item", defaultOPItem, "1Password item containing secret metadata")

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	var err error

	switch args[0] {
	case "encrypt":
		err = a.encryptSecrets(opts)
	case "decrypt":
		err = a.decryptSecrets(opts)
	case "sync":
		err = a.syncSecrets(opts)
	case "encrypt-gitconfig":
		opts.secretTarget = secrets.GitconfigSecret
		err = a.encryptSecrets(opts)
	case "decrypt-gitconfig":
		opts.secretTarget = secrets.GitconfigSecret
		err = a.decryptSecrets(opts)
	case "sync-gitconfig":
		opts.secretTarget = secrets.GitconfigSecret
		err = a.syncSecrets(opts)
	case "help", "-h", "--help":
		a.secretsUsage()

		return 0
	default:
		fmt.Fprintf(a.stderr, "unknown secrets command %q\n\n", args[0])
		a.secretsUsage()

		return 2
	}

	if err != nil {
		fmt.Fprintf(a.stderr, "secrets %s failed: %v\n", args[0], err)

		return 1
	}

	return 0
}

func (a app) secretsUsage() {
	fmt.Fprintln(a.stdout, `Usage:
  mac-os secrets encrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets decrypt [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets sync [--target NAME] [--secrets-config PATH] [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets encrypt-gitconfig [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets decrypt-gitconfig [--op-vault VAULT] [--op-item ITEM] [--dry-run]
  mac-os secrets sync-gitconfig [--op-vault VAULT] [--op-item ITEM] [--dry-run]`)
}

func (a app) tui(args []string) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	result, err := a.tuiRunner(a.stdin, a.stdout, a.tuiWorkflows())

	if err != nil {
		fmt.Fprintf(a.stderr, "tui failed: %v\n", err)

		return 1
	}

	return result.ExitCode
}

func (a app) confirmAndRun(name string, opts options, fn func() error) error {
	fmt.Fprintf(a.stdout, "\n==> %s\n", name)

	if opts.dryRun {
		fmt.Fprintln(a.stdout, "dry-run mode: no changes will be applied")
	}

	if !opts.yes && !opts.dryRun {
		ok, err := a.confirm("Run this phase?")

		if err != nil {
			return err
		}

		if !ok {
			fmt.Fprintln(a.stdout, "skipped")

			return nil
		}
	}

	return fn()
}

func (a app) confirm(prompt string) (bool, error) {
	fmt.Fprintf(a.stdout, "%s [y/N] ", prompt)
	reader := bufio.NewReader(a.stdin)
	line, err := reader.ReadString('\n')

	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	answer := strings.ToLower(strings.TrimSpace(line))

	return answer == "y" || answer == "yes", nil
}

func (a app) requireSudo() error {
	out, err := a.runner.Run("sudo", "-v")

	if err != nil {
		message := strings.TrimSpace(string(out))

		if message != "" {
			return fmt.Errorf("run sudo -v: %w\n%s", err, message)
		}

		return fmt.Errorf("run sudo -v: %w", err)
	}

	return nil
}

func (a app) bootstrapPhases(opts options) []tui.Phase {
	return []tui.Phase{
		{Name: "prerequisites", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensurePrerequisites(opts) }},
		{Name: "homebrew bundle", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyHomebrewBundle(opts) }},
		{Name: "app store apps", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyAppStoreApps(opts) }},
		{Name: "manual app report", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).reportManualApps(opts) }},
		{Name: "adopt safe dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).adoptDotfiles(opts) }},
		{Name: "stow links", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyStow(opts) }},
		{Name: "app config restore", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(opts) }},
		{Name: "macOS defaults", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }},
		{Name: "private archive capture", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).captureArchive(opts) }},
		{Name: "doctor", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }},
	}
}

func (a app) tuiWorkflows() []tui.Workflow {
	dryRunOpts := options{dryRun: true, yes: true, opVault: defaultOPVault, opItem: defaultOPItem}

	return []tui.Workflow{
		{Name: "Bootstrap", Phases: a.bootstrapPhases(dryRunOpts)},
		{Name: "Capture Archive", Phases: []tui.Phase{{Name: "capture dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).captureArchive(dryRunOpts) }}}},
		{Name: "Restore App Configs", Phases: []tui.Phase{{Name: "restore app configs dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(options{dryRun: true, apps: true}) }}}},
		{Name: "Apply macOS Defaults", Phases: []tui.Phase{{Name: "macOS defaults dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(dryRunOpts) }}}},
		{Name: "Doctor", Phases: []tui.Phase{{Name: "doctor", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(dryRunOpts) }}}},
		{Name: "Brewfile Preview", Phases: []tui.Phase{{Name: "brewfile preview", Enabled: true, Run: func(w io.Writer) error { fmt.Fprint(w, brewfile.Content()); return nil }}}},
	}
}

func (a app) withStdout(stdout io.Writer) app {
	a.stdout = stdout

	return a
}

func (a app) ensurePrerequisites(opts options) error {
	return doctor.Service{GOOS: a.goos, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.EnsurePrerequisites(opts.dryRun)
}

func (a app) applyHomebrewBundle(opts options) error {
	brewfilePath := filepath.Join(a.repo, "Brewfile")

	if _, err := os.Stat(brewfilePath); err != nil {
		return fmt.Errorf("missing Brewfile at %s", brewfilePath)
	}

	cmd := []string{"brew", "bundle", "--file", brewfilePath}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	out, err := a.runner.Run(cmd[0], cmd[1:]...)
	fmt.Fprint(a.stdout, string(out))

	return err
}

func (a app) applyAppStoreApps(opts options) error {
	return a.apps().ApplyAppStore(apps.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
}

func (a app) reportManualApps(opts options) error {
	return a.apps().ReportManual(apps.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
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
	return doctor.Service{GOOS: a.goos, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.Run(defaultOPVault, defaultOPItem)
}

func (a app) encryptSecrets(opts options) error {
	return a.secretsService().Encrypt(secretOptions(opts))
}

func (a app) decryptSecrets(opts options) error {
	return a.secretsService().Decrypt(secretOptions(opts))
}

func (a app) syncSecrets(opts options) error {
	return a.secretsService().Sync(secretOptions(opts))
}

func (a app) apps() apps.Service {
	return apps.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}
}

func (a app) secretsService() secrets.Service {
	return secrets.Service{Repo: a.repo, Stdout: a.stdout, Runner: a.runner}
}

func secretOptions(opts options) secrets.Options {
	return secrets.Options{
		DryRun:       opts.dryRun,
		SecretsPath:  opts.secretsPath,
		SecretTarget: opts.secretTarget,
		OPVault:      opts.opVault,
		OPItem:       opts.opItem,
	}
}

func findRepoRoot(start string) string {
	if root, ok := walkForRepoRoot(start); ok {
		return root
	}

	exe, err := os.Executable()

	if err == nil {
		if root, ok := walkForRepoRoot(filepath.Dir(exe)); ok {
			return root
		}
	}

	return start
}

func walkForRepoRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)

	if err != nil {
		return start, false
	}

	for {
		if hasRepoMarkers(dir) {
			return dir, true
		}

		macOSDir := filepath.Join(dir, "mac-os")

		if hasRepoMarkers(macOSDir) {
			return macOSDir, true
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return start, false
		}

		dir = parent
	}
}

func hasRepoMarkers(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "Brewfile")); err != nil {
		return false
	}

	if info, err := os.Stat(filepath.Join(dir, "stow")); err != nil || !info.IsDir() {
		return false
	}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return false
	}

	return true
}
