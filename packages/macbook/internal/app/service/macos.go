package service

import (
	"fmt"

	"github.com/gocanto/dot-files/internal/command"
	convergemacos "github.com/gocanto/dot-files/internal/converge/macos"
)

func (s Service) ApplyMacOSDefaults(opts Options) error {
	return convergemacos.Service{Runner: s.Runner, Stdout: s.Stdout, Stderr: s.Stderr}.Apply(opts.DryRun)
}

func (s Service) OpenEraseAssistant(dryRun bool) error {
	fmt.Fprintln(s.Stdout, "Erase first selected.")
	fmt.Fprintln(s.Stdout, "Use Apple's Erase Assistant: System Settings > General > Transfer or Reset > Erase All Content and Settings.")
	fmt.Fprintln(s.Stdout, "Factory install will stop now. Run this tool again after the Mac returns to setup or after you decide to proceed without erasing.")

	cmd := []string{"open", "x-apple.systempreferences:com.apple.Transfer-Reset-Settings.extension"}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would open reset settings: %s\n", command.ShellQuote(cmd))

		return nil
	}

	if s.GOOS != "darwin" {
		fmt.Fprintf(s.Stdout, "skipped opening reset settings: current OS is %s\n", s.GOOS)

		return nil
	}

	fmt.Fprintf(s.Stdout, "opening reset settings: %s\n", command.ShellQuote(cmd))

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("open Erase Assistant settings: %w", err)
	}

	return nil
}
