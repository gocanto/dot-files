package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
)

func (s Service) ensureAppleSiliconSupport(dryRun bool) error {
	if err := s.ensureRosetta(dryRun); err != nil {
		return err
	}

	return s.ensureDockerDesktopRosettaDisabled(dryRun)
}

func (s Service) ensureRosetta(dryRun bool) error {
	goarch := s.GOARCH

	if goarch == "" {
		goarch = runtime.GOARCH
	}

	if goarch != "arm64" {
		return nil
	}

	check := s.rosettaCheck

	if check == nil {
		check = func() bool {
			_, err := os.Stat(rosettaMarker)

			return err == nil
		}
	}

	if check() {
		fmt.Fprintln(s.Stdout, "Rosetta 2 found")

		return nil
	}

	cmd := []string{"softwareupdate", "--install-rosetta", "--agree-to-license"}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("install Rosetta 2 (Docker Desktop and other x86_64 binaries need it on Apple Silicon): %w", err)
	}

	return nil
}

func (s Service) ensureDockerDesktopRosettaDisabled(dryRun bool) error {
	goarch := s.GOARCH

	if goarch == "" {
		goarch = runtime.GOARCH
	}

	if goarch != "arm64" {
		return nil
	}

	home := s.Home

	if home == "" {
		var err error

		home, err = os.UserHomeDir()

		if err != nil {
			return fmt.Errorf("find home directory for Docker Desktop settings: %w", err)
		}
	}

	settingsPath := filepath.Join(home, dockerDesktopSettingsPath)
	settings := map[string]any{}

	if data, err := os.ReadFile(settingsPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse Docker Desktop settings at %s: %w", settingsPath, err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read Docker Desktop settings at %s: %w", settingsPath, err)
	}

	if settings["UseVirtualizationFrameworkRosetta"] == false && settings["useVirtualizationFrameworkRosetta"] == false {
		fmt.Fprintln(s.Stdout, "Docker Desktop Rosetta integration disabled")

		return nil
	}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would disable Docker Desktop Rosetta integration in %s\n", settingsPath)

		return nil
	}

	settings["UseVirtualizationFrameworkRosetta"] = false
	settings["useVirtualizationFrameworkRosetta"] = false

	data, err := json.MarshalIndent(settings, "", "  ")

	if err != nil {
		return fmt.Errorf("encode Docker Desktop settings: %w", err)
	}

	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("create Docker Desktop settings directory: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		return fmt.Errorf("write Docker Desktop settings at %s: %w", settingsPath, err)
	}

	fmt.Fprintln(s.Stdout, "Docker Desktop Rosetta integration disabled")

	return nil
}
