package app

import (
	"io"

	"github.com/gocanto/dot-files/internal/domain"
)

type convergeMode string

const (
	convergeFresh      convergeMode = "fresh"
	convergeReconverge convergeMode = "reconverge"
)

func (a app) convergePhases(opts options, mode convergeMode) []domain.Phase {
	phases := []domain.Phase{
		{Name: "Check/install prerequisites", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().EnsurePrerequisites(opts) }},
		{Name: "Install Homebrew packages", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().ApplyHomebrewBundle(opts) }},
		{Name: "Set up GitHub access and signing", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().SetupGitHub(opts) }},
		{Name: "Install App Store apps", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().ApplyAppStoreApps(opts) }},
		{Name: "Show manual app install notes", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().ReportManualApps(opts) }},
		{Name: "Restore private secrets from 1Password", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().RestorePrivateSecrets(opts) }},
	}

	if mode == convergeFresh {
		phases = append(phases, domain.Phase{
			Name: "Prepare existing dotfiles", Enabled: true,
			Run: func(w io.Writer) error { return a.withStdout(w).service().AdoptDotfiles(opts) },
		})
	}

	phases = append(phases,
		domain.Phase{Name: "Install oh-my-zsh", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().EnsureOhMyZsh(opts) }},
		domain.Phase{Name: "Link dotfiles", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().ApplyStow(opts) }},
	)

	if mode == convergeReconverge {
		phases = append(phases, domain.Phase{
			Name: "Restore supported app configs from latest snapshot", Enabled: true,
			Run: func(w io.Writer) error { return a.withStdout(w).service().RestoreAppConfigs(opts) },
		})
	}

	return append(phases,
		domain.Phase{Name: "Apply macOS settings", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().ApplyMacOSDefaults(opts) }},
		domain.Phase{Name: "Run health checks", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().RunDoctor(opts) }},
	)
}

func (a app) factoryInstallPhases(opts options) []domain.Phase {
	return a.convergePhases(opts, convergeFresh)
}

func (a app) hostUpdatePhases(opts options) []domain.Phase {
	return a.convergePhases(opts, convergeReconverge)
}

func (a app) workflows() []domain.Workflow {
	baseOpts := options{
		ConfigPath:    a.settings.AppsConfigPath,
		GeneratedPath: a.settings.GeneratedAppsPath,
		SecretsPath:   a.settings.SecretsConfigPath,
		ArchiveRoot:   a.settings.ArchiveRoot,
		OPVault:       a.settings.OPVault,
		OPItem:        a.settings.OPItem,
	}
	dryRunOpts := baseOpts
	dryRunOpts.DryRun = true
	factoryOpts := baseOpts
	factoryOpts.Apps = true
	factoryDryRunOpts := factoryOpts
	factoryDryRunOpts.DryRun = true
	hostUpdateOpts := factoryOpts
	hostUpdateOpts.UseLatestArchive = true
	hostUpdateDryRunOpts := hostUpdateOpts
	hostUpdateDryRunOpts.DryRun = true
	appDryRunOpts := baseOpts
	appDryRunOpts.DryRun = true
	appDryRunOpts.Apps = true
	appLiveOpts := baseOpts
	appLiveOpts.Apps = true
	updateAppsDryRunOpts := baseOpts
	updateAppsDryRunOpts.DryRun = true
	updateAppsOpts := baseOpts
	captureDryRunOpts := factoryDryRunOpts
	captureOpts := factoryOpts

	return domain.Normalize([]domain.Workflow{
		{
			Name:         "Review Template",
			Description:  "Validate the tracked template, then print the source of truth: Homebrew bundle, apps.yaml, macOS defaults, and dotfile bundles. Read-only.",
			ChangesMac:   "No",
			Phases:       reviewTemplatePhases(a, baseOpts),
			Confirmation: safeWorkflowConfirmation("Review Template", "Validate and print the tracked source of truth. This does not change anything on this Mac.", reviewTemplatePhases(a, baseOpts)),
		},
		{
			Name:        "Update Template From This Mac",
			Description: "Save a current-Mac snapshot, then write review-candidate template files without overwriting the tracked template.",
			ChangesMac:  "Writes review candidates",
			Phases:      updateTemplateFromMacPhases(a, updateTemplateFromMacDryOpts(baseOpts)),
			Confirmation: workflowConfirmation(
				"Update Template From This Mac",
				"Capture this Mac and generate review candidates. Preview shows the candidate paths; run now saves a snapshot first and never overwrites tracked template files.",
				updateTemplateFromMacPhases(a, updateTemplateFromMacDryOpts(baseOpts)),
				updateTemplateFromMacPhases(a, updateTemplateFromMacLiveOpts(baseOpts)),
			),
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
			Confirmation: approvingWorkflowConfirmation(
				"Restore Snapshot",
				"Restore supported app settings from a prior snapshot. Preview shows the restore plan without changing app files.",
				restoreAppSettingsPhases(a, appDryRunOpts),
				restoreAppSettingsPhases(a, appLiveOpts),
				a.approvalOption,
			),
		},
		{
			Name:        "Remove Untracked Apps",
			Description: "Uninstall Homebrew formulae and casks that are not in the tracked Brewfile, plus a best-effort Mac App Store cleanup pass.",
			ChangesMac:  "Yes (destructive)",
			Phases:      removeUntrackedPhases(a, removeUntrackedDryOpts(baseOpts)),
			Confirmation: approvingWorkflowConfirmation(
				"Remove Untracked Apps",
				"Scan installed Homebrew formulae, casks, and App Store apps and remove anything not in the tracked template. Preview lists candidates without uninstalling. Run now writes a snapshot first, then uninstalls.",
				removeUntrackedPhases(a, removeUntrackedDryOpts(baseOpts)),
				removeUntrackedPhases(a, removeUntrackedLiveOpts(baseOpts)),
				a.approvalOption,
			),
		},
	})
}

