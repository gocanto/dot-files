package appconfig

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/safefs"
	"github.com/spf13/viper"
)

type Config struct {
	Apps []ManagedApp `mapstructure:"apps" yaml:"apps"`
}

type ManagedApp struct {
	Name              string       `mapstructure:"name" yaml:"name"`
	BundleID          string       `mapstructure:"bundle_id" yaml:"bundle_id"`
	InstallMethod     string       `mapstructure:"install_method" yaml:"install_method"`
	Package           string       `mapstructure:"package" yaml:"package"`
	ConfigMode        string       `mapstructure:"config_mode" yaml:"config_mode"`
	ConfigPaths       []ConfigPath `mapstructure:"config_paths" yaml:"config_paths"`
	OnePasswordFields []string     `mapstructure:"onepassword_fields" yaml:"onepassword_fields"`
}

type ConfigPath struct {
	Source string `mapstructure:"source" yaml:"source"`
	Target string `mapstructure:"target" yaml:"target"`
}

type Loader struct {
	Home string
	Repo string
}

func (l Loader) Load(path string) (Config, error) {
	configPath := l.Path(path)
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read app config %s: %w", configPath, err)
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse app config %s: %w", configPath, err)
	}

	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate app config %s: %w", configPath, err)
	}

	return cfg, nil
}

func (l Loader) Path(path string) string {
	if path == "" {
		return filepath.Join(l.Repo, "apps.yaml")
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(l.Home, strings.TrimPrefix(path, "~/"))
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(l.Repo, path)
}

func Validate(cfg Config) error {
	if len(cfg.Apps) == 0 {
		return errors.New("apps must contain at least one app")
	}

	installModes := map[string]bool{"brew": true, "mas": true, "manual": true, "system": true}
	configModes := map[string]bool{"auto": true, "reference": true, "manual": true}

	for i, app := range cfg.Apps {
		prefix := fmt.Sprintf("apps[%d]", i)

		if strings.TrimSpace(app.Name) == "" {
			return fmt.Errorf("%s.name is required", prefix)
		}

		if !installModes[app.InstallMethod] {
			return fmt.Errorf("%s.install_method %q is invalid", prefix, app.InstallMethod)
		}

		if !configModes[app.ConfigMode] {
			return fmt.Errorf("%s.config_mode %q is invalid", prefix, app.ConfigMode)
		}

		if (app.InstallMethod == "brew" || app.InstallMethod == "mas") && strings.TrimSpace(app.Package) == "" {
			return fmt.Errorf("%s.package is required for %s installs", prefix, app.InstallMethod)
		}

		if app.ConfigMode == "auto" && len(app.ConfigPaths) == 0 {
			return fmt.Errorf("%s.config_paths is required for auto config restore", prefix)
		}

		for j, path := range app.ConfigPaths {
			pathPrefix := fmt.Sprintf("%s.config_paths[%d]", prefix, j)

			if strings.TrimSpace(path.Source) == "" {
				return fmt.Errorf("%s.source is required", pathPrefix)
			}

			if strings.TrimSpace(path.Target) == "" {
				return fmt.Errorf("%s.target is required", pathPrefix)
			}

			if filepath.IsAbs(path.Target) || strings.HasPrefix(path.Target, "..") || strings.Contains(filepath.Clean(path.Target), "../") {
				return fmt.Errorf("%s.target must be archive-relative", pathPrefix)
			}
		}
	}

	return nil
}

func CapturePlan(cfg Config) []safefs.Item {
	items := []safefs.Item{}

	for _, app := range cfg.Apps {
		if app.ConfigMode == "manual" {
			continue
		}

		for _, path := range app.ConfigPaths {
			items = append(items, safefs.Item{Source: path.Source, Target: path.Target})
		}
	}

	return items
}
