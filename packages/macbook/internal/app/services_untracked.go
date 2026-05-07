package app

import (
	"fmt"

	"github.com/gocanto/mac-os/internal/converge/appstore"
	"github.com/gocanto/mac-os/internal/converge/brew"
	"github.com/gocanto/mac-os/internal/template/brewfile"
)

type untrackedReport struct {
	Formulae []string
	Casks    []string
	AppStore []appstore.AppStoreApp
}

func (a app) scanUntracked(opts options) (untrackedReport, error) {
	report := untrackedReport{}
	svc := brew.Service{Stdout: a.stdout, Runner: a.runner}

	installedFormulae, err := svc.InstalledFormulae()

	if err != nil {
		return report, err
	}

	report.Formulae = brew.Untracked(installedFormulae, brewfile.TrackedFormulae())

	installedCasks, err := svc.InstalledCasks()

	if err != nil {
		return report, err
	}

	report.Casks = brew.Untracked(installedCasks, brewfile.TrackedCasks())

	if opts.apps {
		appStore, err := a.appstore().UntrackedAppStore(appstore.Options{ConfigPath: opts.configPath, Apps: true})

		if err != nil {
			return report, err
		}

		report.AppStore = appStore
	}

	return report, nil
}

func (a app) reportUntracked(opts options) error {
	report, err := a.scanUntracked(opts)

	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "Untracked Homebrew formulae (%d):\n", len(report.Formulae))

	for _, name := range report.Formulae {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	fmt.Fprintf(a.stdout, "Untracked Homebrew casks (%d):\n", len(report.Casks))

	for _, name := range report.Casks {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	fmt.Fprintf(a.stdout, "Untracked App Store apps (%d):\n", len(report.AppStore))

	for _, app := range report.AppStore {
		fmt.Fprintf(a.stdout, "  - %s (%s)\n", app.Name, app.ID)
	}

	return nil
}

func (a app) removeUntrackedBrew(opts options) error {
	report, err := a.scanUntracked(opts)

	if err != nil {
		return err
	}

	svc := brew.Service{Stdout: a.stdout, Runner: a.runner}

	for _, name := range report.Formulae {
		if err := svc.Uninstall(brew.KindFormula, name, opts.dryRun); err != nil {
			return err
		}
	}

	for _, name := range report.Casks {
		if err := svc.Uninstall(brew.KindCask, name, opts.dryRun); err != nil {
			return err
		}
	}

	return nil
}

func (a app) removeUntrackedAppStore(opts options) error {
	if !opts.apps {
		fmt.Fprintln(a.stdout, "skipped: run with --apps to inspect Mac App Store removals")

		return nil
	}

	report, err := a.scanUntracked(opts)

	if err != nil {
		return err
	}

	if len(report.AppStore) == 0 {
		fmt.Fprintln(a.stdout, "no untracked App Store apps detected")

		return nil
	}

	if !opts.allowMasUninstall {
		fmt.Fprintln(a.stdout, "mas uninstall is gated off. Open Finder or App Store and remove these manually:")

		for _, app := range report.AppStore {
			fmt.Fprintf(a.stdout, "  - %s (%s)\n", app.Name, app.ID)
		}

		return nil
	}

	manualCleanup := []appstore.AppStoreApp{}

	for _, app := range report.AppStore {
		if err := a.appstore().UninstallAppStore(app, opts.dryRun); err != nil {
			fmt.Fprintf(a.stdout, "warning: %v\n", err)
			manualCleanup = append(manualCleanup, app)
		}
	}

	if len(manualCleanup) > 0 {
		fmt.Fprintf(a.stdout, "manual cleanup needed for %d App Store apps:\n", len(manualCleanup))

		for _, app := range manualCleanup {
			fmt.Fprintf(a.stdout, "  - %s (%s)\n", app.Name, app.ID)
		}
	}

	return nil
}
