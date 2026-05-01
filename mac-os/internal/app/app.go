package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type commandRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

type realRunner struct{}

type app struct {
	home   string
	repo   string
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
	runner commandRunner
}

type options struct {
	dryRun      bool
	yes         bool
	encrypt     bool
	apps        bool
	archiveRoot string
	archivePath string
	configPath  string
	opVault     string
	opItem      string
}

type macSetting struct {
	domain string
	key    string
	args   []string
}

type devTool struct {
	name        string
	versionArgs []string
}

type captureItem struct {
	source string
	target string
}

type appConfig struct {
	Apps []managedApp `mapstructure:"apps"`
}

type managedApp struct {
	Name              string          `mapstructure:"name"`
	BundleID          string          `mapstructure:"bundle_id"`
	InstallMethod     string          `mapstructure:"install_method"`
	Package           string          `mapstructure:"package"`
	ConfigMode        string          `mapstructure:"config_mode"`
	ConfigPaths       []appConfigPath `mapstructure:"config_paths"`
	OnePasswordFields []string        `mapstructure:"onepassword_fields"`
}

type appConfigPath struct {
	Source string `mapstructure:"source"`
	Target string `mapstructure:"target"`
}

const (
	defaultArchiveRoot = ".local/state/macos-settings-archives"
	defaultOPVault     = "Private"
	defaultOPItem      = "Mac Migration Archive"
)

func (realRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)

	return cmd.CombinedOutput()
}

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

	repo = findRepoRoot(repo)

	a := app{
		home:   home,
		repo:   repo,
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdin:  os.Stdin,
		runner: realRunner{},
	}

	if len(args) == 0 {
		a.usage()

		return 0
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
  mac-os adopt [--dry-run] [--yes]
  mac-os bootstrap [--archive PATH] [--apps] [--config PATH] [--dry-run] [--yes]
  mac-os capture [--apps] [--config PATH] [--archive-root PATH] [--encrypt] [--op-vault VAULT] [--op-item ITEM] [--dry-run] [--yes]
  mac-os restore --archive PATH [--apps] [--config PATH] [--dry-run] [--yes]
  mac-os doctor
  mac-os brewfile [--write PATH]
  mac-os macos [--dry-run] [--yes]

Commands:
  adopt      Import safe current dotfiles into the repo's Stow layout.
  bootstrap  Run prompted phases for tools, dotfiles, macOS defaults, capture, and doctor.
  capture    Save a private settings inventory outside the repo by default.
  restore    Restore allowlisted app configuration from a private archive.
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

	phases := []struct {
		name string
		run  func(options) error
	}{
		{"prerequisites", a.ensurePrerequisites},
		{"homebrew bundle", a.applyHomebrewBundle},
		{"app store apps", a.applyAppStoreApps},
		{"manual app report", a.reportManualApps},
		{"adopt safe dotfiles", a.adoptDotfiles},
		{"stow links", a.applyStow},
		{"app config restore", a.restoreAppConfigs},
		{"macOS defaults", a.applyMacOSDefaults},
		{"private archive capture", a.captureArchive},
		{"doctor", a.runDoctor},
	}

	for _, phase := range phases {
		if err := a.confirmAndRun(phase.name, opts, func() error { return phase.run(opts) }); err != nil {
			fmt.Fprintf(a.stderr, "%s failed: %v\n", phase.name, err)

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

	content := brewfileContent()

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

func (a app) ensurePrerequisites(opts options) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("mac-os only supports darwin, current OS is %s", runtime.GOOS)
	}

	steps := [][]string{
		{"xcode-select", "-p"},
		{"brew", "--version"},
	}

	for _, step := range steps {
		if opts.dryRun {
			fmt.Fprintf(a.stdout, "would run: %s\n", shellQuote(step))

			continue
		}

		out, err := a.runner.Run(step[0], step[1:]...)

		if err != nil {
			return fmt.Errorf("%s: %w\n%s", shellQuote(step), err, strings.TrimSpace(string(out)))
		}

		fmt.Fprintf(a.stdout, "%s ok\n", step[0])
	}

	return nil
}

func (a app) applyHomebrewBundle(opts options) error {
	brewfile := filepath.Join(a.repo, "Brewfile")

	if _, err := os.Stat(brewfile); err != nil {
		return fmt.Errorf("missing Brewfile at %s", brewfile)
	}

	cmd := []string{"brew", "bundle", "--file", brewfile}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would run: %s\n", shellQuote(cmd))

		return nil
	}

	out, err := a.runner.Run(cmd[0], cmd[1:]...)
	fmt.Fprint(a.stdout, string(out))

	return err
}

