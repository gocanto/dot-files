package app

import (
	"fmt"

	"github.com/gocanto/mac-os/internal/command"
	convergemacos "github.com/gocanto/mac-os/internal/converge/macos"
)

func (a app) applyMacOSDefaults(opts options) error {
	return convergemacos.Service{Runner: a.runner, Stdout: a.stdout, Stderr: a.stderr}.Apply(opts.dryRun)
}

func (a app) openEraseAssistant(dryRun bool) error {
	fmt.Fprintln(a.stdout, "Erase first selected.")
	fmt.Fprintln(a.stdout, "Use Apple's Erase Assistant: System Settings > General > Transfer or Reset > Erase All Content and Settings.")
	fmt.Fprintln(a.stdout, "Factory install will stop now. Run this tool again after the Mac returns to setup or after you decide to proceed without erasing.")

	cmd := []string{"open", "x-apple.systempreferences:com.apple.Transfer-Reset-Settings.extension"}

	if dryRun {
		fmt.Fprintf(a.stdout, "would open reset settings: %s\n", command.ShellQuote(cmd))

		return nil
	}

	if a.goos != "darwin" {
		fmt.Fprintf(a.stdout, "skipped opening reset settings: current OS is %s\n", a.goos)

		return nil
	}

	fmt.Fprintf(a.stdout, "opening reset settings: %s\n", command.ShellQuote(cmd))

	out, err := a.runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(a.stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("open Erase Assistant settings: %w", err)
	}

	return nil
}
