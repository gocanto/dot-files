package app

import "github.com/gocanto/mac-os/internal/currentstate/doctor"

func (a app) ensurePrerequisites(opts options) error {
	return doctor.Service{GOOS: a.goos, GOARCH: a.goarch, Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.EnsurePrerequisites(opts.dryRun)
}

func (a app) runDoctor(options) error {
	return doctor.Service{GOOS: a.goos, GOARCH: a.goarch, Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}.Run(a.settings.OPVault, a.settings.OPItem, a.settings.SecretsConfigPath)
}