func (a app) applyAppStoreApps(opts options) error {
	if !opts.apps {
		fmt.Fprintln(a.stdout, "skipped: run with --apps to install App Store apps")

		return nil
	}

	cfg, err := a.loadAppConfig(opts.configPath)

	if err != nil {
		return err
	}

	for _, app := range cfg.Apps {
		if app.InstallMethod != "mas" {
			continue
		}

		cmd := []string{"mas", "install", app.Package}

		if opts.dryRun {
			fmt.Fprintf(a.stdout, "would run: %s # %s\n", shellQuote(cmd), app.Name)

			continue
		}

		out, err := a.runner.Run(cmd[0], cmd[1:]...)
		fmt.Fprint(a.stdout, string(out))

		if err != nil {
			return fmt.Errorf("install App Store app %q: %w", app.Name, err)
		}
	}

	return nil
}

func (a app) reportManualApps(opts options) error {
	if !opts.apps {
		fmt.Fprintln(a.stdout, "skipped: run with --apps to report manual apps")

		return nil
	}

	cfg, err := a.loadAppConfig(opts.configPath)

	if err != nil {
		return err
	}

	for _, app := range cfg.Apps {
		switch app.InstallMethod {
		case "manual":
			fmt.Fprintf(a.stdout, "manual install required: %s", app.Name)

			if app.Package != "" {
				fmt.Fprintf(a.stdout, " (%s)", app.Package)
			}

			fmt.Fprintln(a.stdout)
		case "brew":
			fmt.Fprintf(a.stdout, "brew-managed app: %s (%s)\n", app.Name, app.Package)
		case "system":
			fmt.Fprintf(a.stdout, "system app: %s\n", app.Name)
		}
	}

	return nil
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
			fmt.Fprintf(a.stdout, "would run: %s\n", shellQuote(cmd))

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
	for _, item := range adoptPlan(a.home, a.repo) {
		source := item.source
		target := item.target

		if opts.dryRun {
			fmt.Fprintf(a.stdout, "would import: %s -> %s\n", source, target)

			continue
		}

		if shouldSkipSensitive(source) {
			fmt.Fprintf(a.stdout, "skipped sensitive path: %s\n", source)

			continue
		}

		data, err := os.ReadFile(source)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(a.stdout, "missing, skipped: %s\n", source)

				continue
			}

			return err
		}

		data = sanitizeDotfile(source, a.home, data)

		if err := writeFile(target, data, 0o600); err != nil {
			return err
		}

		fmt.Fprintf(a.stdout, "imported: %s\n", target)
	}

	return nil
}

func (a app) applyMacOSDefaults(opts options) error {
	for _, setting := range macOSDefaults() {
		cmd := append([]string{"defaults", "write", setting.domain, setting.key}, setting.args...)

		if opts.dryRun {
			fmt.Fprintf(a.stdout, "would run: %s\n", shellQuote(cmd))

			continue
		}

		out, err := a.runner.Run(cmd[0], cmd[1:]...)

		if len(out) > 0 {
			fmt.Fprint(a.stdout, string(out))
		}

		if err != nil {
			return fmt.Errorf("%s: %w", shellQuote(cmd), err)
		}
	}

	restarts := [][]string{
		{"killall", "Finder"},
		{"killall", "Dock"},
		{"killall", "SystemUIServer"},
	}

	for _, cmd := range restarts {
		if opts.dryRun {
			fmt.Fprintf(a.stdout, "would run: %s\n", shellQuote(cmd))

			continue
		}

		_, _ = a.runner.Run(cmd[0], cmd[1:]...)
	}

	return nil
}

