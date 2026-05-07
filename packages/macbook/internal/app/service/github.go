package service

import (
	"fmt"

	"github.com/gocanto/dot-files/internal/converge/github"
)

func (s Service) SetupGitHub(opts Options) error {
	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would validate 1Password CLI access: op vault list --format=json (and op signin if needed)\n")
	} else if err := s.EnsureOpSession(); err != nil {
		return err
	}

	return github.Service{
		Home:   s.Home,
		Repo:   s.Repo,
		Stdin:  s.Stdin,
		Stdout: s.Stdout,
		Runner: s.Runner,
	}.Setup(github.Options{
		DryRun:  opts.DryRun,
		OPVault: opts.OPVault,
		OPItem:  opts.OPItem,
	})
}
