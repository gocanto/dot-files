package service

import (
	"fmt"

	"github.com/gocanto/dot-files/internal/converge/appstore"
	"github.com/gocanto/dot-files/internal/converge/brew"
	"github.com/gocanto/dot-files/internal/template/brewfile"
)

type untrackedReport struct {
	Formulae []string
	Casks    []string
	AppStore []appstore.AppStoreApp
}

func (s Service) scanUntracked(opts Options) (untrackedReport, error) {
	report := untrackedReport{}
	svc := brew.Service{Stdout: s.Stdout, Runner: s.Runner}

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

	if opts.Apps {
		appStore, err := s.appstore().UntrackedAppStore(appstore.Options{ConfigPath: opts.ConfigPath, Apps: true})

		if err != nil {
			return report, err
		}

		report.AppStore = appStore
	}

	return report, nil
}

func (s Service) ReportUntracked(opts Options) error {
	report, err := s.scanUntracked(opts)

	if err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "Untracked Homebrew formulae (%d):\n", len(report.Formulae))

	for _, name := range report.Formulae {
		fmt.Fprintf(s.Stdout, "  - %s\n", name)
	}

	fmt.Fprintf(s.Stdout, "Untracked Homebrew casks (%d):\n", len(report.Casks))

	for _, name := range report.Casks {
		fmt.Fprintf(s.Stdout, "  - %s\n", name)
	}

	fmt.Fprintf(s.Stdout, "Untracked App Store apps (%d):\n", len(report.AppStore))

	for _, app := range report.AppStore {
		fmt.Fprintf(s.Stdout, "  - %s (%s)\n", app.Name, app.ID)
	}

	return nil
}

func (s Service) RemoveUntrackedBrew(opts Options) error {
	report, err := s.scanUntracked(opts)

	if err != nil {
		return err
	}

	svc := brew.Service{Stdout: s.Stdout, Runner: s.Runner}

	for _, name := range report.Formulae {
		if err := svc.Uninstall(brew.KindFormula, name, opts.DryRun); err != nil {
			return err
		}
	}

	for _, name := range report.Casks {
		if err := svc.Uninstall(brew.KindCask, name, opts.DryRun); err != nil {
			return err
		}
	}

	return nil
}

func (s Service) RemoveUntrackedAppStore(opts Options) error {
	if !opts.Apps {
		fmt.Fprintln(s.Stdout, "skipped: run with --apps to inspect Mac App Store removals")

		return nil
	}

	report, err := s.scanUntracked(opts)

	if err != nil {
		return err
	}

	if len(report.AppStore) == 0 {
		fmt.Fprintln(s.Stdout, "no untracked App Store apps detected")

		return nil
	}

	if !opts.AllowMasUninstall {
		fmt.Fprintln(s.Stdout, "mas uninstall is gated off. Open Finder or App Store and remove these manually:")

		for _, app := range report.AppStore {
			fmt.Fprintf(s.Stdout, "  - %s (%s)\n", app.Name, app.ID)
		}

		return nil
	}

	manualCleanup := []appstore.AppStoreApp{}

	for _, app := range report.AppStore {
		if err := s.appstore().UninstallAppStore(app, opts.DryRun); err != nil {
			fmt.Fprintf(s.Stdout, "warning: %v\n", err)
			manualCleanup = append(manualCleanup, app)
		}
	}

	if len(manualCleanup) > 0 {
		fmt.Fprintf(s.Stdout, "manual cleanup needed for %d App Store apps:\n", len(manualCleanup))

		for _, app := range manualCleanup {
			fmt.Fprintf(s.Stdout, "  - %s (%s)\n", app.Name, app.ID)
		}
	}

	return nil
}