func previewTemplatePhases(a app, opts options) []domain.Phase {
	return []domain.Phase{
		{Name: "Print tracked Homebrew bundle", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().PreviewTemplateBrew(opts)
		}},
		{Name: "List tracked apps", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().PreviewTemplateApps(opts)
		}},
		{Name: "List tracked macOS settings", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().PreviewTemplateMacOS(opts)
		}},
		{Name: "List tracked dotfile bundles", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().PreviewTemplateDotfiles(opts)
		}},
	}
}

func reviewTemplatePhases(a app, opts options) []domain.Phase {
	return append(validateTemplatePhases(a, opts), previewTemplatePhases(a, opts)...)
}

func validateTemplatePhases(a app, opts options) []domain.Phase {
	return []domain.Phase{
		{Name: "Validate template files", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().ValidateTemplate(opts)
		}},
	}
}

func updateTemplateFromMacDryOpts(base options) options {
	opts := base
	opts.DryRun = true
	opts.Apps = true

	return opts
}

func updateTemplateFromMacLiveOpts(base options) options {
	opts := base
	opts.Apps = true

	return opts
}

func updateTemplateFromMacPhases(a app, opts options) []domain.Phase {
	return []domain.Phase{
		{Name: "Save current Mac snapshot", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().CaptureArchive(opts)
		}},
		{Name: "Generate installed app review candidate", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().UpdateInstalledAppList(opts)
		}},
		{Name: "Generate dotfile review candidates", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().WriteDotfileCandidates(opts)
		}},
	}
}

func inspectCurrentPhases(a app, opts options) []domain.Phase {
	return []domain.Phase{
		{Name: "Run health checks", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().RunDoctor(opts)
		}},
		{Name: "List installed Homebrew formulae and casks", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().InspectCurrentBrew(opts)
		}},
		{Name: "Show current macOS defaults values", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().InspectCurrentMacOS(opts)
		}},
	}
}

