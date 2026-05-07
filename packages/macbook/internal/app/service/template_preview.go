package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/template/appconfig"
	"github.com/gocanto/dot-files/internal/template/brewfile"
	templatemacos "github.com/gocanto/dot-files/internal/template/macos"
	"github.com/gocanto/dot-files/internal/template/secrets"
)

func (s Service) PreviewTemplateBrew(_ Options) error {
	fmt.Fprintln(s.Stdout, "# Tracked Homebrew bundle")
	fmt.Fprint(s.Stdout, brewfile.Content())

	return nil
}

func (s Service) PreviewTemplateApps(opts Options) error {
	loader := appconfig.Loader{Home: s.Home, Repo: s.Repo}
	cfg, err := loader.Load(opts.ConfigPath)

	if err != nil {
		return err
	}

	groups := map[string][]string{}

	for _, app := range cfg.Apps {
		groups[app.InstallMethod] = append(groups[app.InstallMethod], app.Name)
	}

	keys := make([]string, 0, len(groups))

	for key := range groups {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	fmt.Fprintln(s.Stdout, "# Tracked apps from apps.yaml")

	for _, method := range keys {
		names := groups[method]

		sort.Strings(names)

		fmt.Fprintf(s.Stdout, "[%s] (%d)\n", method, len(names))

		for _, name := range names {
			fmt.Fprintf(s.Stdout, "  - %s\n", name)
		}
	}

	return nil
}

func (s Service) PreviewTemplateMacOS(_ Options) error {
	fmt.Fprintln(s.Stdout, "# Tracked macOS settings")

	for _, setting := range templatemacos.Settings() {
		fmt.Fprintf(s.Stdout, "  %s %s %s\n", setting.Domain, setting.Key, command.ShellQuote(setting.Args))
	}

	return nil
}

func (s Service) PreviewTemplateDotfiles(_ Options) error {
	stowDir := filepath.Join(s.Repo, "stow")

	fmt.Fprintf(s.Stdout, "reading stow directory: %s\n", stowDir)

	entries, err := os.ReadDir(stowDir)

	if err != nil {
		return fmt.Errorf("read stow dir %s: %w", stowDir, err)
	}

	fmt.Fprintf(s.Stdout, "found %d stow %s\n", len(entries), plural(len(entries), "entry", "entries"))
	fmt.Fprintln(s.Stdout, "# Tracked dotfile bundles under stow/")

	bundles := 0

	for _, entry := range entries {
		fmt.Fprintf(s.Stdout, "checking stow entry: %s\n", entry.Name())

		if !entry.IsDir() {
			fmt.Fprintf(s.Stdout, "  skipped non-directory: %s\n", entry.Name())

			continue
		}

		fmt.Fprintf(s.Stdout, "  - %s\n", entry.Name())
		bundles++
	}

	if bundles == 0 {
		fmt.Fprintln(s.Stdout, "  (no dotfile bundle directories found)")
	}

	fmt.Fprintf(s.Stdout, "done: found %d dotfile %s\n", bundles, plural(bundles, "bundle", "bundles"))

	return nil
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}

func (s Service) ValidateTemplate(opts Options) error {
	loader := appconfig.Loader{Home: s.Home, Repo: s.Repo}

	if _, err := loader.Load(opts.ConfigPath); err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "ok: apps.yaml at %s parses and validates\n", loader.Path(opts.ConfigPath))

	secretsSvc := secrets.Service{Home: s.Home, Repo: s.Repo}

	if _, err := secretsSvc.Load(opts.SecretsPath); err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "ok: secrets.yaml at %s parses and validates\n", secretsSvc.ConfigPath(opts.SecretsPath))

	stowDir := filepath.Join(s.Repo, "stow")
	info, err := os.Stat(stowDir)

	if err != nil {
		return fmt.Errorf("stow dir %s: %w", stowDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("stow path %s is not a directory", stowDir)
	}

	fmt.Fprintf(s.Stdout, "ok: stow dir %s exists\n", stowDir)
	fmt.Fprintf(s.Stdout, "ok: tracked Brewfile lists %d formulae and %d casks\n",
		len(brewfile.TrackedFormulae()), len(brewfile.TrackedCasks()))
	fmt.Fprintf(s.Stdout, "ok: tracked macOS settings cover %d entries across %d domains\n",
		len(templatemacos.Settings()), len(templatemacos.Domains()))

	return nil
}
