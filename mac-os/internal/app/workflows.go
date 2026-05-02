package app

import (
	"fmt"
	"io"

	"github.com/gocanto/mac-os/internal/brewfile"
	"github.com/gocanto/mac-os/internal/tui"
)

func (a app) bootstrapPhases(opts options) []tui.Phase {
	return []tui.Phase{
		{Name: "prerequisites", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensurePrerequisites(opts) }},
		{Name: "homebrew bundle", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyHomebrewBundle(opts) }},
		{Name: "app store apps", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyAppStoreApps(opts) }},
		{Name: "manual app report", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).reportManualApps(opts) }},
		{Name: "adopt safe dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).adoptDotfiles(opts) }},
		{Name: "oh-my-zsh", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensureOhMyZsh(opts) }},
		{Name: "stow links", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyStow(opts) }},
		{Name: "app config restore", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(opts) }},
		{Name: "macOS defaults", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }},
		{Name: "private archive capture", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).captureArchive(opts) }},
		{Name: "doctor", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }},
	}
}

func (a app) factoryInstallPhases(opts options) []tui.Phase {
	return []tui.Phase{
		{Name: "prerequisites", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensurePrerequisites(opts) }},
		{Name: "homebrew bundle", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyHomebrewBundle(opts) }},
		{Name: "app store apps", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyAppStoreApps(opts) }},
		{Name: "manual app report", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).reportManualApps(opts) }},
		{Name: "private secrets restore", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restorePrivateSecrets(opts) }},
		{Name: "adopt safe dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).adoptDotfiles(opts) }},
		{Name: "oh-my-zsh", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensureOhMyZsh(opts) }},
		{Name: "stow links", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyStow(opts) }},
		{Name: "macOS defaults", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }},
		{Name: "doctor", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }},
	}
}

func (a app) tuiWorkflows() []tui.Workflow {
	dryRunOpts := options{dryRun: true, opVault: defaultOPVault, opItem: defaultOPItem}
	factoryOpts := options{apps: true, opVault: defaultOPVault, opItem: defaultOPItem}
	factoryDryRunOpts := options{dryRun: true, apps: true, opVault: defaultOPVault, opItem: defaultOPItem}

	return []tui.Workflow{
		{Name: "Factory Install", Confirmation: a.factoryInstallConfirmation(false), Phases: a.factoryInstallPhases(factoryOpts)},
		{Name: "Factory Install Dry Run", Confirmation: a.factoryInstallConfirmation(true), Phases: a.factoryInstallPhases(factoryDryRunOpts)},
		{Name: "Bootstrap", Phases: a.bootstrapPhases(dryRunOpts)},
		{Name: "Capture Archive", Phases: []tui.Phase{{Name: "capture dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).captureArchive(dryRunOpts) }}}},
		{Name: "Restore App Configs", Phases: []tui.Phase{{Name: "restore app configs dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(options{dryRun: true, apps: true}) }}}},
		{Name: "Apply macOS Defaults", Phases: []tui.Phase{{Name: "macOS defaults dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(dryRunOpts) }}}},
		{Name: "Doctor", Phases: []tui.Phase{{Name: "doctor", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(dryRunOpts) }}}},
		{Name: "Brewfile Preview", Phases: []tui.Phase{{Name: "brewfile preview", Enabled: true, Run: func(w io.Writer) error { fmt.Fprint(w, brewfile.Content()); return nil }}}},
	}
}

func (a app) factoryInstallConfirmation(dryRun bool) *tui.Confirmation {
	return &tui.Confirmation{
		Title:   "Confirm erase state",
		Message: "Factory Install is intended for a clean Mac. If this computer still has existing data, use Apple's Erase Assistant first. You can also continue when the Mac was already erased or when you intentionally want to rerun setup without erasing.",
		Options: []tui.ConfirmationOption{
			{
				Label:       "Erase first",
				Description: "open reset settings and stop",
				Continue:    false,
				Run: func(w io.Writer) error {
					return a.withStdout(w).openEraseAssistant(dryRun)
				},
			},
			{
				Label:       "Already erased",
				Description: "continue factory install",
				Continue:    true,
				Run: func(w io.Writer) error {
					fmt.Fprintln(w, "confirmed: Mac was already erased; continuing factory install")

					return nil
				},
			},
			{
				Label:       "Proceed without erase",
				Description: "continue and log skipped erase",
				Continue:    true,
				Run: func(w io.Writer) error {
					fmt.Fprintln(w, "confirmed: erase was intentionally skipped; continuing factory install")

					return nil
				},
			},
		},
	}
}

func (a app) withStdout(stdout io.Writer) app {
	a.stdout = stdout

	return a
}