func (a app) convergeConfirmation(freshDry, freshLive, reconvergeDry, reconvergeLive options) *domain.Confirmation {
	return &domain.Confirmation{
		Title:   "Converge to Template",
		Message: "Apply the tracked template to this Mac. Preview shows the plan without changing anything. Fresh setup adopts existing dotfiles into the repo (for clean/erased Macs). Re-converge skips adoption and restores app configs from the latest snapshot.",
		Options: []domain.ConfirmationOption{
			{Label: "Preview only (fresh)", Description: "show what a fresh setup would do", Continue: true, Phases: a.convergePhases(freshDry, convergeFresh)},
			{Label: "Preview only (re-converge)", Description: "show what a re-converge would do", Continue: true, Phases: a.convergePhases(reconvergeDry, convergeReconverge)},
			a.approvalOption(domain.ConfirmationOption{Label: "Erase first", Description: "open reset settings and stop", Continue: false, Phases: a.convergePhases(freshLive, convergeFresh), Run: func(w io.Writer) error {
				return a.withStdout(w).service().OpenEraseAssistant(false)
			}}),
			a.approvalOption(domain.ConfirmationOption{Label: "Fresh setup", Description: "install everything and adopt existing dotfiles", Continue: true, Phases: a.convergePhases(freshLive, convergeFresh)}),
			a.approvalOption(domain.ConfirmationOption{Label: "Re-converge", Description: "update from policy and restore from latest snapshot", Continue: true, Phases: a.convergePhases(reconvergeLive, convergeReconverge)}),
			{Label: "Back", Description: "return to workflow menu", Back: true},
		},
	}
}

func removeUntrackedDryOpts(base options) options {
	opts := base
	opts.DryRun = true
	opts.Apps = true

	return opts
}

func removeUntrackedLiveOpts(base options) options {
	opts := base
	opts.Apps = true

	return opts
}

func removeUntrackedPhases(a app, opts options) []domain.Phase {
	captureOpts := opts
	captureOpts.Apps = true

	return []domain.Phase{
		{Name: "Scan untracked items", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().ReportUntracked(opts)
		}},
		{Name: "Snapshot before remove", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().CaptureArchive(captureOpts)
		}},
		{Name: "Uninstall untracked Homebrew formulae and casks", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().RemoveUntrackedBrew(opts)
		}},
		{Name: "Uninstall untracked App Store apps (best effort)", Enabled: true, Run: func(w io.Writer) error {
			return a.withStdout(w).service().RemoveUntrackedAppStore(opts)
		}},
	}
}

func capturePhases(a app, opts options) []domain.Phase {
	return []domain.Phase{{Name: "Save supported app settings snapshot", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().CaptureArchive(opts) }}}
}

func restoreAppSettingsPhases(a app, opts options) []domain.Phase {
	return []domain.Phase{{Name: "Restore supported app settings", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().RestoreAppConfigs(opts) }}}
}

func updateInstalledAppListPhases(a app, opts options) []domain.Phase {
	return []domain.Phase{{Name: "Generate installed app list candidate", Enabled: true, Run: func(w io.Writer) error { return a.withStdout(w).service().UpdateInstalledAppList(opts) }}}
}

func workflowConfirmation(title, message string, previewPhases, livePhases []domain.Phase) *domain.Confirmation {
	return &domain.Confirmation{
		Title:   title,
		Message: message,
		Options: []domain.ConfirmationOption{
			{Label: "Preview only", Description: "show what would happen", Continue: true, Phases: previewPhases},
			{Label: "Run now", Description: "make the described changes", Continue: true, Phases: livePhases},
			{Label: "Back", Description: "return to workflow menu", Back: true},
		},
	}
}

func approvingWorkflowConfirmation(title, message string, previewPhases, livePhases []domain.Phase, approve func(domain.ConfirmationOption) domain.ConfirmationOption) *domain.Confirmation {
	confirmation := workflowConfirmation(title, message, previewPhases, livePhases)
	confirmation.Options[1] = approve(confirmation.Options[1])

	return confirmation
}

func safeWorkflowConfirmation(title, message string, phases []domain.Phase) *domain.Confirmation {
	return &domain.Confirmation{
		Title:   title,
		Message: message,
		Options: []domain.ConfirmationOption{
			{Label: "Run now", Description: "continue", Continue: true, Phases: phases},
			{Label: "Back", Description: "return to workflow menu", Back: true},
		},
	}
}

func (a app) withStdout(stdout io.Writer) app {
	a.stdout = stdout

	return a
}