func (a app) captureArchive(opts options) error {
	root, err := a.resolveArchiveRoot(opts)

	if err != nil {
		return err
	}

	stamp := time.Now().Format("20060102-150405")
	dest := filepath.Join(root, stamp)
	fmt.Fprintf(a.stdout, "archive destination: %s\n", dest)

	if opts.dryRun {
		for _, item := range capturePlan() {
			fmt.Fprintf(a.stdout, "would capture: %s -> %s\n", item.source, item.target)
		}

		if opts.apps {
			cfg, err := a.loadAppConfig(opts.configPath)

			if err != nil {
				return err
			}

			for _, item := range appCapturePlan(cfg) {
				fmt.Fprintf(a.stdout, "would capture app config: %s -> %s\n", item.source, item.target)
			}
		}

		for _, domain := range defaultsDomains {
			fmt.Fprintf(a.stdout, "would export defaults domain: %s\n", domain)
		}

		if opts.encrypt {
			fmt.Fprintf(a.stdout, "would read 1Password item: %s/%s\n", opts.opVault, opts.opItem)
			fmt.Fprintf(a.stdout, "would encrypt archive with Age recipient from 1Password\n")
			fmt.Fprintf(a.stdout, "would update 1Password latest_archive metadata\n")
		}

		return nil
	}

	if err := os.MkdirAll(dest, 0o700); err != nil {
		return err
	}

	if err := a.writeManifest(dest); err != nil {
		return err
	}

	if err := a.writeCommandOutput(dest, "system/sw_vers.txt", "sw_vers"); err != nil {
		return err
	}

	if err := a.writeCommandOutput(dest, "system/uname.txt", "uname", "-a"); err != nil {
		return err
	}

	if err := a.writeCommandOutput(dest, "brew/leaves.txt", "brew", "leaves"); err != nil {
		return err
	}

	if err := a.writeCommandOutput(dest, "brew/casks.txt", "brew", "list", "--cask"); err != nil {
		return err
	}

	if err := a.writeCommandOutput(dest, "brew/bundle-dump.txt", "brew", "bundle", "dump", "--file=-"); err != nil {
		fmt.Fprintf(a.stderr, "warning: brew bundle dump failed: %v\n", err)
	}

	if err := a.writeCommandOutput(dest, "apps/applications.txt", "find", "/Applications", "-maxdepth", "2", "-name", "*.app", "-print"); err != nil {
		fmt.Fprintf(a.stderr, "warning: application inventory failed: %v\n", err)
	}

	if err := a.writeCommandOutput(dest, "launch/agents-daemons.txt", "sh", "-c", `find "$HOME/Library/LaunchAgents" /Library/LaunchAgents /Library/LaunchDaemons -maxdepth 1 -type f -name '*.plist' -print 2>/dev/null | sort`); err != nil {
		fmt.Fprintf(a.stderr, "warning: launch inventory failed: %v\n", err)
	}

	if err := a.writeToolVersions(dest); err != nil {
		return err
	}

	if err := a.copySafeFiles(dest); err != nil {
		return err
	}

	if opts.apps {
		if err := a.copyAppConfigFiles(dest, opts); err != nil {
			return err
		}
	}

	if err := a.exportDefaults(dest); err != nil {
		return err
	}

	if opts.encrypt {
		encryptedPath, err := a.encryptArchive(dest, root, opts)

		if err != nil {
			return err
		}

		if err := a.updateArchiveMetadata(encryptedPath, root, opts); err != nil {
			return err
		}

		fmt.Fprintf(a.stdout, "encrypted archive at %s\n", encryptedPath)
	}

	fmt.Fprintf(a.stdout, "captured archive at %s\n", dest)

	return nil
}

func (a app) resolveArchiveRoot(opts options) (string, error) {
	root := opts.archiveRoot

	if root == "" && opts.encrypt {
		fields, err := a.onePasswordFields(opts)

		if err == nil {
			root = fields["archive_root"]
		}
	}

	if root == "" {
		root = filepath.Join(a.home, defaultArchiveRoot)
	}

	if strings.HasPrefix(root, "~/") {
		root = filepath.Join(a.home, strings.TrimPrefix(root, "~/"))
	}

	return root, nil
}

