package secrets

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocanto/mac-os/internal/command"
)

type stubRunner struct {
	outputs map[string][]byte
	calls   *[]string
}

func (r stubRunner) Run(name string, args ...string) ([]byte, error) {
	key := command.ShellQuote(append([]string{name}, args...))

	if r.calls != nil {
		*r.calls = append(*r.calls, key)
	}

	return r.outputs[key], nil
}

func writeTestSecretConfig(t *testing.T, dir string) {
	t.Helper()

	content := []byte(`
secrets:
  - name: gitconfig
    op_field: gitconfig_plaintext
    plaintext_path: stow/git/.config/git/private.gitconfig
    mode: plaintext
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsCallPrefix(calls []string, prefix string) bool {
	for _, call := range calls {
		if strings.HasPrefix(call, prefix) {
			return true
		}
	}

	return false
}

func TestLoadValidatesViperManifest(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)

	cfg, err := Service{Repo: dir}.Load("")

	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Secrets) != 1 {
		t.Fatalf("loaded %d secrets, want 1", len(cfg.Secrets))
	}

	if got := cfg.Secrets[0].Name; got != GitconfigSecret {
		t.Fatalf("secret name = %q, want %q", got, GitconfigSecret)
	}
}

func TestLoadRejectsDuplicateNames(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
secrets:
  - name: gitconfig
    op_field: gitconfig_plaintext
    plaintext_path: stow/git/.config/git/private.gitconfig
    mode: plaintext
  - name: gitconfig
    op_field: other_plaintext
    plaintext_path: stow/git/.config/git/other
    mode: plaintext
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Service{Repo: dir}.Load("")

	if err == nil {
		t.Fatal("expected duplicate secret name error")
	}

	if !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("error = %v, want duplicated", err)
	}
}

func TestLoadRejectsMissingMode(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
secrets:
  - name: gitconfig
    op_field: gitconfig_plaintext
    plaintext_path: stow/git/.config/git/private.gitconfig
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Service{Repo: dir}.Load("")

	if err == nil {
		t.Fatal("expected missing-mode error")
	}

	if !strings.Contains(err.Error(), "plaintext") {
		t.Fatalf("error = %v, want plaintext-mode requirement", err)
	}
}

func TestLoadRejectsTraversalPaths(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
secrets:
  - name: gitconfig
    op_field: gitconfig_plaintext
    plaintext_path: ../escape
    mode: plaintext
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Service{Repo: dir}.Load("")

	if err == nil {
		t.Fatal("expected traversal error")
	}

	if !strings.Contains(err.Error(), "traverse") {
		t.Fatalf("error = %v, want traverse rejection", err)
	}
}

func TestPlaintextPathExpandsHomePrefix(t *testing.T) {
	home, err := os.UserHomeDir()

	if err != nil {
		t.Fatal(err)
	}

	got := Service{Repo: "/tmp/repo"}.PlaintextPath(ManagedSecret{PlaintextPath: "~/.ssh/allowed_signers"})
	want := filepath.Join(home, ".ssh", "allowed_signers")

	if got != want {
		t.Fatalf("PlaintextPath = %q, want %q", got, want)
	}
}

func TestDecryptRequiresOPField(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)
	s := Service{
		Repo: dir,
		Runner: stubRunner{outputs: map[string][]byte{
			`op item get 'Mac Migration Archive' --vault Private --format json`: []byte(`{"fields": []}`),
		}},
	}

	err := s.Decrypt(Options{OPVault: "Private", OPItem: "Mac Migration Archive", SecretTarget: GitconfigSecret})

	if err == nil {
		t.Fatal("expected missing-field error")
	}

	if !strings.Contains(err.Error(), GitconfigPlaintext) {
		t.Fatalf("error = %v, want %s", err, GitconfigPlaintext)
	}
}

func TestDecryptWritesFromOPFieldAndSkipsAge(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)

	var calls []string

	var stdout bytes.Buffer
	s := Service{
		Repo:   dir,
		Stdout: &stdout,
		Runner: stubRunner{
			calls: &calls,
			outputs: map[string][]byte{
				`op item get 'Mac Migration Archive' --vault Private --format json`: []byte(`{
					"fields": [
						{"id": "gitconfig_plaintext", "label": "gitconfig_plaintext", "value": "[user]\n\tname = Private User\n"}
					]
				}`),
			},
		},
	}

	if err := s.Decrypt(Options{OPVault: "Private", OPItem: "Mac Migration Archive", SecretTarget: GitconfigSecret}); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.HasPrefix(call, "age ") {
			t.Fatalf("decrypt unexpectedly called age: %v", calls)
		}
	}

	data, err := os.ReadFile(s.PrivateGitconfigPath())

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "Private User") {
		t.Fatalf("private gitconfig missing field plaintext: %s", string(data))
	}

	if strings.Contains(stdout.String(), "Private User") {
		t.Fatalf("stdout leaked plaintext: %s", stdout.String())
	}
}

func TestDecryptHomePrefixWritesToHomeDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	content := []byte(`
secrets:
  - name: allowed_signers
    op_field: allowed_signers_plaintext
    plaintext_path: ~/.ssh/allowed_signers
    mode: plaintext
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	s := Service{
		Repo:   dir,
		Stdout: &stdout,
		Runner: stubRunner{outputs: map[string][]byte{
			`op item get 'Mac Migration Archive' --vault Private --format json`: []byte(`{
				"fields": [
					{"id": "allowed_signers_plaintext", "label": "allowed_signers_plaintext", "value": "user@example.com ssh-ed25519 AAAA"}
				]
			}`),
		}},
	}

	if err := s.Decrypt(Options{OPVault: "Private", OPItem: "Mac Migration Archive", SecretTarget: "allowed_signers"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".ssh", "allowed_signers"))

	if err != nil {
		t.Fatal(err)
	}

	if got := string(data); got != "user@example.com ssh-ed25519 AAAA\n" {
		t.Fatalf("allowed_signers = %q, want trailing-newline plaintext", got)
	}
}

func TestSyncPushesPlaintextWithoutEncryption(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)

	if err := os.MkdirAll(filepath.Dir(Service{Repo: dir}.PrivateGitconfigPath()), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(Service{Repo: dir}.PrivateGitconfigPath(), []byte("[user]\n\temail = private@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var calls []string

	var stdout bytes.Buffer
	s := Service{
		Repo:   dir,
		Stdout: &stdout,
		Runner: stubRunner{calls: &calls},
	}

	if err := s.Sync(Options{OPVault: "Private", OPItem: "Mac Migration Archive", SecretTarget: GitconfigSecret}); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.HasPrefix(call, "age ") {
			t.Fatalf("sync unexpectedly called age: %v", calls)
		}
	}

	if !containsCallPrefix(calls, "op item edit 'Mac Migration Archive' --vault Private ") {
		t.Fatalf("expected op item edit call, got %v", calls)
	}
}
