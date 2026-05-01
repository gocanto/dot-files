package doctor

import (
	"fmt"
	"io"
	"os/exec"
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
	GOOS   string
	Repo   string
	Stdout io.Writer
	Runner command.Runner
}

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

		return nil
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
