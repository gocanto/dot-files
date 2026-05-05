package github

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/safefs"
)

func (s Service) ensureSSHKey(dryRun bool, identity Identity) (string, error) {
	keyPath := filepath.Join(s.Home, ".ssh", "id_ed25519_github")
	pubPath := keyPath + ".pub"

	if _, err := os.Stat(pubPath); err == nil {
		fmt.Fprintf(s.Stdout, "SSH public key found: %s\n", pubPath)

		return pubPath, nil
	}

	cmd := []string{"ssh-keygen", "-t", "ed25519", "-C", identity.GitHubEmail, "-f", keyPath, "-N", ""}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

		return pubPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return "", err
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return "", fmt.Errorf("generate SSH key: %w", err)
	}

	return pubPath, nil
}

func (s Service) ensureGPGKey(dryRun bool, identity Identity) (string, error) {
	if key := s.findGPGKey(identity.GitHubEmail); key != "" {
		fmt.Fprintf(s.Stdout, "GPG signing key found: %s\n", key)

		return key, nil
	}

	batch := fmt.Sprintf(`Key-Type: RSA
Key-Length: 4096
Key-Usage: sign
Name-Real: %s
Name-Email: %s
Expire-Date: 0
%%no-protection
%%commit
`, identity.GitAuthorName, identity.GitHubEmail)

	cmd := []string{"gpg", "--batch", "--generate-key"}

	if dryRun {
		fmt.Fprintf(s.Stdout, "would run: %s for %s\n", command.ShellQuote(cmd), identity.GitHubEmail)

		return "<GPG_SIGNING_KEY>", nil
	}

	out, err := s.Runner.Run("sh", "-c", "gpg --batch --generate-key <<'EOF'\n"+batch+"EOF")

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return "", fmt.Errorf("generate GPG key: %w", err)
	}

	key := s.findGPGKey(identity.GitHubEmail)

	if key == "" {
		return "", fmt.Errorf("generated GPG key for %s but could not resolve its fingerprint", identity.GitHubEmail)
	}

	return key, nil
}

func (s Service) findGPGKey(email string) string {
	out, err := s.Runner.Run("gpg", "--list-secret-keys", "--with-colons", email)

	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(line, ":")

		if len(parts) > 9 && parts[0] == "fpr" && strings.TrimSpace(parts[9]) != "" {
			return strings.TrimSpace(parts[9])
		}
	}

	return ""
}

func (s Service) writePrivateGitconfig(dryRun bool, identity Identity, gpgKey string) error {
	path := filepath.Join(s.Repo, "stow", "git", ".config", "git", "private.gitconfig")
	gpgProgram := "/opt/homebrew/bin/gpg"

	if !fileExists(gpgProgram) {
		if out, err := s.Runner.Run("sh", "-c", "command -v gpg"); err == nil && strings.TrimSpace(string(out)) != "" {
			gpgProgram = strings.TrimSpace(string(out))
		}
	}

	data := fmt.Sprintf(`[user]
	name = %s
	email = %s
	signingkey = %s
[gpg]
	program = %s
`, identity.GitAuthorName, identity.GitHubEmail, gpgKey, gpgProgram)

	if dryRun {
		fmt.Fprintf(s.Stdout, "would write private Git config: %s\n", path)

		return nil
	}

	if err := safefs.WriteFile(path, []byte(data), 0o600); err != nil {
		return err
	}

	fmt.Fprintf(s.Stdout, "updated private Git config: %s\n", path)

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func (s Service) uploadSSHKey(pubPath string) error {
	pub, err := os.ReadFile(pubPath)

	if err != nil {
		return err
	}

	if out, err := s.Runner.Run("gh", "ssh-key", "list"); err == nil && strings.Contains(string(out), strings.TrimSpace(string(pub))) {
		fmt.Fprintln(s.Stdout, "GitHub SSH public key already registered")

		return nil
	}

	title := s.keyTitle("ssh")
	cmd := []string{"gh", "ssh-key", "add", pubPath, "--title", title}
	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("upload SSH public key: %w", err)
	}

	return nil
}

func (s Service) uploadGPGKey(key string) error {
	if out, err := s.Runner.Run("gh", "gpg-key", "list"); err == nil && containsGPGKey(string(out), key) {
		fmt.Fprintln(s.Stdout, "GitHub GPG public key already registered")

		return nil
	}

	out, err := s.Runner.Run("gpg", "--armor", "--export", key)

	if err != nil {
		return fmt.Errorf("export GPG public key: %w", err)
	}

	tmp, err := os.CreateTemp("", "github-gpg-key-*.asc")

	if err != nil {
		return err
	}

	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, bytes.NewReader(out)); err != nil {
		tmp.Close()

		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	cmd := []string{"gh", "gpg-key", "add", tmp.Name(), "--title", s.keyTitle("gpg")}
	uploadOut, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(uploadOut) > 0 {
		fmt.Fprint(s.Stdout, string(uploadOut))
	}

	if err != nil {
		return fmt.Errorf("upload GPG public key: %w", err)
	}

	return nil
}

func containsGPGKey(list, key string) bool {
	key = strings.TrimSpace(key)

	if key == "" {
		return false
	}

	if strings.Contains(list, key) {
		return true
	}

	if len(key) > 16 && strings.Contains(list, key[len(key)-16:]) {
		return true
	}

	return false
}

func (s Service) keyTitle(kind string) string {
	host := s.Hostname

	if host == "" {
		if detected, err := os.Hostname(); err == nil {
			host = detected
		}
	}

	if host == "" {
		host = "mac"
	}

	return fmt.Sprintf("%s-%s", host, kind)
}
