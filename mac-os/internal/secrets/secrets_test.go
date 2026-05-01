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
    encrypted_path: stow/git/.config/git/private.gitconfig.age
    mode: age-file
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}

	return false
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
    encrypted_path: stow/git/.config/git/private.gitconfig.age
    mode: age-file
  - name: gitconfig
    op_field: other_plaintext
    plaintext_path: stow/git/.config/git/other
    encrypted_path: stow/git/.config/git/other.age
    mode: age-file
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

func TestLoadRejectsUnsafePaths(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
secrets:
  - name: gitconfig
    op_field: gitconfig_plaintext
    plaintext_path: /tmp/private.gitconfig
    encrypted_path: stow/git/.config/git/private.gitconfig.age
    mode: age-file
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Service{Repo: dir}.Load("")

	if err == nil {
		t.Fatal("expected unsafe path error")
	}

	if !strings.Contains(err.Error(), "repo-relative") {
		t.Fatalf("error = %v, want repo-relative", err)
	}
}

func TestEncryptGitconfigSecretRequiresPlaintextField(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)
	s := Service{
		Repo: dir,
		Runner: stubRunner{outputs: map[string][]byte{
			`op item get 'Mac Migration Archive' --vault Private --format json`: []byte(`{
				"fields": [
					{"id": "archive_age_recipient", "label": "archive_age_recipient", "value": "age1example"}
				]
			}`),
		}},
	}

	err := s.Encrypt(Options{OPVault: "Private", OPItem: "Mac Migration Archive", SecretTarget: GitconfigSecret})

	if err == nil {
		t.Fatal("expected missing gitconfig_plaintext error")
	}

	if !strings.Contains(err.Error(), GitconfigPlaintext) {
		t.Fatalf("error = %v, want %s", err, GitconfigPlaintext)
	}
}

func TestEncryptGitconfigSecretDoesNotPrintPlaintext(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)
	secret := "[user]\n\temail = private@example.com\n"

	var stdout bytes.Buffer

	var calls []string
	s := Service{
		Repo:   dir,
		Stdout: &stdout,
		Runner: stubRunner{
			calls: &calls,
			outputs: map[string][]byte{
				`op item get 'Mac Migration Archive' --vault Private --format json`: []byte(`{
					"fields": [
						{"id": "archive_age_recipient", "label": "archive_age_recipient", "value": "age1example"},
						{"id": "gitconfig_plaintext", "label": "gitconfig_plaintext", "value": "[user]\n\temail = private@example.com\n"}
					]
				}`),
			},
		},
	}

	if err := s.Encrypt(Options{OPVault: "Private", OPItem: "Mac Migration Archive", SecretTarget: GitconfigSecret}); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(stdout.String(), "private@example.com") {
		t.Fatalf("stdout leaked gitconfig plaintext: %s", stdout.String())
	}

	data, err := os.ReadFile(s.PrivateGitconfigPath())

	if err != nil {
		t.Fatal(err)
	}

	if got := string(data); got != secret {
		t.Fatalf("private gitconfig = %q, want %q", got, secret)
	}

	ageCall := command.ShellQuote([]string{"age", "-r", "age1example", "-o", s.EncryptedGitconfigPath(), s.PrivateGitconfigPath()})

	if !containsCall(calls, ageCall) {
		t.Fatalf("calls = %v, want %s", calls, ageCall)
	}
}

func TestDecryptGitconfigFallsBackToPlaintextWhenEncryptedFileMissing(t *testing.T) {
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
			t.Fatalf("decrypt fallback called age unexpectedly: %v", calls)
		}
	}

	data, err := os.ReadFile(s.PrivateGitconfigPath())

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "Private User") {
		t.Fatalf("private gitconfig missing fallback plaintext: %s", string(data))
	}
}

func TestSyncGitconfigUpdatesOnePasswordAndEncryptedFile(t *testing.T) {
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
						{"id": "archive_age_recipient", "label": "archive_age_recipient", "value": "age1example"}
					]
				}`),
			},
		},
	}

	if err := os.MkdirAll(filepath.Dir(s.PrivateGitconfigPath()), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(s.PrivateGitconfigPath(), []byte("[user]\n\temail = private@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.Sync(Options{OPVault: "Private", OPItem: "Mac Migration Archive", SecretTarget: GitconfigSecret}); err != nil {
		t.Fatal(err)
	}

	opPrefix := "op item edit 'Mac Migration Archive' --vault Private "
	ageCall := command.ShellQuote([]string{"age", "-r", "age1example", "-o", s.EncryptedGitconfigPath(), s.PrivateGitconfigPath()})

	if !containsCallPrefix(calls, opPrefix) {
		t.Fatalf("calls = %v, want prefix %s", calls, opPrefix)
	}

	if !containsCall(calls, ageCall) {
		t.Fatalf("calls = %v, want %s", calls, ageCall)
	}
}
