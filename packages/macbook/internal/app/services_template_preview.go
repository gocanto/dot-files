package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/template/appconfig"
	"github.com/gocanto/mac-os/internal/template/brewfile"
	templatemacos "github.com/gocanto/mac-os/internal/template/macos"
	"github.com/gocanto/mac-os/internal/template/secrets"
)

func (a app) previewTemplateBrew(_ options) error {
	fmt.Fprintln(a.stdout, "# Tracked Homebrew bundle")
	fmt.Fprint(a.stdout, brewfile.Content())

	return nil
}

func (a app) previewTemplateApps(opts options) error {
	loader := appconfig.Loader{Home: a.home, Repo: a.repo}
	cfg, err := loader.Load(opts.configPath)

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

	fmt.Fprintln(a.stdout, "# Tracked apps from apps.yaml")

	for _, method := range keys {
		names := groups[method]

		sort.Strings(names)

		fmt.Fprintf(a.stdout, "[%s] (%d)\n", method, len(names))

		for _, name := range names {
			fmt.Fprintf(a.stdout, "  - %s\n", name)
		}
	}

	return nil
}

func (a app) previewTemplateMacOS(_ options) error {
	fmt.Fprintln(a.stdout, "# Tracked macOS settings")

	for _, setting := range templatemacos.Settings() {
		fmt.Fprintf(a.stdout, "  %s %s %s\n", setting.Domain, setting.Key, command.ShellQuote(setting.Args))
	}

	return nil
}

func (a app) previewTemplateDotfiles(_ options) error {
	stowDir := filepath.Join(a.repo, "stow")

	fmt.Fprintf(a.stdout, "reading stow directory: %s\n", stowDir)

	entries, err := os.ReadDir(stowDir)

	if err != nil {
		return fmt.Errorf("read stow dir %s: %w", stowDir, err)
	}

	fmt.Fprintf(a.stdout, "found %d stow %s\n", len(entries), plural(len(entries), "entry", "entries"))
	fmt.Fprintln(a.stdout, "# Tracked dotfile bundles under stow/")

	bundles := 0

	for _, entry := range entries {
		fmt.Fprintf(a.stdout, "checking stow entry: %s\n", entry.Name())

		if !entry.IsDir() {
			fmt.Fprintf(a.stdout, "  skipped non-directory: %s\n", entry.Name())

			continue
		}

		fmt.Fprintf(a.stdout, "  - %s\n", entry.Name())
		bundles++
	}

	if bundles == 0 {
		fmt.Fprintln(a.stdout, "  (no dotfile bundle directories found)")
	}

	fmt.Fprintf(a.stdout, "done: found %d dotfile %s\n", bundles, plural(bundles, "bundle", "bundles"))

	return nil
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}

func (a app) validateTemplate(opts options) error {
	loader := appconfig.Loader{Home: a.home, Repo: a.repo}

	if _, err := loader.Load(opts.configPath); err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "ok: apps.yaml at %s parses and validates\n", loader.Path(opts.configPath))

	secretsSvc := secrets.Service{Home: a.home, Repo: a.repo}

	if _, err := secretsSvc.Load(opts.secretsPath); err != nil {
		return err
	}

	fmt.Fprintf(a.stdout, "ok: secrets.yaml at %s parses and validates\n", secretsSvc.ConfigPath(opts.secretsPath))

	stowDir := filepath.Join(a.repo, "stow")
	info, err := os.Stat(stowDir)

	if err != nil {
		return fmt.Errorf("stow dir %s: %w", stowDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("stow path %s is not a directory", stowDir)
	}

	fmt.Fprintf(a.stdout, "ok: stow dir %s exists\n", stowDir)
	fmt.Fprintf(a.stdout, "ok: tracked Brewfile lists %d formulae and %d casks\n",
		len(brewfile.TrackedFormulae()), len(brewfile.TrackedCasks()))
	fmt.Fprintf(a.stdout, "ok: tracked macOS settings cover %d entries across %d domains\n",
		len(templatemacos.Settings()), len(templatemacos.Domains()))

	return nil
}
