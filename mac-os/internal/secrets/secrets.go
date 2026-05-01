package secrets

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/safefs"
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
	Secrets []ManagedSecret `mapstructure:"secrets"`
}

type ManagedSecret struct {
	Name          string `mapstructure:"name"`
	OPField       string `mapstructure:"op_field"`
	PlaintextPath string `mapstructure:"plaintext_path"`
	EncryptedPath string `mapstructure:"encrypted_path"`
	Mode          string `mapstructure:"mode"`
}

type Service struct {
	Repo   string
	Stdout io.Writer
	Runner command.Runner
}

const (
	GitconfigPlaintext = "gitconfig_plaintext"
	GitconfigSecret    = "gitconfig"
	ModeAgeFile        = "age-file"
)

func (s Service) Encrypt(opts Options) error {
	secrets, err := s.targets(opts)

	if err != nil {
		return err
	}

	fields, err := command.OnePasswordFields(s.Runner, opts.OPVault, opts.OPItem)

	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if err := s.encryptSecret(opts, fields, secret); err != nil {
			return err
		}
	}

	return nil
}

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

	fields, err := command.OnePasswordFields(s.Runner, opts.OPVault, opts.OPItem)

	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if err := s.syncSecret(opts, fields, secret); err != nil {
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
		} else if _, err := os.Stat(s.EncryptedPath(secret)); errors.Is(err, os.ErrNotExist) {
			status = "missing encrypted file"
		} else if err != nil {
			status = "encrypted file unreadable"
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

	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()

		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(s.Repo, path)
}

func (s Service) PlaintextPath(secret ManagedSecret) string {
	return filepath.Join(s.Repo, secret.PlaintextPath)
}

func (s Service) EncryptedPath(secret ManagedSecret) string {
	return filepath.Join(s.Repo, secret.EncryptedPath)
}

func (s Service) PrivateGitconfigPath() string {
	return s.PlaintextPath(ManagedSecret{PlaintextPath: "stow/git/.config/git/private.gitconfig"})
}

func (s Service) EncryptedGitconfigPath() string {
	return s.EncryptedPath(ManagedSecret{EncryptedPath: "stow/git/.config/git/private.gitconfig.age"})
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

func (s Service) encryptSecret(opts Options, fields map[string]string, secret ManagedSecret) error {
	plaintext := fields[secret.OPField]

	if strings.TrimSpace(plaintext) == "" {
		return fmt.Errorf("missing %s in 1Password item %q", secret.OPField, opts.OPItem)
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would read %s from 1Password item: %s/%s\n", secret.OPField, opts.OPVault, opts.OPItem)
		fmt.Fprintf(s.Stdout, "would write ignored secret %s: %s\n", secret.Name, s.PlaintextPath(secret))
		fmt.Fprintf(s.Stdout, "would encrypt secret %s with Age recipient from 1Password: %s\n", secret.Name, s.EncryptedPath(secret))

		return nil
	}

	if err := safefs.WriteFile(s.PlaintextPath(secret), []byte(strings.TrimRight(plaintext, "\n")+"\n"), 0o600); err != nil {
		return err
	}

	if err := s.encryptSecretFile(opts, fields, secret); err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "encrypted secret %s at %s\n", secret.Name, s.EncryptedPath(secret))

	return nil
}