func (a app) encryptArchive(sourceDir, archiveRoot string, opts options) (string, error) {
	fields, err := a.onePasswordFields(opts)

	if err != nil {
		return "", err
	}

	recipient := strings.TrimSpace(fields["archive_age_recipient"])

	if recipient == "" {
		return "", fmt.Errorf("missing archive_age_recipient in 1Password item %q", opts.opItem)
	}

	name := filepath.Base(sourceDir) + ".tar.gz.age"
	target := filepath.Join(archiveRoot, name)
	cmd := fmt.Sprintf("tar -C %s -czf - . | age -r %s -o %s", shellQuote([]string{sourceDir}), shellQuote([]string{recipient}), shellQuote([]string{target}))
	out, err := a.runner.Run("sh", "-c", cmd)

	if len(out) > 0 {
		fmt.Fprint(a.stdout, string(out))
	}

	if err != nil {
		return "", fmt.Errorf("encrypt archive: %w", err)
	}

	return target, nil
}

func (a app) updateArchiveMetadata(encryptedPath, archiveRoot string, opts options) error {
	args := []string{
		"item", "edit", opts.opItem,
		"--vault", opts.opVault,
		"archive_root=" + archiveRoot,
		"latest_archive=" + encryptedPath,
	}

	out, err := a.runner.Run("op", args...)

	if len(out) > 0 {
		fmt.Fprint(a.stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("update 1Password archive metadata: %w", err)
	}

	return nil
}

func (a app) onePasswordFields(opts options) (map[string]string, error) {
	out, err := a.runner.Run("op", "item", "get", opts.opItem, "--vault", opts.opVault, "--format", "json")

	if err != nil {
		return nil, fmt.Errorf("read 1Password item %q in vault %q: %w", opts.opItem, opts.opVault, err)
	}

	var item struct {
		Fields []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"fields"`
	}

	if err := json.Unmarshal(out, &item); err != nil {
		return nil, fmt.Errorf("parse 1Password item JSON: %w", err)
	}

	fields := make(map[string]string, len(item.Fields))

	for _, field := range item.Fields {
		if field.ID != "" {
			fields[field.ID] = field.Value
		}

		if field.Label != "" {
			fields[field.Label] = field.Value
		}
	}

	return fields, nil
}

func (a app) writeManifest(dest string) error {
	content := `# macOS Settings Archive

This archive is private machine inventory, not a replay script.

Captured:
- OS, Homebrew, app, launch agent, and developer tool inventories.
- Selected safe dotfiles and plain-text configuration.
- Curated defaults exports for reference.

Skipped or redacted:
- SSH private keys, GPG keyrings, shell histories, API tokens, auth files.
- Browser/app caches, sessions, Claude/Codex file history, machine IDs.
- Docker VM data, database data directories, sockets, and generated state.
`

	return writeFile(filepath.Join(dest, "MANIFEST.md"), []byte(content), 0o600)
}

func (a app) writeCommandOutput(root, rel, name string, args ...string) error {
	out, err := a.runner.Run(name, args...)

	if err != nil {
		return fmt.Errorf("%s: %w\n%s", shellQuote(append([]string{name}, args...)), err, strings.TrimSpace(string(out)))
	}

	return writeFile(filepath.Join(root, rel), out, 0o600)
}

func (a app) writeToolVersions(root string) error {
	var b strings.Builder

	for _, tool := range devTools {
		path, _ := exec.LookPath(tool.name)
		fmt.Fprintf(&b, "## %s\n", tool.name)

		if path == "" {
			fmt.Fprintln(&b, "missing")
			fmt.Fprintln(&b)

			continue
		}

		fmt.Fprintf(&b, "path: %s\n", path)
		out, err := a.runner.Run(tool.name, tool.versionArgs...)

		if err != nil {
			fmt.Fprintf(&b, "version error: %v\n%s\n\n", err, strings.TrimSpace(string(out)))

			continue
		}

		fmt.Fprintf(&b, "%s\n", strings.TrimSpace(string(out)))
		fmt.Fprintln(&b)
	}

	return writeFile(filepath.Join(root, "dev-tools/versions.md"), []byte(b.String()), 0o600)
}

func (a app) copySafeFiles(root string) error {
	for _, item := range capturePlan() {
		source := item.source

		if strings.HasPrefix(source, "~/") {
			source = filepath.Join(a.home, strings.TrimPrefix(source, "~/"))
		}

		info, err := os.Stat(source)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return err
		}

		target := filepath.Join(root, item.target)

		if info.IsDir() {
			if err := copyDirSafe(source, target); err != nil {
				return err
			}

			continue
		}

		if shouldSkipSensitive(source) {
			continue
		}

		data, err := os.ReadFile(source)

		if err != nil {
			return err
		}

		data = sanitizeDotfile(source, a.home, data)

		if err := writeFile(target, data, 0o600); err != nil {
			return err
		}
	}

	return nil
}

func (a app) exportDefaults(root string) error {
	for _, domain := range defaultsDomains {
		out, err := a.runner.Run("defaults", "export", domain, "-")

		if err != nil {
			fmt.Fprintf(a.stderr, "warning: defaults export %s failed: %v\n", domain, err)

			continue
		}

		name := strings.ReplaceAll(domain, "/", "_") + ".plist"

		if err := writeFile(filepath.Join(root, "defaults", name), out, 0o600); err != nil {
			return err
		}
	}

	return nil
}

func (a app) copyAppConfigFiles(root string, opts options) error {
	cfg, err := a.loadAppConfig(opts.configPath)

	if err != nil {
		return err
	}

	if err := copyFile(a.appConfigPath(opts.configPath), filepath.Join(root, "apps/apps.yaml")); err != nil {
		return err
	}

	for _, item := range appCapturePlan(cfg) {
		source := expandHome(item.source, a.home)

		if shouldSkipSensitive(source) || shouldSkipSensitive(item.target) {
			fmt.Fprintf(a.stdout, "skipped sensitive app config: %s\n", item.source)

			continue
		}

		info, err := os.Stat(source)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(a.stdout, "missing app config, skipped: %s\n", item.source)

				continue
			}

			return err
		}

		target := filepath.Join(root, item.target)

		if info.IsDir() {
			if err := copyDirSafe(source, target); err != nil {
				return err
			}

			fmt.Fprintf(a.stdout, "captured app config: %s\n", item.target)

			continue
		}

		data, err := os.ReadFile(source)

		if err != nil {
			return err
		}

		data = sanitizeDotfile(source, a.home, data)

		if err := writeFile(target, data, 0o600); err != nil {
			return err
		}

		fmt.Fprintf(a.stdout, "captured app config: %s\n", item.target)
	}

	return nil
}

func (a app) restoreAppConfigs(opts options) error {
	if !opts.apps {
		fmt.Fprintln(a.stdout, "skipped: run with --apps to restore app configs")

		return nil
	}

	if opts.archivePath == "" {
		fmt.Fprintln(a.stdout, "skipped: no --archive supplied for app config restore")

		return nil
	}

	cfg, err := a.loadAppConfig(opts.configPath)

	if err != nil {
		return err
	}

	archive := expandHome(opts.archivePath, a.home)

	for _, app := range cfg.Apps {
		switch app.ConfigMode {
		case "auto":
			for _, path := range app.ConfigPaths {
				source := filepath.Join(archive, path.Target)
				target := expandHome(path.Source, a.home)

				if shouldSkipSensitive(source) || shouldSkipSensitive(target) {
					fmt.Fprintf(a.stdout, "skipped sensitive restore path: %s\n", path.Source)

					continue
				}

				info, err := os.Stat(source)

				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						fmt.Fprintf(a.stdout, "missing archive config, skipped: %s\n", path.Target)

						continue
					}

					return err
				}

				if opts.dryRun {
					fmt.Fprintf(a.stdout, "would restore app config: %s -> %s\n", source, target)

					continue
				}

				if info.IsDir() {
					if err := copyDirSafe(source, target); err != nil {
						return err
					}
				} else if err := copyFile(source, target); err != nil {
					return err
				}

				fmt.Fprintf(a.stdout, "restored app config: %s\n", target)
			}
		case "reference":
			fmt.Fprintf(a.stdout, "reference only: %s\n", app.Name)
		case "manual":
			fmt.Fprintf(a.stdout, "manual config restore: %s\n", app.Name)
		}
	}

	return nil
}

func (a app) runDoctor(options) error {
	if runtime.GOOS != "darwin" {
		fmt.Fprintf(a.stdout, "OS: %s (unsupported)\n", runtime.GOOS)
	} else {
		fmt.Fprintln(a.stdout, "OS: darwin")
	}

	required := []string{"brew", "git", "stow", "op", "age", "mas"}

	for _, name := range required {
		path, err := exec.LookPath(name)

		if err != nil {
			fmt.Fprintf(a.stdout, "missing: %s\n", name)

			continue
		}

		fmt.Fprintf(a.stdout, "found: %s -> %s\n", name, path)
	}

	fmt.Fprintln(a.stdout, "\nDeveloper tools:")

	for _, tool := range devTools {
		path, err := exec.LookPath(tool.name)

		if err != nil {
			fmt.Fprintf(a.stdout, "  %-14s missing\n", tool.name)

			continue
		}

		out, err := a.runner.Run(tool.name, tool.versionArgs...)
		version := strings.TrimSpace(firstLine(out))

		if err != nil {
			version = "version check failed"
		}

		fmt.Fprintf(a.stdout, "  %-14s %s (%s)\n", tool.name, version, path)
	}

	a.printOnePasswordArchiveStatus(defaultOPVault, defaultOPItem)

	return nil
}

func (a app) printOnePasswordArchiveStatus(vault, item string) {
	fmt.Fprintln(a.stdout, "\nPrivate archive:")

	if _, err := exec.LookPath("op"); err != nil {
		fmt.Fprintln(a.stdout, "  1Password CLI missing")

		return
	}

	if out, err := a.runner.Run("op", "account", "list"); err != nil {
		fmt.Fprintf(a.stdout, "  1Password account unavailable: %s\n", strings.TrimSpace(string(out)))

		return
	}

	fmt.Fprintln(a.stdout, "  1Password account available")

	if _, err := a.onePasswordFields(options{opVault: vault, opItem: item}); err != nil {
		fmt.Fprintf(a.stdout, "  archive item missing or unreadable: %v\n", err)

		return
	}

	fmt.Fprintf(a.stdout, "  archive item found: %s/%s\n", vault, item)
}

func macOSDefaults() []macSetting {
	return []macSetting{
		{"NSGlobalDomain", "AppleInterfaceStyle", []string{"-string", "Dark"}},
		{"NSGlobalDomain", "AppleShowAllExtensions", []string{"-bool", "true"}},
		{"NSGlobalDomain", "ApplePressAndHoldEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticDashSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticQuoteSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticPeriodSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSNavPanelExpandedStateForSaveMode", []string{"-bool", "true"}},
		{"NSGlobalDomain", "PMPrintingExpandedStateForPrint", []string{"-bool", "true"}},
		{"com.apple.finder", "AppleShowAllFiles", []string{"-bool", "true"}},
		{"com.apple.finder", "ShowPathbar", []string{"-bool", "true"}},
		{"com.apple.finder", "ShowStatusBar", []string{"-bool", "true"}},
		{"com.apple.finder", "FXPreferredViewStyle", []string{"-string", "Nlsv"}},
		{"com.apple.finder", "_FXShowPosixPathInTitle", []string{"-bool", "true"}},
		{"com.apple.dock", "autohide", []string{"-bool", "true"}},
		{"com.apple.dock", "mineffect", []string{"-string", "scale"}},
		{"com.apple.dock", "minimize-to-application", []string{"-bool", "true"}},
		{"com.apple.screencapture", "type", []string{"-string", "png"}},
		{"com.apple.screencapture", "disable-shadow", []string{"-bool", "true"}},
	}
}

var devTools = []devTool{
	{"git", []string{"--version"}},
	{"gh", []string{"--version"}},
	{"node", []string{"--version"}},
	{"npm", []string{"--version"}},
	{"pnpm", []string{"--version"}},
	{"yarn", []string{"--version"}},
	{"python3", []string{"--version"}},
	{"go", []string{"version"}},
	{"php", []string{"--version"}},
	{"composer", []string{"--version"}},
	{"mas", []string{"version"}},
	{"mysql", []string{"--version"}},
	{"psql", []string{"--version"}},
	{"docker", []string{"--version"}},
	{"claude", []string{"--version"}},
	{"codex", []string{"--version"}},
	{"opencode", []string{"--version"}},
	{"agent-browser", []string{"--version"}},
	{"op", []string{"--version"}},
	{"age", []string{"--version"}},
}

func (a app) loadAppConfig(path string) (appConfig, error) {
	configPath := a.appConfigPath(path)
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return appConfig{}, fmt.Errorf("read app config %s: %w", configPath, err)
	}

	var cfg appConfig

	if err := v.Unmarshal(&cfg); err != nil {
		return appConfig{}, fmt.Errorf("parse app config %s: %w", configPath, err)
	}

	if err := validateAppConfig(cfg); err != nil {
		return appConfig{}, fmt.Errorf("validate app config %s: %w", configPath, err)
	}

	return cfg, nil
}

func (a app) appConfigPath(path string) string {
	if path == "" {
		return filepath.Join(a.repo, "apps.yaml")
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(a.home, strings.TrimPrefix(path, "~/"))
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(a.repo, path)
}

func validateAppConfig(cfg appConfig) error {
	if len(cfg.Apps) == 0 {
		return errors.New("apps must contain at least one app")
	}

	installModes := map[string]bool{"brew": true, "mas": true, "manual": true, "system": true}
	configModes := map[string]bool{"auto": true, "reference": true, "manual": true}

	for i, app := range cfg.Apps {
		prefix := fmt.Sprintf("apps[%d]", i)

		if strings.TrimSpace(app.Name) == "" {
			return fmt.Errorf("%s.name is required", prefix)
		}

		if !installModes[app.InstallMethod] {
			return fmt.Errorf("%s.install_method %q is invalid", prefix, app.InstallMethod)
		}

		if !configModes[app.ConfigMode] {
			return fmt.Errorf("%s.config_mode %q is invalid", prefix, app.ConfigMode)
		}

		if (app.InstallMethod == "brew" || app.InstallMethod == "mas") && strings.TrimSpace(app.Package) == "" {
			return fmt.Errorf("%s.package is required for %s installs", prefix, app.InstallMethod)
		}

		if app.ConfigMode == "auto" && len(app.ConfigPaths) == 0 {
			return fmt.Errorf("%s.config_paths is required for auto config restore", prefix)
		}

		for j, path := range app.ConfigPaths {
			pathPrefix := fmt.Sprintf("%s.config_paths[%d]", prefix, j)

			if strings.TrimSpace(path.Source) == "" {
				return fmt.Errorf("%s.source is required", pathPrefix)
			}

			if strings.TrimSpace(path.Target) == "" {
				return fmt.Errorf("%s.target is required", pathPrefix)
			}

			if filepath.IsAbs(path.Target) || strings.HasPrefix(path.Target, "..") || strings.Contains(filepath.Clean(path.Target), "../") {
				return fmt.Errorf("%s.target must be archive-relative", pathPrefix)
			}
		}
	}

	return nil
}

func adoptPlan(home, repo string) []captureItem {
	return []captureItem{
		{filepath.Join(home, ".zshrc"), filepath.Join(repo, "stow/shell/.zshrc")},
		{filepath.Join(home, ".zprofile"), filepath.Join(repo, "stow/shell/.zprofile")},
		{filepath.Join(home, ".bash_profile"), filepath.Join(repo, "stow/shell/.bash_profile")},
		{filepath.Join(home, ".gitconfig"), filepath.Join(repo, "stow/git/.gitconfig")},
		{filepath.Join(home, ".vimrc"), filepath.Join(repo, "stow/vim/.vimrc")},
		{filepath.Join(home, ".config/git/ignore"), filepath.Join(repo, "stow/git/.config/git/ignore")},
		{filepath.Join(home, ".config/ghostty/config"), filepath.Join(repo, "stow/ghostty/.config/ghostty/config")},
	}
}

func appCapturePlan(cfg appConfig) []captureItem {
	items := []captureItem{}

	for _, app := range cfg.Apps {
		if app.ConfigMode == "manual" {
			continue
		}

		for _, path := range app.ConfigPaths {
			items = append(items, captureItem{source: path.Source, target: path.Target})
		}
	}

	return items
}

func capturePlan() []captureItem {
	return []captureItem{
		{"~/.zshrc", "dotfiles/.zshrc"},
		{"~/.zprofile", "dotfiles/.zprofile"},
		{"~/.bash_profile", "dotfiles/.bash_profile"},
		{"~/.gitconfig", "dotfiles/.gitconfig"},
		{"~/.vimrc", "dotfiles/.vimrc"},
		{"~/.config/git/ignore", "dotfiles/.config/git/ignore"},
		{"~/.config/ghostty/config", "dotfiles/.config/ghostty/config"},
		{"~/.vscode/extensions/extensions.json", "editors/vscode/extensions.json"},
		{"~/Library/Application Support/Code/User/settings.json", "editors/vscode/settings.json"},
	}
}

var defaultsDomains = []string{
	"NSGlobalDomain",
	"com.apple.dock",
	"com.apple.finder",
	"com.apple.screencapture",
	"com.apple.AppleMultitouchTrackpad",
	"com.apple.driver.AppleBluetoothMultitouch.trackpad",
	"com.mitchellh.ghostty",
	"com.googlecode.iterm2",
	"com.jordanbaird.Ice",
}

func brewfileContent() string {
	formulae := []string{
		"1password-cli",
		"age",
		"agent-browser",
		"autossh",
		"bruno",
		"claude-code",
		"codex",
		"csvlens",
		"fd",
		"ffmpeg",
		"fzf",
		"gh",
		"git",
		"glow",
		"gnupg",
		"jq",
		"libavif",
		"libpq",
		"mas",
		"mysql",
		"nginx",
		"node@24",
		"opencode",
		"pinentry-mac",
		"portless",
		"sevenzip",
		"stow",
		"vim",
		"yazi",
		"zsh-syntax-highlighting",
	}
	casks := []string{
		"1password",
		"bruno",
		"claude",
		"codex",
		"codexbar",
		"dbeaver-community",
		"discord",
		"docker",
		"ghostty",
		"google-chrome",
		"iterm2",
		"jetbrains-toolbox",
		"jordanbaird-ice",
		"latest",
		"linearmouse",
		"markedit",
		"microsoft-teams",
		"notion",
		"postman",
		"raycast",
		"spotify",
		"stats",
		"sublime-text",
		"visual-studio-code",
		"zoom",
	}

	var b strings.Builder

	fmt.Fprintln(&b, `tap "homebrew/bundle"`)
	fmt.Fprintln(&b)

	for _, name := range formulae {
		fmt.Fprintf(&b, "brew %s\n", strconv.Quote(name))
	}

	fmt.Fprintln(&b)

	for _, name := range casks {
		fmt.Fprintf(&b, "cask %s\n", strconv.Quote(name))
	}

	return b.String()
}

func writeFile(path string, content []byte, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, content, perm)
}

func copyFile(source, target string) error {
	data, err := os.ReadFile(source)

	if err != nil {
		return err
	}

	return writeFile(target, data, 0o600)
}

func copyDirSafe(source, target string) error {
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

		if shouldSkipSensitive(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		dst := filepath.Join(target, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0o700)
		}

		return copyFile(path, dst)
	})
}

func shouldSkipSensitive(path string) bool {
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

func sanitizeDotfile(path, home string, data []byte) []byte {
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

func shellQuote(parts []string) string {
	quoted := make([]string, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			quoted = append(quoted, "''")

			continue
		}

		if strings.ContainsAny(part, " \t\n\"'\\$`!*?[]{}()&;|<>") {
			quoted = append(quoted, "'"+strings.ReplaceAll(part, "'", `'\''`)+"'")

			continue
		}

		quoted = append(quoted, part)
	}

	return strings.Join(quoted, " ")
}

func firstLine(b []byte) string {
	line, _, _ := bytes.Cut(b, []byte("\n"))

	return string(line)
}

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	return path
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
