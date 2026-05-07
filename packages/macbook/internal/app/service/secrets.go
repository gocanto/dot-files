package service

import (
	"fmt"

	"github.com/gocanto/dot-files/internal/template/secrets"
)

func (s Service) RestorePrivateSecrets(opts Options) error {
	svc := secrets.Service{Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Runner: s.Runner}

	secretOpts := secrets.Options{
		DryRun:      opts.DryRun,
		SecretsPath: opts.SecretsPath,
		OPVault:     opts.OPVault,
		OPItem:      opts.OPItem,
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would validate 1Password CLI access: op vault list --format=json (and op signin if needed)\n")
		fmt.Fprintf(s.Stdout, "would decrypt private secrets from 1Password item %q in vault %q\n", opts.OPItem, opts.OPVault)

		return svc.Decrypt(secretOpts)
	}

	if err := s.EnsureOpSession(); err != nil {
		return err
	}

	return svc.Decrypt(secretOpts)
}
