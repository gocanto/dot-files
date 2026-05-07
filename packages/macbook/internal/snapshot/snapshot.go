package snapshot

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/converge/appstore"
	"github.com/gocanto/dot-files/internal/currentstate/doctor"
	currentmacos "github.com/gocanto/dot-files/internal/currentstate/macos"
	"github.com/gocanto/dot-files/internal/safefs"
	"github.com/gocanto/dot-files/internal/template/appconfig"
	"github.com/gocanto/dot-files/internal/template/dotfiles"
	templatemacos "github.com/gocanto/dot-files/internal/template/macos"
)

type Options struct {
	DryRun      bool
	Encrypt     bool
	Apps        bool
	ArchiveRoot string
	ConfigPath  string
	OPVault     string
	OPItem      string
}

type Service struct {
	Home   string
	Repo   string
	Stdout io.Writer
	Stderr io.Writer
	Runner command.Runner
}

const DefaultRoot = ".local/state/macos-settings-archives"

func DefaultLocalRoot(home string) string {
	return filepath.Join(home, DefaultRoot)
}

func LatestLocalSnapshot(home string) (string, bool, error) {
	return LatestSnapshot(DefaultLocalRoot(home))
}

func LatestSnapshot(root string) (string, bool, error) {
	entries, err := os.ReadDir(root)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}

		return "", false, err
	}

	var latestName string

	var latestTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		timestamp, err := time.Parse("20060102-150405", entry.Name())

		if err != nil {
			continue
		}

		if latestName == "" || timestamp.After(latestTime) {
			latestName = entry.Name()
			latestTime = timestamp
		}
	}

	if latestName == "" {
		return "", false, nil
	}

	return filepath.Join(root, latestName), true, nil
}

func (s Service) Capture(opts Options) error {
	root, err := s.ResolveRoot(opts)

	if err != nil {
		return err
	}

	stamp := time.Now().Format("20060102-150405")
	dest := filepath.Join(root, stamp)
	fmt.Fprintf(s.Stdout, "archive destination: %s\n", dest)

	if opts.DryRun {
		for _, item := range dotfiles.CapturePlan() {
			fmt.Fprintf(s.Stdout, "would capture: %s -> %s\n", item.Source, item.Target)
		}

		if opts.Apps {
			cfg, err := appconfig.Loader{Home: s.Home, Repo: s.Repo}.Load(opts.ConfigPath)

			if err != nil {
				return err
			}

			for _, item := range appconfig.CapturePlan(cfg) {
				fmt.Fprintf(s.Stdout, "would capture app config: %s -> %s\n", item.Source, item.Target)
			}
		}

		for _, domain := range templatemacos.Domains() {
			fmt.Fprintf(s.Stdout, "would export defaults domain: %s\n", domain)
		}

		if opts.Encrypt {
			fmt.Fprintf(s.Stdout, "would read 1Password item: %s/%s\n", opts.OPVault, opts.OPItem)
			fmt.Fprintf(s.Stdout, "would encrypt archive with Age recipient from 1Password\n")
			fmt.Fprintf(s.Stdout, "would update 1Password latest_archive metadata\n")
		}

		return nil
	}

	if err := os.MkdirAll(dest, 0o700); err != nil {
		return err
	}

	if err := s.writeManifest(dest); err != nil {
		return err
	}

	if err := s.writeCommandOutput(dest, "system/sw_vers.txt", "sw_vers"); err != nil {
		return err
	}

	if err := s.writeCommandOutput(dest, "system/uname.txt", "uname", "-a"); err != nil {
		return err
	}

	if err := s.writeCommandOutput(dest, "brew/leaves.txt", "brew", "leaves"); err != nil {
		fmt.Fprintf(s.Stderr, "warning: brew leaves failed: %v\n", err)
	}

	if err := s.writeCommandOutput(dest, "brew/casks.txt", "brew", "list", "--cask"); err != nil {
		fmt.Fprintf(s.Stderr, "warning: brew list --cask failed: %v\n", err)
	}

	if err := s.writeCommandOutput(dest, "brew/bundle-dump.txt", "brew", "bundle", "dump", "--file=-"); err != nil {
		fmt.Fprintf(s.Stderr, "warning: brew bundle dump failed: %v\n", err)
	}

	if err := s.writeCommandOutput(dest, "applications.txt", "find", "/Applications", "-maxdepth", "2", "-name", "*.app", "-print"); err != nil {
		fmt.Fprintf(s.Stderr, "warning: application inventory failed: %v\n", err)
	}

	if err := s.writeCommandOutput(dest, "launch/agents-daemons.txt", "sh", "-c", `find "$HOME/Library/LaunchAgents" /Library/LaunchAgents /Library/LaunchDaemons -maxdepth 1 -type f -name '*.plist' -print 2>/dev/null | sort`); err != nil {
		fmt.Fprintf(s.Stderr, "warning: launch inventory failed: %v\n", err)
	}

	if err := s.writeToolVersions(dest); err != nil {
		return err
	}

	for _, item := range dotfiles.CapturePlan() {
		if err := safefs.CopyPlanItem(dest, s.Home, item); err != nil {
			return err
		}
	}

	if opts.Apps {
		appService := appstore.Service{Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}

		if err := appService.CaptureConfigs(dest, appstore.Options{ConfigPath: opts.ConfigPath}); err != nil {
			return err
		}
	}

	defaultsService := currentmacos.Service{Runner: s.Runner, Stdout: s.Stdout, Stderr: s.Stderr}

	if err := defaultsService.Export(dest); err != nil {
		return err
	}

	if opts.Encrypt {
		encryptedPath, err := s.Encrypt(dest, root, opts)

		if err != nil {
			return err
		}

		if err := s.UpdateMetadata(encryptedPath, root, opts); err != nil {
			return err
		}

		fmt.Fprintf(s.Stdout, "encrypted archive at %s\n", encryptedPath)
	}

	fmt.Fprintf(s.Stdout, "captured archive at %s\n", dest)

	return nil
}

