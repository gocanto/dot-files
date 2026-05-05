package appstore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/safefs"
	"github.com/gocanto/mac-os/internal/template/appconfig"
)

type Options struct {
	DryRun        bool
	Apps          bool
	ArchivePath   string
	ConfigPath    string
	GeneratedPath string
	AppRoots      []string
}

type Service struct {
	Home   string
	Repo   string
	Stdout io.Writer
	Runner command.Runner
}

func (s Service) loader() appconfig.Loader {
	return appconfig.Loader{Home: s.Home, Repo: s.Repo}
}

func (s Service) ApplyAppStore(opts Options) error {
	if !opts.Apps {
		fmt.Fprintln(s.Stdout, "skipped: run with --apps to install App Store apps")

		return nil
	}

	cfg, err := s.loader().Load(opts.ConfigPath)

	if err != nil {
		return err
	}

	for _, app := range cfg.Apps {
		if app.InstallMethod != "mas" {
			continue
		}

		cmd := []string{"mas", "install", app.Package}

		if opts.DryRun {
			fmt.Fprintf(s.Stdout, "would run: %s # %s\n", command.ShellQuote(cmd), app.Name)

			continue
		}

		if err := command.RunInteractive(s.Runner, s.Stdout, cmd[0], cmd[1:]...); err != nil {
			return fmt.Errorf("install App Store app %q: %w", app.Name, err)
		}
	}

	return nil
}

func (s Service) ReportManual(opts Options) error {
	if !opts.Apps {
		fmt.Fprintln(s.Stdout, "skipped: run with --apps to report manual apps")

		return nil
	}

	cfg, err := s.loader().Load(opts.ConfigPath)

	if err != nil {
		return err
	}

	for _, app := range cfg.Apps {
		switch app.InstallMethod {
		case "manual":
			fmt.Fprintf(s.Stdout, "manual install required: %s", app.Name)

			if app.Package != "" {
				fmt.Fprintf(s.Stdout, " (%s)", app.Package)
			}

			fmt.Fprintln(s.Stdout)
		case "brew":
			fmt.Fprintf(s.Stdout, "brew-managed app: %s (%s)\n", app.Name, app.Package)
		case "system":
			fmt.Fprintf(s.Stdout, "system app: %s\n", app.Name)
		}
	}

	return nil
}

func (s Service) CaptureConfigs(root string, opts Options) error {
	cfg, err := s.loader().Load(opts.ConfigPath)

	if err != nil {
		return err
	}

	if err := safefs.CopyFile(s.loader().Path(opts.ConfigPath), filepath.Join(root, "apps.yaml")); err != nil {
		return err
	}

	for _, item := range appconfig.CapturePlan(cfg) {
		source := safefs.ExpandHome(item.Source, s.Home)

		if safefs.ShouldSkipSensitive(source) || safefs.ShouldSkipSensitive(item.Target) {
			fmt.Fprintf(s.Stdout, "skipped sensitive app config: %s\n", item.Source)

			continue
		}

		info, err := os.Stat(source)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(s.Stdout, "missing app config, skipped: %s\n", item.Source)

				continue
			}

			return err
		}

		target := filepath.Join(root, item.Target)

		if info.IsDir() {
			if err := safefs.CopyDirSafe(source, target); err != nil {
				return err
			}

			fmt.Fprintf(s.Stdout, "captured app config: %s\n", item.Target)

			continue
		}

		data, err := os.ReadFile(source)

		if err != nil {
			return err
		}

		data = safefs.SanitizeDotfile(source, s.Home, data)

		if err := safefs.WriteFile(target, data, 0o600); err != nil {
			return err
		}

		fmt.Fprintf(s.Stdout, "captured app config: %s\n", item.Target)
	}

	return nil
}

func (s Service) RestoreConfigs(opts Options) error {
	if !opts.Apps {
		fmt.Fprintln(s.Stdout, "skipped: run with --apps to restore app configs")

		return nil
	}

	if opts.ArchivePath == "" {
		fmt.Fprintln(s.Stdout, "skipped: no --archive supplied for app config restore")

		return nil
	}

	cfg, err := s.loader().Load(opts.ConfigPath)

	if err != nil {
		return err
	}

	archive := safefs.ExpandHome(opts.ArchivePath, s.Home)

	for _, app := range cfg.Apps {
		switch app.ConfigMode {
		case "auto":
			for _, path := range app.ConfigPaths {
				source := filepath.Join(archive, path.Target)
				target := safefs.ExpandHome(path.Source, s.Home)

				if safefs.ShouldSkipSensitive(source) || safefs.ShouldSkipSensitive(target) {
					fmt.Fprintf(s.Stdout, "skipped sensitive restore path: %s\n", path.Source)

					continue
				}

				info, err := os.Stat(source)

				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						fmt.Fprintf(s.Stdout, "missing archive config, skipped: %s\n", path.Target)

						continue
					}

					return err
				}

				if opts.DryRun {
					fmt.Fprintf(s.Stdout, "would restore app config: %s -> %s\n", source, target)

					continue
				}

				if info.IsDir() {
					if err := safefs.CopyDirSafe(source, target); err != nil {
						return err
					}
				} else if err := safefs.CopyFile(source, target); err != nil {
					return err
				}

				fmt.Fprintf(s.Stdout, "restored app config: %s\n", target)
			}
		case "reference":
			fmt.Fprintf(s.Stdout, "reference only: %s\n", app.Name)
		case "manual":
			fmt.Fprintf(s.Stdout, "manual config restore: %s\n", app.Name)
		}
	}

	return nil
}
