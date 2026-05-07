package app

import (
	"fmt"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/converge/brew"
	templatemacos "github.com/gocanto/mac-os/internal/template/macos"
)

func (a app) inspectCurrentBrew(_ options) error {
	svc := brew.Service{Stdout: a.stdout, Runner: a.runner}
	formulae, err := svc.InstalledFormulae()

	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "# Installed Homebrew formulae (%d)\n", len(formulae))

	for _, name := range formulae {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	casks, err := svc.InstalledCasks()

	if err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "# Installed Homebrew casks (%d)\n", len(casks))

	for _, name := range casks {
		fmt.Fprintf(a.stdout, "  - %s\n", name)
	}

	return nil
}

func (a app) inspectCurrentMacOS(_ options) error {
	fmt.Fprintln(a.stdout, "# Tracked macOS defaults domains (current values)")

	for _, domain := range templatemacos.Domains() {
		out, err := a.runner.Run("defaults", "read", domain)

		if err != nil {
			fmt.Fprintf(a.stdout, "  %s: read failed: %v\n", domain, err)

			continue
		}

		fmt.Fprintf(a.stdout, "## %s\n", domain)
		fmt.Fprintln(a.stdout, command.FirstLine(out))
	}

	return nil
}
