package appstore

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
	currentapps "github.com/gocanto/mac-os/internal/currentstate/apps"
)

type AppStoreApp struct {
	ID   string
	Name string
}

var ErrManualAppStoreCleanup = errors.New("mas uninstall failed; manual cleanup required")

func (s Service) UntrackedAppStore(opts Options) ([]AppStoreApp, error) {
	cfg, err := s.loader().Load(opts.ConfigPath)

	if err != nil {
		return nil, err
	}

	tracked := make(map[string]struct{})

	for _, app := range cfg.Apps {
		if app.InstallMethod != "mas" {
			continue
		}

		tracked[strings.TrimSpace(app.Package)] = struct{}{}
	}

	out, err := s.Runner.Run("mas", "list")

	if err != nil {
		return nil, fmt.Errorf("mas list: %w", err)
	}

	installed := currentapps.ParseMASList(out)
	untracked := make([]AppStoreApp, 0)

	for _, app := range installed {
		if _, ok := tracked[app.ID]; ok {
			continue
		}

		untracked = append(untracked, AppStoreApp{ID: app.ID, Name: app.Name})
	}

	sort.Slice(untracked, func(i, j int) bool {
		return strings.ToLower(untracked[i].Name) < strings.ToLower(untracked[j].Name)
	})

	return untracked, nil
}

func (s Service) UninstallAppStore(app AppStoreApp, dryRun bool) error {
	if strings.TrimSpace(app.ID) == "" {
		return fmt.Errorf("uninstall app store: empty id")
	}

	cmd := []string{"mas", "uninstall", app.ID}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s # %s\n", command.ShellQuote(cmd), app.Name)

		return nil
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("mas uninstall %q (%s): %w: open Finder or App Store and remove %q manually",
			app.Name, app.ID, ErrManualAppStoreCleanup, app.Name)
	}

	return nil
}
