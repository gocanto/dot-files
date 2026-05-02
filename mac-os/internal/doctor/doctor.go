package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/secrets"
)

type Tool struct {
	Name        string
	VersionArgs []string
}

type Service struct {
	GOOS         string
	GOARCH       string
	Home         string
	Repo         string
	Stdout       io.Writer
	Runner       command.Runner
	rosettaCheck func() bool
}

const rosettaMarker = "/Library/Apple/usr/share/rosetta/rosetta"
const dockerDesktopSettingsPath = "Library/Group Containers/group.com.docker/settings-store.json"

func DevTools() []Tool {
	return []Tool{
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
}

func (s Service) EnsurePrerequisites(dryRun bool) error {
	goos := s.GOOS

	if goos == "" {
		goos = runtime.GOOS
	}

	if goos != "darwin" {
		return fmt.Errorf("mac-os only supports darwin, current OS is %s", goos)
	}

	cmd := []string{"xcode-select", "-p"}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))
		fmt.Fprintln(s.Stdout, "would check Xcode Command Line Tools license status")

		return s.ensureAppleSiliconSupport(dryRun)
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if err != nil {
		return fmt.Errorf("Xcode Command Line Tools are missing or unusable; run `xcode-select --install`, complete Apple's installer, then rerun setup\n%s", strings.TrimSpace(string(out)))
	}

	fmt.Fprintf(s.Stdout, "%s ok\n", cmd[0])

	if out, err := s.Runner.Run("xcodebuild", "-license", "check"); err != nil {
		message := strings.TrimSpace(string(out))
		lower := strings.ToLower(message)

		if strings.Contains(lower, "license") || strings.Contains(lower, "agree") {
			return fmt.Errorf("Xcode Command Line Tools license needs attention; run `sudo xcodebuild -license` and accept Apple's prompts\n%s", message)
		}
	}

	return s.ensureAppleSiliconSupport(dryRun)
}

func (s Service) ensureAppleSiliconSupport(dryRun bool) error {
	if err := s.ensureRosetta(dryRun); err != nil {
		return err
	}

	return s.ensureDockerDesktopRosettaDisabled(dryRun)
}

func (s Service) ensureRosetta(dryRun bool) error {
	goarch := s.GOARCH

	if goarch == "" {
		goarch = runtime.GOARCH
	}

	if goarch != "arm64" {
		return nil
	}

	check := s.rosettaCheck

	if check == nil {
		check = func() bool {
			_, err := os.Stat(rosettaMarker)

			return err == nil
		}
	}

	if check() {
		fmt.Fprintln(s.Stdout, "Rosetta 2 found")

		return nil
	}

	cmd := []string{"softwareupdate", "--install-rosetta", "--agree-to-license"}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("install Rosetta 2 (Docker Desktop and other x86_64 binaries need it on Apple Silicon): %w", err)
	}

	return nil
}

func (s Service) ensureDockerDesktopRosettaDisabled(dryRun bool) error {
	goarch := s.GOARCH

	if goarch == "" {
		goarch = runtime.GOARCH
	}

	if goarch != "arm64" {
		return nil
	}

	home := s.Home

	if home == "" {
		var err error

		home, err = os.UserHomeDir()

		if err != nil {
			return fmt.Errorf("find home directory for Docker Desktop settings: %w", err)
		}
	}

	settingsPath := filepath.Join(home, dockerDesktopSettingsPath)
	settings := map[string]any{}

	if data, err := os.ReadFile(settingsPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse Docker Desktop settings at %s: %w", settingsPath, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read Docker Desktop settings at %s: %w", settingsPath, err)
	}

	if settings["UseVirtualizationFrameworkRosetta"] == false && settings["useVirtualizationFrameworkRosetta"] == false {
		fmt.Fprintln(s.Stdout, "Docker Desktop Rosetta integration disabled")

		return nil
	}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would disable Docker Desktop Rosetta integration in %s\n", settingsPath)

		return nil
	}

	settings["UseVirtualizationFrameworkRosetta"] = false
	settings["useVirtualizationFrameworkRosetta"] = false

	data, err := json.MarshalIndent(settings, "", "  ")

	if err != nil {
		return fmt.Errorf("encode Docker Desktop settings: %w", err)
	}

	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("create Docker Desktop settings directory: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		return fmt.Errorf("write Docker Desktop settings at %s: %w", settingsPath, err)
	}

	fmt.Fprintln(s.Stdout, "Docker Desktop Rosetta integration disabled")

	return nil
}

func (s Service) Run(defaultOPVault, defaultOPItem string) error {
	if runtime.GOOS != "darwin" {
		fmt.Fprintf(s.Stdout, "OS: %s (unsupported)\n", runtime.GOOS)
	} else {
		fmt.Fprintln(s.Stdout, "OS: darwin")
	}

	required := []string{"brew", "git", "stow", "op", "age", "mas"}

	for _, name := range required {
		path, err := exec.LookPath(name)

		if err != nil {
			fmt.Fprintf(s.Stdout, "missing: %s\n", name)

			continue
		}

		fmt.Fprintf(s.Stdout, "found: %s -> %s\n", name, path)
	}

	fmt.Fprintln(s.Stdout, "\nDeveloper tools:")

	for _, tool := range DevTools() {
		path, err := exec.LookPath(tool.Name)

		if err != nil {
			fmt.Fprintf(s.Stdout, "  %-14s missing\n", tool.Name)

			continue
		}

		out, err := s.Runner.Run(tool.Name, tool.VersionArgs...)
		version := strings.TrimSpace(command.FirstLine(out))

		if err != nil {
			version = "version check failed"
		}

		fmt.Fprintf(s.Stdout, "  %-14s %s (%s)\n", tool.Name, version, path)
	}

	s.printOnePasswordArchiveStatus(defaultOPVault, defaultOPItem)

	return nil
}

func (s Service) printOnePasswordArchiveStatus(vault, item string) {
	fmt.Fprintln(s.Stdout, "\nPrivate archive:")

	if _, err := exec.LookPath("op"); err != nil {
		fmt.Fprintln(s.Stdout, "  1Password CLI missing")

		return
	}

	if out, err := s.Runner.Run("op", "account", "list"); err != nil {
		fmt.Fprintf(s.Stdout, "  1Password account unavailable: %s\n", strings.TrimSpace(string(out)))

		return
	}

	fmt.Fprintln(s.Stdout, "  1Password account available")

	if _, err := command.OnePasswordFields(s.Runner, vault, item); err != nil {
		fmt.Fprintf(s.Stdout, "  archive item missing or unreadable: %v\n", err)

		return
	}

	fmt.Fprintf(s.Stdout, "  archive item found: %s/%s\n", vault, item)
	secrets.Service{Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}.PrintStatus(secrets.Options{OPVault: vault, OPItem: item})
}
