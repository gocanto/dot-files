package services

import (
	"fmt"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/converge/brew"
	templatemacos "github.com/gocanto/dot-files/internal/template/macos"
)

func (s Service) InspectCurrentBrew(_ Options) error {
	svc := brew.Service{Stdout: s.Stdout, Runner: s.Runner}
	formulae, err := svc.InstalledFormulae()

	if err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "# Installed Homebrew formulae (%d)\n", len(formulae))

	for _, name := range formulae {
		fmt.Fprintf(s.Stdout, "  - %s\n", name)
	}

	casks, err := svc.InstalledCasks()

	if err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "# Installed Homebrew casks (%d)\n", len(casks))

	for _, name := range casks {
		fmt.Fprintf(s.Stdout, "  - %s\n", name)
	}

	return nil
}

func (s Service) InspectCurrentMacOS(_ Options) error {
	fmt.Fprintln(s.Stdout, "# Tracked macOS defaults domains (current values)")

	for _, domain := range templatemacos.Domains() {
		out, err := s.Runner.Run("defaults", "read", domain)

		if err != nil {
			fmt.Fprintf(s.Stdout, "  %s: read failed: %v\n", domain, err)

			continue
		}

		fmt.Fprintf(s.Stdout, "## %s\n", domain)
		fmt.Fprintln(s.Stdout, command.FirstLine(out))
	}

	return nil
}
