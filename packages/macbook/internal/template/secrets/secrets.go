package secrets

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/safefs"
	"github.com/spf13/viper"
)

type Options struct {
	DryRun       bool
	SecretsPath  string
	SecretTarget string
	OPVault      string
	OPItem       string
}

type Config struct {
	Secrets []ManagedSecret `mapstructure:"secrets" yaml:"secrets"`
}

type ManagedSecret struct {
	Name          string `mapstructure:"name" yaml:"name"`
	OPField       string `mapstructure:"op_field" yaml:"op_field"`
	PlaintextPath string `mapstructure:"plaintext_path" yaml:"plaintext_path"`
	Mode          string `mapstructure:"mode" yaml:"mode"`
}

type Service struct {
	Home   string
	Repo   string
	Stdout io.Writer
	Runner command.Runner
}

const (
	GitconfigPlaintext = "gitconfig_plaintext"
	GitconfigSecret    = "gitconfig"
	ModePlaintext      = "plaintext"
)

func (s Service) Decrypt(opts Options) error {
	secrets, err := s.targets(opts)

	if err != nil {
		return err
	}

	fields, err := command.OnePasswordFields(s.Runner, opts.OPVault, opts.OPItem)

	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if err := s.decryptSecret(opts, fields, secret); err != nil {
			return err
		}
	}

	return nil
}

func (s Service) Sync(opts Options) error {
	secrets, err := s.targets(opts)

	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if err := s.syncSecret(opts, secret); err != nil {
			return err
		}
	}

	return nil
}

func (s Service) PrintStatus(opts Options) {
	cfg, err := s.Load(opts.SecretsPath)

	if err != nil {
		fmt.Fprintf(s.Stdout, "  secret manifest unavailable: %v\n", err)

		return
	}

	fields, err := command.OnePasswordFields(s.Runner, opts.OPVault, opts.OPItem)

	if err != nil {
		fmt.Fprintf(s.Stdout, "  secret fields unavailable: %v\n", err)

		return
	}

	for _, secret := range cfg.Secrets {
		status := "ok"

		if strings.TrimSpace(fields[secret.OPField]) == "" {
			status = "missing 1Password field"
		}

		fmt.Fprintf(s.Stdout, "  secret %-18s %s\n", secret.Name, status)
	}
}

func (s Service) Load(path string) (Config, error) {
	configPath := s.ConfigPath(path)
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read secret config %s: %w", configPath, err)
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse secret config %s: %w", configPath, err)
	}

	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate secret config %s: %w", configPath, err)
	}

	return cfg, nil
}

func (s Service) ConfigPath(path string) string {
	if path == "" {
		return filepath.Join(s.Repo, "secrets.yaml")
	}

	return s.resolveSecretPath(path)
}

func (s Service) PlaintextPath(secret ManagedSecret) string {
	return s.resolveSecretPath(secret.PlaintextPath)
}

func (s Service) resolveSecretPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home := s.Home

		if home == "" {
			if userHome, err := os.UserHomeDir(); err == nil {
				home = userHome
			}
		}

		if home != "" {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(s.Repo, path)
}

func (s Service) PrivateGitconfigPath() string {
	return s.PlaintextPath(ManagedSecret{PlaintextPath: "stow/git/.config/git/private.gitconfig"})
}

func (s Service) targets(opts Options) ([]ManagedSecret, error) {
	cfg, err := s.Load(opts.SecretsPath)

	if err != nil {
		return nil, err
	}

	if opts.SecretTarget == "" {
		return cfg.Secrets, nil
	}

	for _, secret := range cfg.Secrets {
		if secret.Name == opts.SecretTarget {
			return []ManagedSecret{secret}, nil
		}
	}

	return nil, fmt.Errorf("secret target %q not found in %s", opts.SecretTarget, s.ConfigPath(opts.SecretsPath))
}

func (s Service) decryptSecret(opts Options, fields map[string]string, secret ManagedSecret) error {
	plaintext := fields[secret.OPField]

	if strings.TrimSpace(plaintext) == "" {
		return fmt.Errorf("missing %s in 1Password item %q", secret.OPField, opts.OPItem)
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would read %s from 1Password item: %s/%s\n", secret.OPField, opts.OPVault, opts.OPItem)
		fmt.Fprintf(s.Stdout, "would write secret %s: %s\n", secret.Name, s.PlaintextPath(secret))

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.PlaintextPath(secret)), 0o700); err != nil {
		return err
	}

	if err := safefs.WriteFile(s.PlaintextPath(secret), []byte(strings.TrimRight(plaintext, "\n")+"\n"), 0o600); err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "restored secret %s from 1Password at %s\n", secret.Name, s.PlaintextPath(secret))

	return nil
}

func (s Service) syncSecret(opts Options, secret ManagedSecret) error {
	data, err := os.ReadFile(s.PlaintextPath(secret))

	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would update %s in 1Password item: %s/%s\n", secret.OPField, opts.OPVault, opts.OPItem)

		return nil
	}

	args := []string{
		"item", "edit", opts.OPItem,
		"--vault", opts.OPVault,
		secret.OPField + "[concealed]=" + strings.TrimRight(string(data), "\n"),
	}

	out, err := s.Runner.Run("op", args...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("update 1Password secret %s: %w", secret.Name, err)
	}

	fmt.Fprintf(s.Stdout, "synced secret %s to 1Password\n", secret.Name)

	return nil
}

func Validate(cfg Config) error {
	if len(cfg.Secrets) == 0 {
		return errors.New("secrets must contain at least one secret")
	}

	names := map[string]bool{}

	for i, secret := range cfg.Secrets {
		prefix := fmt.Sprintf("secrets[%d]", i)

		if strings.TrimSpace(secret.Name) == "" {
			return fmt.Errorf("%s.name is required", prefix)
		}

		if names[secret.Name] {
			return fmt.Errorf("%s.name %q is duplicated", prefix, secret.Name)
		}

		names[secret.Name] = true

		if strings.TrimSpace(secret.OPField) == "" {
			return fmt.Errorf("%s.op_field is required", prefix)
		}

		if strings.TrimSpace(secret.PlaintextPath) == "" {
			return fmt.Errorf("%s.plaintext_path is required", prefix)
		}

		if secret.Mode != ModePlaintext {
			return fmt.Errorf("%s.mode must be %q", prefix, ModePlaintext)
		}

		if err := validateSafePath(secret.PlaintextPath); err != nil {
			return fmt.Errorf("%s.plaintext_path: %w", prefix, err)
		}
	}

	return nil
}

func validateSafePath(path string) error {
	if strings.HasPrefix(path, "..") || strings.Contains(filepath.Clean(path), "../") {
		return errors.New("must not traverse parent directories")
	}

	return nil
}
