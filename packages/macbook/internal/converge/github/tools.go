package github

import (
	"fmt"
	"os/exec"

	"github.com/gocanto/mac-os/internal/command"
)

var commandLookPath = exec.LookPath

func (s Service) ensureTools(dryRun bool) error {
	for _, tool := range []struct {
		name    string
		formula string
	}{
		{name: "git", formula: "git"},
		{name: "gh", formula: "gh"},
		{name: "gpg", formula: "gnupg"},
	} {
		if s.commandExists(tool.name) {
			continue
		}

		cmd := []string{"brew", "install", tool.formula}

		if dryRun {
			fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

			continue
		}

		out, err := s.Runner.Run(cmd[0], cmd[1:]...)

		if len(out) > 0 {
			fmt.Fprint(s.Stdout, string(out))
		}

		if err != nil {
			return fmt.Errorf("install %s: %w", tool.formula, err)
		}
	}

	return nil
}

func (s Service) commandExists(name string) bool {
	_, err := commandLookPath(name)

	return err == nil
}
