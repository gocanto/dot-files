package app

import (
	"fmt"
	"os"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/template/brewfile"
)

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
