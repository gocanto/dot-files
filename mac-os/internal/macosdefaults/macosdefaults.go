package macosdefaults

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/safefs"
)

type Setting struct {
	Domain string
	Key    string
	Args   []string
}

type Service struct {
	Runner command.Runner
	Stdout io.Writer
	Stderr io.Writer
}

func Settings() []Setting {
	return []Setting{
		{"NSGlobalDomain", "AppleInterfaceStyle", []string{"-string", "Dark"}},
		{"NSGlobalDomain", "AppleShowAllExtensions", []string{"-bool", "true"}},
		{"NSGlobalDomain", "ApplePressAndHoldEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticDashSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticQuoteSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSAutomaticPeriodSubstitutionEnabled", []string{"-bool", "false"}},
		{"NSGlobalDomain", "NSNavPanelExpandedStateForSaveMode", []string{"-bool", "true"}},
		{"NSGlobalDomain", "PMPrintingExpandedStateForPrint", []string{"-bool", "true"}},
		{"com.apple.finder", "AppleShowAllFiles", []string{"-bool", "true"}},
		{"com.apple.finder", "ShowPathbar", []string{"-bool", "true"}},
		{"com.apple.finder", "ShowStatusBar", []string{"-bool", "true"}},
		{"com.apple.finder", "FXPreferredViewStyle", []string{"-string", "Nlsv"}},
		{"com.apple.finder", "_FXShowPosixPathInTitle", []string{"-bool", "true"}},
		{"com.apple.dock", "autohide", []string{"-bool", "true"}},
		{"com.apple.dock", "mineffect", []string{"-string", "scale"}},
		{"com.apple.dock", "minimize-to-application", []string{"-bool", "true"}},
		{"com.apple.screencapture", "type", []string{"-string", "png"}},
		{"com.apple.screencapture", "disable-shadow", []string{"-bool", "true"}},
	}
}

func Domains() []string {
	return []string{
		"NSGlobalDomain",
		"com.apple.dock",
		"com.apple.finder",
		"com.apple.screencapture",
		"com.apple.AppleMultitouchTrackpad",
		"com.apple.driver.AppleBluetoothMultitouch.trackpad",
		"com.mitchellh.ghostty",
		"com.googlecode.iterm2",
		"com.jordanbaird.Ice",
	}
}

func (s Service) Apply(dryRun bool) error {
	for _, setting := range Settings() {
		cmd := append([]string{"defaults", "write", setting.Domain, setting.Key}, setting.Args...)

		if dryRun {
			fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

			continue
		}

		out, err := s.Runner.Run(cmd[0], cmd[1:]...)

		if len(out) > 0 {
			fmt.Fprint(s.Stdout, string(out))
		}

		if err != nil {
			return fmt.Errorf("%s: %w", command.ShellQuote(cmd), err)
		}
	}

	for _, cmd := range [][]string{
		{"killall", "Finder"},
		{"killall", "Dock"},
		{"killall", "SystemUIServer"},
	} {
		if dryRun {
			fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

			continue
		}

		_, _ = s.Runner.Run(cmd[0], cmd[1:]...)
	}

	return nil
}

func (s Service) Export(root string) error {
	for _, domain := range Domains() {
		out, err := s.Runner.Run("defaults", "export", domain, "-")

		if err != nil {
			fmt.Fprintf(s.Stderr, "warning: defaults export %s failed: %v\n", domain, err)

			continue
		}

		name := strings.ReplaceAll(domain, "/", "_") + ".plist"

		if err := safefs.WriteFile(filepath.Join(root, "defaults", name), out, 0o600); err != nil {
			return err
		}
	}

	return nil
}
