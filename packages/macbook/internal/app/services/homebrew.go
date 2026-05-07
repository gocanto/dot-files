package services

import (
	"fmt"
	"os"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/template/brewfile"
)

func (s Service) ApplyHomebrewBundle(opts Options) error {
	brewfileFile, err := os.CreateTemp("", "api-*.Brewfile")

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

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	logFile, err := os.CreateTemp("", "api-homebrew-bundle-*.log")

	if err != nil {
		return fmt.Errorf("create Homebrew bundle log: %w", err)
	}

	logPath := logFile.Name()

	defer logFile.Close()

	fmt.Fprintf(s.Stdout, "logging full output to %s\n", logPath)

	out, runErr := s.Runner.Run(cmd[0], cmd[1:]...)

	if _, writeErr := logFile.Write(out); writeErr != nil {
		fmt.Fprintf(s.Stdout, "warning: could not write log file: %v\n", writeErr)
	}

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if runErr != nil {
		return fmt.Errorf("brew bundle failed (full log: %s): %w", logPath, runErr)
	}

	return nil
}