func (s Service) ResolveRoot(opts Options) (string, error) {
	root := opts.ArchiveRoot

	if root == "" && opts.Encrypt {
		fields, err := command.OnePasswordFields(s.Runner, opts.OPVault, opts.OPItem)

		if err == nil {
			root = fields["archive_root"]
		}
	}

	if root == "" {
		root = filepath.Join(s.Home, DefaultRoot)
	}

	if strings.HasPrefix(root, "~/") {
		root = filepath.Join(s.Home, strings.TrimPrefix(root, "~/"))
	}

	return root, nil
}

func (s Service) Encrypt(sourceDir, archiveRoot string, opts Options) (string, error) {
	fields, err := command.OnePasswordFields(s.Runner, opts.OPVault, opts.OPItem)

	if err != nil {
		return "", err
	}

	recipient := strings.TrimSpace(fields["archive_age_recipient"])

	if recipient == "" {
		return "", fmt.Errorf("missing archive_age_recipient in 1Password item %q", opts.OPItem)
	}

	name := filepath.Base(sourceDir) + ".tar.gz.age"
	target := filepath.Join(archiveRoot, name)
	cmd := fmt.Sprintf("tar -C %s -czf - . | age -r %s -o %s", command.ShellQuote([]string{sourceDir}), command.ShellQuote([]string{recipient}), command.ShellQuote([]string{target}))
	out, err := s.Runner.Run("sh", "-c", cmd)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return "", fmt.Errorf("encrypt archive: %w", err)
	}

	return target, nil
}

func (s Service) UpdateMetadata(encryptedPath, archiveRoot string, opts Options) error {
	args := []string{
		"item", "edit", opts.OPItem,
		"--vault", opts.OPVault,
		"archive_root=" + archiveRoot,
		"latest_archive=" + encryptedPath,
	}

	out, err := s.Runner.Run("op", args...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("update 1Password archive metadata: %w", err)
	}

	return nil
}

func (s Service) writeManifest(dest string) error {
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

	return safefs.WriteFile(filepath.Join(dest, "MANIFEST.md"), []byte(content), 0o600)
}

func (s Service) writeCommandOutput(root, rel, name string, args ...string) error {
	out, err := s.Runner.Run(name, args...)

	if err != nil {
		return fmt.Errorf("%s: %w\n%s", command.ShellQuote(append([]string{name}, args...)), err, strings.TrimSpace(string(out)))
	}

	return safefs.WriteFile(filepath.Join(root, rel), out, 0o600)
}

func (s Service) writeToolVersions(root string) error {
	var b strings.Builder

	for _, tool := range doctor.DevTools() {
		path, _ := exec.LookPath(tool.Name)
		fmt.Fprintf(&b, "## %s\n", tool.Name)

		if path == "" {
			fmt.Fprintln(&b, "missing")
			fmt.Fprintln(&b)

			continue
		}

		fmt.Fprintf(&b, "path: %s\n", path)
		out, err := s.Runner.Run(tool.Name, tool.VersionArgs...)

		if err != nil {
			fmt.Fprintf(&b, "version error: %v\n%s\n\n", err, strings.TrimSpace(string(out)))

			continue
		}

		fmt.Fprintf(&b, "%s\n", strings.TrimSpace(string(out)))
		fmt.Fprintln(&b)
	}

	return safefs.WriteFile(filepath.Join(root, "dev-tools/versions.md"), []byte(b.String()), 0o600)
}