func (s Service) decryptSecret(opts Options, fields map[string]string, secret ManagedSecret) error {
	encryptedPath := s.EncryptedPath(secret)

	if _, err := os.Stat(encryptedPath); err == nil {
		if opts.DryRun {
			fmt.Fprintf(s.Stdout, "would decrypt secret %s with Age identity from 1Password: %s\n", secret.Name, encryptedPath)
			fmt.Fprintf(s.Stdout, "would write ignored secret %s: %s\n", secret.Name, s.PlaintextPath(secret))

			return nil
		}

		if err := s.decryptSecretFile(opts, fields, secret); err != nil {
			return err
		}

		fmt.Fprintf(s.Stdout, "decrypted secret %s at %s\n", secret.Name, s.PlaintextPath(secret))

		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	plaintext := fields[secret.OPField]

	if strings.TrimSpace(plaintext) == "" {
		return fmt.Errorf("missing %s in 1Password item %q", secret.OPField, opts.OPItem)
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would read %s from 1Password item: %s/%s\n", secret.OPField, opts.OPVault, opts.OPItem)
		fmt.Fprintf(s.Stdout, "would write ignored secret %s: %s\n", secret.Name, s.PlaintextPath(secret))

		return nil
	}

	if err := safefs.WriteFile(s.PlaintextPath(secret), []byte(strings.TrimRight(plaintext, "\n")+"\n"), 0o600); err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "restored secret %s from 1Password at %s\n", secret.Name, s.PlaintextPath(secret))

	return nil
}

func (s Service) syncSecret(opts Options, fields map[string]string, secret ManagedSecret) error {
	data, err := os.ReadFile(s.PlaintextPath(secret))

	if err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would update %s in 1Password item: %s/%s\n", secret.OPField, opts.OPVault, opts.OPItem)
		fmt.Fprintf(s.Stdout, "would encrypt secret %s with Age recipient from 1Password: %s\n", secret.Name, s.EncryptedPath(secret))

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

	if err := s.encryptSecretFile(opts, fields, secret); err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "synced secret %s to 1Password and %s\n", secret.Name, s.EncryptedPath(secret))

	return nil
}

func (s Service) encryptSecretFile(opts Options, fields map[string]string, secret ManagedSecret) error {
	recipient := strings.TrimSpace(fields["archive_age_recipient"])

	if recipient == "" {
		return fmt.Errorf("missing archive_age_recipient in 1Password item %q", opts.OPItem)
	}

	if err := os.MkdirAll(filepath.Dir(s.EncryptedPath(secret)), 0o700); err != nil {
		return err
	}

	out, err := s.Runner.Run("age", "-r", recipient, "-o", s.EncryptedPath(secret), s.PlaintextPath(secret))

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("encrypt secret %s: %w", secret.Name, err)
	}

	return nil
}

func (s Service) decryptSecretFile(opts Options, fields map[string]string, secret ManagedSecret) error {
	identity := strings.TrimSpace(fields["archive_age_identity"])

	if identity == "" {
		return fmt.Errorf("missing archive_age_identity in 1Password item %q", opts.OPItem)
	}

	tmp, err := os.CreateTemp("", "mac-os-age-identity-*")

	if err != nil {
		return err
	}

	tmpPath := tmp.Name()

	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()

		return err
	}

	if _, err := tmp.WriteString(identity + "\n"); err != nil {
		tmp.Close()

		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.PlaintextPath(secret)), 0o700); err != nil {
		return err
	}

	out, err := s.Runner.Run("age", "-d", "-i", tmpPath, "-o", s.PlaintextPath(secret), s.EncryptedPath(secret))

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("decrypt secret %s: %w", secret.Name, err)
	}

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

		if strings.TrimSpace(secret.EncryptedPath) == "" {
			return fmt.Errorf("%s.encrypted_path is required", prefix)
		}

		if err := validateRepoRelativePath(secret.PlaintextPath); err != nil {
			return fmt.Errorf("%s.plaintext_path: %w", prefix, err)
		}

		if err := validateRepoRelativePath(secret.EncryptedPath); err != nil {
			return fmt.Errorf("%s.encrypted_path: %w", prefix, err)
		}

		if secret.Mode != ModeAgeFile {
			return fmt.Errorf("%s.mode must be %q", prefix, ModeAgeFile)
		}
	}

	return nil
}

func validateRepoRelativePath(path string) error {
	if filepath.IsAbs(path) || strings.HasPrefix(path, "..") || strings.Contains(filepath.Clean(path), "../") {
		return errors.New("must be repo-relative")
	}

	return nil
}
