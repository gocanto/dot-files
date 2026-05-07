package macos

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/safefs"
	templatemacos "github.com/gocanto/mac-os/internal/template/macos"
)

type Service struct {
	Runner command.Runner
	Stdout io.Writer
	Stderr io.Writer
}

func (s Service) Export(root string) error {
	if s.Runner == nil {
		return fmt.Errorf("defaults export: runner is nil")
	}

	stderr := s.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	domains := templatemacos.Domains()
	exported := 0

	for _, domain := range domains {
		out, err := s.Runner.Run("defaults", "export", domain, "-")

		if err != nil {
			fmt.Fprintf(stderr, "warning: defaults export %s failed: %v\n", domain, err)

			continue
		}

		name := strings.ReplaceAll(domain, "/", "_") + ".plist"

		if err := safefs.WriteFile(filepath.Join(root, "defaults", name), out, 0o600); err != nil {
			return err
		}

		exported++
	}

	if exported == 0 && len(domains) > 0 {
		return fmt.Errorf("defaults export: 0 of %d domains exported", len(domains))
	}

	return nil
}
