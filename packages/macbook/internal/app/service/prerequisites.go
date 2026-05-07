package service

import "github.com/gocanto/dot-files/internal/currentstate/doctor"

func (s Service) EnsurePrerequisites(opts Options) error {
	return doctor.Service{GOOS: s.GOOS, GOARCH: s.GOARCH, Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}.EnsurePrerequisites(opts.DryRun)
}

func (s Service) RunDoctor(opts Options) error {
	return doctor.Service{GOOS: s.GOOS, GOARCH: s.GOARCH, Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}.Run(opts.OPVault, opts.OPItem, opts.SecretsPath)
}
