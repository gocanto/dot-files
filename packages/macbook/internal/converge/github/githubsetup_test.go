package github

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocanto/mac-os/internal/command"
)

type stubRunner struct {
	outputs map[string][]byte
	errors  map[string]error
	calls   *[]string
}

func (r stubRunner) Run(name string, args ...string) ([]byte, error) {
	key := command.ShellQuote(append([]string{name}, args...))

	if r.calls != nil {
		*r.calls = append(*r.calls, key)
	}

	return r.outputs[key], r.errors[key]
}

func opFieldsJSON(fields string) []byte {
	return []byte(`{"fields":[` + fields + `]}`)
}

func TestSetupUsesOnePasswordIdentityAndWritesPrivateGitconfig(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}

	pubPath := filepath.Join(home, ".ssh", "id_ed25519_github.pub")

	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAA user@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var calls []string

	var stdout bytes.Buffer
	s := Service{
		Home:     home,
		Repo:     repo,
		Stdin:    strings.NewReader(""),
		Stdout:   &stdout,
		Hostname: "test-mac",
		Runner: stubRunner{
			calls: &calls,
			outputs: map[string][]byte{
				`op item get 'Mac Migration Archive' --vault Private --format json`: opFieldsJSON(`
					{"id":"github_username","value":"gocanto"},
					{"id":"github_email","value":"gustavo@example.com"},
					{"id":"git_author_name","value":"Gustavo Ocanto"}`),
				`gpg --list-secret-keys --with-colons gustavo@example.com`: []byte("sec:u:4096:1:29B2F10879262069:0:::::::\nfpr:::::::::ABCDEF1234567890:\n"),
				`sh -c 'command -v gpg'`:                                   []byte("/opt/homebrew/bin/gpg\n"),
				`gpg --armor --export ABCDEF1234567890`:                    []byte("-----BEGIN PGP PUBLIC KEY BLOCK-----\n"),
			},
		},
	}

	if err := s.Setup(Options{OPVault: "Private", OPItem: "Mac Migration Archive"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(repo, "stow", "git", ".config", "git", "private.gitconfig"))

	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"name = Gustavo Ocanto",
		"email = gustavo@example.com",
		"signingkey = ABCDEF1234567890",
		"program = /opt/homebrew/bin/gpg",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("private gitconfig missing %q:\n%s", want, string(data))
		}
	}

	for _, want := range []string{
		"gh auth status",
		"gh ssh-key add " + pubPath + " --title test-mac-ssh",
		"gh gpg-key add ",
	} {
		if !containsCall(calls, want) {
			t.Fatalf("calls missing %q: %#v", want, calls)
		}
	}
}

func TestSetupPromptsForMissingIdentityFields(t *testing.T) {
	tmp := t.TempDir()

	var stdout bytes.Buffer
	s := Service{
		Home:   filepath.Join(tmp, "home"),
		Repo:   filepath.Join(tmp, "repo"),
		Stdin:  strings.NewReader("gocanto\ngustavo@example.com\nGustavo Ocanto\n"),
		Stdout: &stdout,
		Runner: stubRunner{
			outputs: map[string][]byte{
				`op item get 'Mac Migration Archive' --vault Private --format json`: opFieldsJSON(``),
			},
		},
	}

	if err := s.Setup(Options{DryRun: true, OPVault: "Private", OPItem: "Mac Migration Archive"}); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"GitHub username:", "GitHub email:", "Git author name:", "would upload SSH public key", "would upload GPG public key"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestSetupRejectsEmptyPromptAnswer(t *testing.T) {
	tmp := t.TempDir()

	s := Service{
		Home:   filepath.Join(tmp, "home"),
		Repo:   filepath.Join(tmp, "repo"),
		Stdin:  strings.NewReader("\n"),
		Stdout: &bytes.Buffer{},
		Runner: stubRunner{
			outputs: map[string][]byte{
				`op item get 'Mac Migration Archive' --vault Private --format json`: opFieldsJSON(``),
			},
		},
	}

	err := s.Setup(Options{DryRun: true, OPVault: "Private", OPItem: "Mac Migration Archive"})

	if err == nil {
		t.Fatal("expected prompt error")
	}

	if !strings.Contains(err.Error(), "GitHub username is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestSetupDryRunInstallsMissingToolsWithoutMutating(t *testing.T) {
	tmp := t.TempDir()
	previousLookPath := commandLookPath
	commandLookPath = func(string) (string, error) {
		return "", errors.New("missing")
	}
	t.Cleanup(func() { commandLookPath = previousLookPath })

	var calls []string

	var stdout bytes.Buffer

	s := Service{
		Home:   filepath.Join(tmp, "home"),
		Repo:   filepath.Join(tmp, "repo"),
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Runner: stubRunner{
			calls: &calls,
			outputs: map[string][]byte{
				`op item get 'Mac Migration Archive' --vault Private --format json`: opFieldsJSON(`
					{"id":"github_username","value":"gocanto"},
					{"id":"github_email","value":"gustavo@example.com"},
					{"id":"git_author_name","value":"Gustavo Ocanto"}`),
			},
		},
	}

	if err := s.Setup(Options{DryRun: true, OPVault: "Private", OPItem: "Mac Migration Archive"}); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"would run: brew install git", "would run: brew install gh", "would run: brew install gnupg"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}

	if _, err := os.Stat(filepath.Join(tmp, "repo", "stow", "git", ".config", "git", "private.gitconfig")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote private gitconfig, err=%v", err)
	}
}

func containsCall(calls []string, want string) bool {
	for _, call := range calls {
		if strings.Contains(call, want) {
			return true
		}
	}

	return false
}
