package doctor

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/template/secrets"
)

func (s Service) Run(defaultOPVault, defaultOPItem, secretsPath string) error {
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

	s.printOnePasswordArchiveStatus(defaultOPVault, defaultOPItem, secretsPath)

	return nil
}

func (s Service) printOnePasswordArchiveStatus(vault, item, secretsPath string) {
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
	secrets.Service{Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}.PrintStatus(secrets.Options{SecretsPath: secretsPath, OPVault: vault, OPItem: item})
}
