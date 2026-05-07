package macos

import (
	"fmt"
	"io"

	"github.com/gocanto/dot-files/internal/command"
	templatemacos "github.com/gocanto/dot-files/internal/template/macos"
)

type Service struct {
	Runner command.Runner
	Stdout io.Writer
	Stderr io.Writer
}

func (s Service) Apply(dryRun bool) error {
	for _, setting := range templatemacos.Settings() {
		cmd := append([]string{"defaults", "write", setting.Domain, setting.Key}, setting.Args...)

		if dryRun {
			fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

			continue
		}

		out, err := s.Runner.Run(cmd[0], cmd[1:]...)

		if len(out) > 0 {
			fmt.Fprint(s.Stdout, string(out))
		}

		if err != nil {
			return fmt.Errorf("%s: %w", command.ShellQuote(cmd), err)
		}
	}

	for _, cmd := range [][]string{
		{"killall", "Finder"},
		{"killall", "Dock"},
		{"killall", "SystemUIServer"},
	} {
		if dryRun {
			fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

			continue
		}

		_, _ = s.Runner.Run(cmd[0], cmd[1:]...)
	}

	return nil
}
