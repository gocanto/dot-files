package app

import (
	"fmt"
	"io"

	"github.com/gocanto/mac-os/internal/brewfile"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

func (a app) factoryInstallPhases(opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{
		{Name: "Check/install prerequisites", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensurePrerequisites(opts) }},
		{Name: "Install Homebrew packages", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyHomebrewBundle(opts) }},
		{Name: "Set up GitHub access and signing", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).setupGitHub(opts) }},
		{Name: "Install App Store apps", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyAppStoreApps(opts) }},
		{Name: "Show manual app install notes", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).reportManualApps(opts) }},
		{Name: "Restore private secrets from 1Password", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restorePrivateSecrets(opts) }},
		{Name: "Prepare existing dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).adoptDotfiles(opts) }},
		{Name: "Install oh-my-zsh", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensureOhMyZsh(opts) }},
		{Name: "Link dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyStow(opts) }},
		{Name: "Apply macOS settings", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }},
		{Name: "Run health checks", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }},
	}
}

func (a app) hostUpdatePhases(opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{
		{Name: "Check/install prerequisites", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensurePrerequisites(opts) }},
		{Name: "Install Homebrew packages", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyHomebrewBundle(opts) }},
		{Name: "Set up GitHub access and signing", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).setupGitHub(opts) }},
		{Name: "Install App Store apps", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyAppStoreApps(opts) }},
		{Name: "Show manual app install notes", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).reportManualApps(opts) }},
		{Name: "Restore private secrets from 1Password", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restorePrivateSecrets(opts) }},
		{Name: "Install oh-my-zsh", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensureOhMyZsh(opts) }},
		{Name: "Link dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyStow(opts) }},
		{Name: "Restore supported app configs from latest snapshot", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(opts) }},
		{Name: "Apply macOS settings", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }},
		{Name: "Run health checks", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }},
	}
}

func (a app) workflows() []workflowdomain.Workflow {
	dryRunOpts := options{dryRun: true, opVault: defaultOPVault, opItem: defaultOPItem}
	factoryOpts := options{apps: true, opVault: defaultOPVault, opItem: defaultOPItem}
	factoryDryRunOpts := options{dryRun: true, apps: true, opVault: defaultOPVault, opItem: defaultOPItem}
	hostUpdateOpts := options{apps: true, useLatestArchive: true, opVault: defaultOPVault, opItem: defaultOPItem}
	hostUpdateDryRunOpts := options{dryRun: true, apps: true, useLatestArchive: true, opVault: defaultOPVault, opItem: defaultOPItem}
	appDryRunOpts := options{dryRun: true, apps: true}
	appLiveOpts := options{apps: true}
	updateAppsDryRunOpts := options{dryRun: true}
	updateAppsOpts := options{}
	captureDryRunOpts := options{dryRun: true, apps: true, opVault: defaultOPVault, opItem: defaultOPItem}
	captureOpts := options{apps: true, opVault: defaultOPVault, opItem: defaultOPItem}

	return workflowdomain.Normalize([]workflowdomain.Workflow{
		{
			Name:         "Set Up This Mac",
			Description:  "Run the complete setup flow for this Mac: tools, apps, private secrets, dotfiles, macOS settings, and health checks.",
			ChangesMac:   "Yes",
			Phases:       a.factoryInstallPhases(factoryDryRunOpts),
			Confirmation: a.setupConfirmation(factoryDryRunOpts, factoryOpts),
		},
		{
			Name:        "Update This Mac",
			Description: "Update this host from tracked repo policy and the latest local app settings snapshot without importing current dotfiles back into the repo.",
			ChangesMac:  "Yes",
			Phases:      a.hostUpdatePhases(hostUpdateDryRunOpts),
			Confirmation: workflowConfirmation(
				"Update This Mac",
				"Apply tracked packages, apps, private secrets, dotfiles, app configs from the latest local snapshot, macOS settings, and health checks. Preview shows the full update plan without changing files or settings.",
				a.hostUpdatePhases(hostUpdateDryRunOpts),
				a.hostUpdatePhases(hostUpdateOpts),
			),
		},
		{
			Name:        "Save App Settings Snapshot",
			Description: "Collect supported app settings and setup reference files so they can be reviewed or restored later. This is selective, not a full Mac backup.",
			ChangesMac:  "Writes a snapshot",
			Phases:      capturePhases(a, captureDryRunOpts),
			Confirmation: workflowConfirmation(
				"Save App Settings Snapshot",
				"Collect supported app settings and setup reference files. Preview shows what would be collected; run now writes the snapshot archive.",
				capturePhases(a, captureDryRunOpts),
				capturePhases(a, captureOpts),
			),
		},
		{
			Name:        "Restore App Settings",
			Description: "Restore supported app settings from a prior snapshot.",
			ChangesMac:  "Yes",
			Phases:      restoreAppSettingsPhases(a, appDryRunOpts),
			Confirmation: workflowConfirmation(
				"Restore App Settings",
				"Restore supported app settings from a prior snapshot. Preview shows the restore plan without changing app files.",
				restoreAppSettingsPhases(a, appDryRunOpts),
				restoreAppSettingsPhases(a, appLiveOpts),
			),
		},
		{
			Name:        "Update Installed App List",
			Description: "Scan installed apps and write a reviewable apps.generated.yaml candidate without changing the tracked apps.yaml source of truth.",
			ChangesMac:  "Writes a generated file",
			Phases:      updateInstalledAppListPhases(a, updateAppsDryRunOpts),
			Confirmation: workflowConfirmation(
				"Update Installed App List",
				"Scan GUI apps, Homebrew casks, and Mac App Store apps. Preview prints the merge summary; run now writes apps.generated.yaml for review.",
				updateInstalledAppListPhases(a, updateAppsDryRunOpts),
				updateInstalledAppListPhases(a, updateAppsOpts),
			),
		},
		{
			Name:        "Apply macOS Settings",
			Description: "Apply the tracked macOS preferences for this setup.",
			ChangesMac:  "Yes",
			Phases:      macOSSettingsPhases(a, dryRunOpts),
			Confirmation: workflowConfirmation(
				"Apply macOS Settings",
				"Apply the tracked macOS preferences. Preview prints the defaults commands without applying them.",
				macOSSettingsPhases(a, dryRunOpts),
				macOSSettingsPhases(a, options{}),
			),
		},
		{
			Name:         "Check Setup",
			Description:  "Check whether prerequisites, tools, and expected setup state look correct.",
			ChangesMac:   "No",
			Phases:       doctorPhases(a, dryRunOpts),
			Confirmation: safeWorkflowConfirmation("Check Setup", "Run health checks only. This does not install packages or change settings.", doctorPhases(a, dryRunOpts)),
		},
		{
			Name:         "Show Homebrew Packages",
			Description:  "Print the generated Homebrew package list.",
			ChangesMac:   "No",
			Phases:       brewfilePreviewPhases(),
			Confirmation: safeWorkflowConfirmation("Show Homebrew Packages", "Print the generated Homebrew package list. This does not install anything.", brewfilePreviewPhases()),
		},
	})
}

func capturePhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{{Name: "Save supported app settings snapshot", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).captureArchive(opts) }}}
}

func restoreAppSettingsPhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{{Name: "Restore supported app settings", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(opts) }}}
}

func updateInstalledAppListPhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{{Name: "Generate installed app list candidate", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).updateInstalledAppList(opts) }}}
}

func macOSSettingsPhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{{Name: "Apply tracked macOS settings", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }}}
}

func doctorPhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{{Name: "Run health checks", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }}}
}

func brewfilePreviewPhases() []workflowdomain.Phase {
	return []workflowdomain.Phase{{Name: "Print generated Homebrew package list", Enabled: true, Run: func(w io.Writer) error { fmt.Fprint(w, brewfile.Content()); return nil }}}
}

func workflowConfirmation(title, message string, previewPhases, livePhases []workflowdomain.Phase) *workflowdomain.Confirmation {
	return &workflowdomain.Confirmation{
		Title:   title,
		Message: message,
		Options: []workflowdomain.ConfirmationOption{
			{Label: "Preview only", Description: "show what would happen", Continue: true, Phases: previewPhases},
			{Label: "Run now", Description: "make the described changes", Continue: true, Phases: livePhases},
			{Label: "Back", Description: "return to workflow menu", Back: true},
		},
	}
}

func safeWorkflowConfirmation(title, message string, phases []workflowdomain.Phase) *workflowdomain.Confirmation {
	return &workflowdomain.Confirmation{
		Title:   title,
		Message: message,
		Options: []workflowdomain.ConfirmationOption{
			{Label: "Run now", Description: "continue", Continue: true, Phases: phases},
			{Label: "Back", Description: "return to workflow menu", Back: true},
		},
	}
}

func (a app) setupConfirmation(previewOpts, liveOpts options) *workflowdomain.Confirmation {
	return &workflowdomain.Confirmation{
		Title:   "Set Up This Mac",
		Message: "Run the complete setup flow for a clean or intentionally reconfigured Mac. Preview shows the full plan without installing packages, opening reset settings, or changing files.",
		Options: []workflowdomain.ConfirmationOption{
			{
				Label:       "Preview only",
				Description: "show what would happen",
				Continue:    true,
				Phases:      a.factoryInstallPhases(previewOpts),
				Run: func(w io.Writer) error {
					fmt.Fprintln(w, "preview selected: no setup changes will be made")

					return nil
				},
			},
			{
				Label:       "Erase first",
				Description: "open reset settings and stop",
				Continue:    false,
				Phases:      a.factoryInstallPhases(liveOpts),
				Run: func(w io.Writer) error {
					return a.withStdout(w).openEraseAssistant(false)
				},
			},
			{
				Label:       "Already erased, run now",
				Description: "continue setup",
				Continue:    true,
				Phases:      a.factoryInstallPhases(liveOpts),
				Run: func(w io.Writer) error {
					fmt.Fprintln(w, "confirmed: Mac was already erased; continuing setup")

					return nil
				},
			},
			{
				Label:       "Run without erasing",
				Description: "continue setup and log skipped erase",
				Continue:    true,
				Phases:      a.factoryInstallPhases(liveOpts),
				Run: func(w io.Writer) error {
					fmt.Fprintln(w, "confirmed: erase was intentionally skipped; continuing setup")

					return nil
				},
			},
			{
				Label:       "Back",
				Description: "return to workflow menu",
				Back:        true,
			},
		},
	}
}

func (a app) withStdout(stdout io.Writer) app {
	a.stdout = stdout

	return a
}
