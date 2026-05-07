package app

import (
	"fmt"

	"github.com/gocanto/mac-os/internal/template/secrets"
)

func (a app) restorePrivateSecrets(opts options) error {
	svc := secrets.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Runner: a.runner}

	secretOpts := secrets.Options{
		DryRun:      opts.dryRun,
		SecretsPath: opts.secretsPath,
		OPVault:     opts.opVault,
		OPItem:      opts.opItem,
	}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would validate 1Password CLI access: op vault list --format=json (and op signin if needed)\n")
		fmt.Fprintf(a.stdout, "would decrypt private secrets from 1Password item %q in vault %q\n", opts.opItem, opts.opVault)

		return svc.Decrypt(secretOpts)
	}

	if err := a.ensureOpSession(); err != nil {
		return err
	}

	return svc.Decrypt(secretOpts)
}
