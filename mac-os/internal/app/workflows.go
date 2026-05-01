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
		{Name: "stow links", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyStow(opts) }},
		{Name: "app config restore", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(opts) }},
		{Name: "macOS defaults", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }},
		{Name: "private archive capture", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).captureArchive(opts) }},
		{Name: "doctor", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }},
	}
}

func (a app) tuiWorkflows() []tui.Workflow {
	dryRunOpts := options{dryRun: true, yes: true, opVault: defaultOPVault, opItem: defaultOPItem}

	return []tui.Workflow{
		{Name: "Bootstrap", Phases: a.bootstrapPhases(dryRunOpts)},
		{Name: "Capture Archive", Phases: []tui.Phase{{Name: "capture dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).captureArchive(dryRunOpts) }}}},
		{Name: "Restore App Configs", Phases: []tui.Phase{{Name: "restore app configs dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(options{dryRun: true, apps: true}) }}}},
		{Name: "Apply macOS Defaults", Phases: []tui.Phase{{Name: "macOS defaults dry-run", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(dryRunOpts) }}}},
		{Name: "Doctor", Phases: []tui.Phase{{Name: "doctor", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(dryRunOpts) }}}},
		{Name: "Brewfile Preview", Phases: []tui.Phase{{Name: "brewfile preview", Enabled: true, Run: func(w io.Writer) error { fmt.Fprint(w, brewfile.Content()); return nil }}}},
	}
}

func (a app) withStdout(stdout io.Writer) app {
	a.stdout = stdout

	return a
}
