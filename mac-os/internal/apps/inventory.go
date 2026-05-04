package apps

import (
	"fmt"
	"path/filepath"

	"github.com/gocanto/mac-os/internal/appconfig"
	"github.com/gocanto/mac-os/internal/safefs"
	"go.yaml.in/yaml/v3"
)

type appInventory struct {
	Bundles []installedBundle
	Casks   []string
	MAS     []masInstall
}

type installedBundle struct {
	Name     string
	BundleID string
	Path     string
	System   bool
}

type masInstall struct {
	ID   string
	Name string
}

func (s Service) GenerateInstalledList(opts Options) error {
	cfg, err := s.loader().Load(opts.ConfigPath)

	if err != nil {
		return err
	}

	summary, err := s.generateInstalledList(cfg, opts)

	if err != nil {
		return err
	}

	if opts.DryRun {
		printMergeSummary(s.Stdout, summary, true)

		return nil
	}

	out, err := yaml.Marshal(appconfig.Config{Apps: summary.Generated})

	if err != nil {
		return fmt.Errorf("encode generated app list: %w", err)
	}

	if err := safefs.WriteFile(summary.Output, out, 0o600); err != nil {
		return fmt.Errorf("write generated app list %s: %w", summary.Output, err)
	}

	printMergeSummary(s.Stdout, summary, false)

	return nil
}

func (s Service) generateInstalledList(cfg appconfig.Config, opts Options) (mergeSummary, error) {
	inventory, warnings, err := s.scanInventory(opts)

	if err != nil {
		return mergeSummary{}, err
	}

	summary := mergeApps(cfg, inventory)
	summary.Warnings = warnings
	summary.Output = s.generatedPath(opts.GeneratedPath)

	return summary, nil
}

func (s Service) scanInventory(opts Options) (appInventory, []string, error) {
	var warnings []string

	bundles, err := scanAppBundles(appRoots(opts, s.Home))

	if err != nil {
		return appInventory{}, nil, err
	}

	casksOut, err := s.Runner.Run("brew", "list", "--cask")
	casks := parseBrewCasks(casksOut)

	if err != nil {
		warnings = append(warnings, fmt.Sprintf("brew cask inventory failed: %v", err))
	}

	masOut, err := s.Runner.Run("mas", "list")
	masApps := parseMASList(masOut)

	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Mac App Store inventory failed: %v", err))
	}

	return appInventory{Bundles: bundles, Casks: casks, MAS: masApps}, warnings, nil
}

func appRoots(opts Options, home string) []string {
	if len(opts.AppRoots) > 0 {
		roots := make([]string, 0, len(opts.AppRoots))

		for _, root := range opts.AppRoots {
			roots = append(roots, safefs.ExpandHome(root, home))
		}

		return roots
	}

	return []string{"/Applications", filepath.Join(home, "Applications"), "/System/Applications"}
}

func (s Service) generatedPath(path string) string {
	if path == "" {
		return filepath.Join(s.Repo, "apps.generated.yaml")
	}

	return s.loader().Path(path)
}
