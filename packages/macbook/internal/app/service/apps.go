package service

import (
	"github.com/gocanto/dot-files/internal/converge/appstore"
	"github.com/gocanto/dot-files/internal/currentstate/inventory"
)

func (s Service) ApplyAppStoreApps(opts Options) error {
	return s.appstore().ApplyAppStore(appstore.Options{DryRun: opts.DryRun, Apps: opts.Apps, ConfigPath: opts.ConfigPath})
}

func (s Service) ReportManualApps(opts Options) error {
	return s.appstore().ReportManual(appstore.Options{DryRun: opts.DryRun, Apps: opts.Apps, ConfigPath: opts.ConfigPath})
}

func (s Service) UpdateInstalledAppList(opts Options) error {
	return s.currentApps().GenerateInstalledList(inventory.Options{DryRun: opts.DryRun, ConfigPath: opts.ConfigPath, GeneratedPath: opts.GeneratedPath})
}

func (s Service) appstore() appstore.Service {
	return appstore.Service{Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}
}

func (s Service) currentApps() inventory.Service {
	return inventory.Service{Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}
}
