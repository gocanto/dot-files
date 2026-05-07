package app

import (
	"fmt"

	"github.com/gocanto/mac-os/internal/converge/github"
)

func (a app) setupGitHub(opts options) error {
	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would validate 1Password CLI access: op vault list --format=json (and op signin if needed)\n")
	} else if err := a.ensureOpSession(); err != nil {
		return err
	}

	return github.Service{
		Home:   a.home,
		Repo:   a.repo,
		Stdin:  a.stdin,
		Stdout: a.stdout,
		Runner: a.runner,
	}.Setup(github.Options{
		DryRun:  opts.dryRun,
		OPVault: opts.opVault,
		OPItem:  opts.opItem,
	})
}
