package app

import (
	"github.com/gocanto/mac-os/internal/converge/appstore"
	currentapps "github.com/gocanto/mac-os/internal/currentstate/apps"
)

func (a app) applyAppStoreApps(opts options) error {
	return a.appstore().ApplyAppStore(appstore.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
}

func (a app) reportManualApps(opts options) error {
	return a.appstore().ReportManual(appstore.Options{DryRun: opts.dryRun, Apps: opts.apps, ConfigPath: opts.configPath})
}

func (a app) updateInstalledAppList(opts options) error {
	return a.currentApps().GenerateInstalledList(currentapps.Options{DryRun: opts.dryRun, ConfigPath: opts.configPath, GeneratedPath: opts.generatedPath})
}

func (a app) appstore() appstore.Service {
	return appstore.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}
}

func (a app) currentApps() currentapps.Service {
	return currentapps.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}
}
