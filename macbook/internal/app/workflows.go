package app

import (
	"io"

	"github.com/gocanto/mac-os/internal/workflowdomain"
)

type convergeMode string

const (
	convergeFresh      convergeMode = "fresh"
	convergeReconverge convergeMode = "reconverge"
)

func (a app) convergePhases(opts options, mode convergeMode) []workflowdomain.Phase {
	phases := []workflowdomain.Phase{
		{Name: "Check/install prerequisites", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensurePrerequisites(opts) }},
		{Name: "Install Homebrew packages", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyHomebrewBundle(opts) }},
		{Name: "Set up GitHub access and signing", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).setupGitHub(opts) }},
		{Name: "Install App Store apps", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyAppStoreApps(opts) }},
		{Name: "Show manual app install notes", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).reportManualApps(opts) }},
		{Name: "Restore private secrets from 1Password", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).restorePrivateSecrets(opts) }},
	}

	if mode == convergeFresh {
		phases = append(phases, workflowdomain.Phase{
			Name: "Prepare existing dotfiles", Enabled: true,
			Run: func(w io.Writer) error { return a.withStdout(w).adoptDotfiles(opts) },
		})
	}

	phases = append(phases,
		workflowdomain.Phase{Name: "Install oh-my-zsh", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).ensureOhMyZsh(opts) }},
		workflowdomain.Phase{Name: "Link dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyStow(opts) }},
	)

	if mode == convergeReconverge {
		phases = append(phases, workflowdomain.Phase{
			Name: "Restore supported app configs from latest snapshot", Enabled: true,
			Run: func(w io.Writer) error { return a.withStdout(w).restoreAppConfigs(opts) },
		})
	}

	return append(phases,
		workflowdomain.Phase{Name: "Apply macOS settings", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).applyMacOSDefaults(opts) }},
		workflowdomain.Phase{Name: "Run health checks", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).runDoctor(opts) }},
	)
}

func (a app) factoryInstallPhases(opts options) []workflowdomain.Phase {
	return a.convergePhases(opts, convergeFresh)
}

func (a app) hostUpdatePhases(opts options) []workflowdomain.Phase {
	return a.convergePhases(opts, convergeReconverge)
}

func (a app) workflows() []workflowdomain.Workflow {
	baseOpts := options{
		configPath:    a.settings.AppsConfigPath,
		generatedPath: a.settings.GeneratedAppsPath,
		secretsPath:   a.settings.SecretsConfigPath,
		archiveRoot:   a.settings.ArchiveRoot,
		opVault:       a.settings.OPVault,
		opItem:        a.settings.OPItem,
	}
	dryRunOpts := baseOpts
	dryRunOpts.dryRun = true
	factoryOpts := baseOpts
	factoryOpts.apps = true
	factoryDryRunOpts := factoryOpts
	factoryDryRunOpts.dryRun = true
	hostUpdateOpts := factoryOpts
	hostUpdateOpts.useLatestArchive = true
	hostUpdateDryRunOpts := hostUpdateOpts
	hostUpdateDryRunOpts.dryRun = true
	appDryRunOpts := baseOpts
	appDryRunOpts.dryRun = true
	appDryRunOpts.apps = true
	appLiveOpts := baseOpts
	appLiveOpts.apps = true
	updateAppsDryRunOpts := baseOpts
	updateAppsDryRunOpts.dryRun = true
	updateAppsOpts := baseOpts
	captureDryRunOpts := factoryDryRunOpts
	captureOpts := factoryOpts

	return workflowdomain.Normalize([]workflowdomain.Workflow{
		{
			Name:         "Preview Template",
			Description:  "Print the tracked source of truth: Homebrew bundle, apps.yaml, macOS defaults, and dotfile bundles. Read-only.",
			ChangesMac:   "No",
			Phases:       previewTemplatePhases(a, baseOpts),
			Confirmation: safeWorkflowConfirmation("Preview Template", "Print the tracked source of truth. This does not change anything on this Mac.", previewTemplatePhases(a, baseOpts)),
		},
		{
			Name:         "Validate Template",
			Description:  "Validate that apps.yaml, secrets.yaml, the stow directory, the tracked Brewfile, and the tracked macOS settings are well-formed.",
			ChangesMac:   "No",
			Phases:       validateTemplatePhases(a, baseOpts),
			Confirmation: safeWorkflowConfirmation("Validate Template", "Validate the source of truth. This does not change anything on this Mac.", validateTemplatePhases(a, baseOpts)),
		},
		{
			Name:         "Inspect Current State",
			Description:  "Read-only inspection of this Mac: doctor checks, installed Homebrew formulae and casks, current macOS defaults values.",
			ChangesMac:   "No",
			Phases:       inspectCurrentPhases(a, baseOpts),
			Confirmation: safeWorkflowConfirmation("Inspect Current State", "Inspect this Mac without changing anything.", inspectCurrentPhases(a, baseOpts)),
		},
		{
			Name:        "Regenerate Installed App List",
			Description: "Scan installed apps and write a reviewable apps.generated.yaml candidate without changing the tracked apps.yaml source of truth.",
			ChangesMac:  "Writes a generated file",
			Phases:      updateInstalledAppListPhases(a, updateAppsDryRunOpts),
			Confirmation: workflowConfirmation(
				"Regenerate Installed App List",
				"Scan GUI apps, Homebrew casks, and Mac App Store apps. Preview prints the merge summary; run now writes apps.generated.yaml for review.",
				updateInstalledAppListPhases(a, updateAppsDryRunOpts),
				updateInstalledAppListPhases(a, updateAppsOpts),
			),
		},
		{
			Name:        "Save Snapshot",
			Description: "Collect supported app settings and setup reference files so they can be reviewed or restored later. This is selective, not a full Mac backup.",
			ChangesMac:  "Writes a snapshot",
			Phases:      capturePhases(a, captureDryRunOpts),
			Confirmation: workflowConfirmation(
				"Save Snapshot",
				"Collect supported app settings and setup reference files. Preview shows what would be collected; run now writes the snapshot archive.",
				capturePhases(a, captureDryRunOpts),
				capturePhases(a, captureOpts),
			),
		},
		{
			Name:         "Converge to Template",
			Description:  "Apply the tracked template (Homebrew, apps, secrets, dotfiles, macOS settings) to this Mac. Choose Fresh setup for a clean Mac (adopts existing dotfiles) or Re-converge to update without importing dotfiles.",
			ChangesMac:   "Yes",
			Phases:       a.convergePhases(factoryDryRunOpts, convergeFresh),
			Confirmation: a.convergeConfirmation(factoryDryRunOpts, factoryOpts, hostUpdateDryRunOpts, hostUpdateOpts),
		},
		{
			Name:        "Restore Snapshot",
			Description: "Restore supported app settings from a prior snapshot.",
			ChangesMac:  "Yes",
			Phases:      restoreAppSettingsPhases(a, appDryRunOpts),
			Confirmation: workflowConfirmation(
				"Restore Snapshot",
				"Restore supported app settings from a prior snapshot. Preview shows the restore plan without changing app files.",
				restoreAppSettingsPhases(a, appDryRunOpts),
				restoreAppSettingsPhases(a, appLiveOpts),
			),
		},
		{
			Name:        "Remove Untracked Apps",
			Description: "Uninstall Homebrew formulae and casks that are not in the tracked Brewfile, plus a best-effort Mac App Store cleanup pass.",
			ChangesMac:  "Yes (destructive)",
			Phases:      removeUntrackedPhases(a, removeUntrackedDryOpts(baseOpts)),
			Confirmation: workflowConfirmation(
				"Remove Untracked Apps",
				"Scan installed Homebrew formulae, casks, and App Store apps and remove anything not in the tracked template. Preview lists candidates without uninstalling. Run now writes a snapshot first, then uninstalls.",
				removeUntrackedPhases(a, removeUntrackedDryOpts(baseOpts)),
				removeUntrackedPhases(a, removeUntrackedLiveOpts(baseOpts)),
			),
		},
	})
}

func previewTemplatePhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{
		{Name: "Print tracked Homebrew bundle", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).previewTemplateBrew(opts)
		}},
		{Name: "List tracked apps", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).previewTemplateApps(opts)
		}},
		{Name: "List tracked macOS settings", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).previewTemplateMacOS(opts)
		}},
		{Name: "List tracked dotfile bundles", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).previewTemplateDotfiles(opts)
		}},
	}
}

func validateTemplatePhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{
		{Name: "Validate template files", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).validateTemplate(opts)
		}},
	}
}

func inspectCurrentPhases(a app, opts options) []workflowdomain.Phase {
	return []workflowdomain.Phase{
		{Name: "Run health checks", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).runDoctor(opts)
		}},
		{Name: "List installed Homebrew formulae and casks", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).inspectCurrentBrew(opts)
		}},
		{Name: "Show current macOS defaults values", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).inspectCurrentMacOS(opts)
		}},
	}
}

func (a app) convergeConfirmation(freshDry, freshLive, reconvergeDry, reconvergeLive options) *workflowdomain.Confirmation {
	return &workflowdomain.Confirmation{
		Title:   "Converge to Template",
		Message: "Apply the tracked template to this Mac. Preview shows the plan without changing anything. Fresh setup adopts existing dotfiles into the repo (for clean/erased Macs). Re-converge skips adoption and restores app configs from the latest snapshot.",
		Options: []workflowdomain.ConfirmationOption{
			{Label: "Preview only (fresh)", Description: "show what a fresh setup would do", Continue: true, Phases: a.convergePhases(freshDry, convergeFresh)},
			{Label: "Preview only (re-converge)", Description: "show what a re-converge would do", Continue: true, Phases: a.convergePhases(reconvergeDry, convergeReconverge)},
			{Label: "Erase first", Description: "open reset settings and stop", Continue: false, Phases: a.convergePhases(freshLive, convergeFresh), Run: func(w io.Writer) error {
				return a.withStdout(w).openEraseAssistant(false)
			}},
			{Label: "Fresh setup", Description: "install everything and adopt existing dotfiles", Continue: true, Phases: a.convergePhases(freshLive, convergeFresh)},
			{Label: "Re-converge", Description: "update from policy and restore from latest snapshot", Continue: true, Phases: a.convergePhases(reconvergeLive, convergeReconverge)},
			{Label: "Back", Description: "return to workflow menu", Back: true},
		},
	}
}

func removeUntrackedDryOpts(base options) options {
	opts := base
	opts.dryRun = true
	opts.apps = true

	return opts
}

func removeUntrackedLiveOpts(base options) options {
	opts := base
	opts.apps = true

	return opts
}

func removeUntrackedPhases(a app, opts options) []workflowdomain.Phase {
	captureOpts := opts
	captureOpts.apps = true

	return []workflowdomain.Phase{
		{Name: "Scan untracked items", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).reportUntracked(opts)
		}},
		{Name: "Snapshot before remove", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).captureArchive(captureOpts)
		}},
		{Name: "Uninstall untracked Homebrew formulae and casks", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).removeUntrackedBrew(opts)
		}},
		{Name: "Uninstall untracked App Store apps (best effort)", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).removeUntrackedAppStore(opts)
		}},
	}
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

func (a app) withStdout(stdout io.Writer) app {
	a.stdout = stdout

	return a
}
