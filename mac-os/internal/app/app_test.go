package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubRunner struct {
	outputs map[string][]byte
}

func (r stubRunner) Run(name string, args ...string) ([]byte, error) {
	return r.outputs[shellQuote(append([]string{name}, args...))], nil
}

func TestShouldSkipSensitive(t *testing.T) {
	cases := map[string]bool{
		"/Users/gus/.ssh/id_ed25519":                                                 true,
		"/Users/gus/.ssh/id_ed25519.pub":                                             false,
		"/Users/gus/.zsh_history":                                                    true,
		"/Users/gus/.config/gh/hosts.yml":                                            true,
		"/Users/gus/.config/ghostty/config":                                          false,
		"/Users/gus/Library/Application Support/Code/User/settings.json":             false,
		"/Users/gus/Library/Application Support/Code/User/globalStorage/state.vscdb": true,
		"/Users/gus/.claude/file-history/abc/def":                                    true,
	}

	for path, want := range cases {
		if got := shouldSkipSensitive(path); got != want {
			t.Fatalf("shouldSkipSensitive(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestBrewfileIncludesDevToolsAndStow(t *testing.T) {
	content := brewfileContent()

	for _, want := range []string{
		`brew "stow"`,
		`brew "age"`,
		`brew "agent-browser"`,
		`brew "codex"`,
		`brew "claude-code"`,
		`brew "opencode"`,
		`brew "node@24"`,
		`brew "mysql"`,
		`brew "libpq"`,
		`cask "docker"`,
		`cask "visual-studio-code"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("Brewfile missing %s\n%s", want, content)
		}
	}
}

func TestOnePasswordFieldsParsesIDAndLabel(t *testing.T) {
	a := app{
		runner: stubRunner{outputs: map[string][]byte{
			`op item get 'Mac Migration Archive' --vault Private --format json`: []byte(`{
				"fields": [
					{"id": "archive_root", "label": "archive_root", "value": "/Volumes/Migration"},
					{"id": "archive_age_recipient", "label": "archive_age_recipient", "value": "age1example"}
				]
			}`),
		}},
	}

	fields, err := a.onePasswordFields(options{opVault: defaultOPVault, opItem: defaultOPItem})

	if err != nil {
		t.Fatal(err)
	}

	if got := fields["archive_root"]; got != "/Volumes/Migration" {
		t.Fatalf("archive_root = %q", got)
	}

	if got := fields["archive_age_recipient"]; got != "age1example" {
		t.Fatalf("archive_age_recipient = %q", got)
	}
}

func TestCaptureDryRunShowsEncryptionPlan(t *testing.T) {
	var stdout bytes.Buffer
	a := app{
		home:   "/Users/gus",
		repo:   "/repo",
		stdout: &stdout,
		runner: stubRunner{},
	}

	if err := a.captureArchive(options{dryRun: true, encrypt: true, opVault: defaultOPVault, opItem: defaultOPItem}); err != nil {
		t.Fatal(err)
	}

	got := stdout.String()

	for _, want := range []string{
		"would read 1Password item: Private/Mac Migration Archive",
		"would encrypt archive with Age recipient from 1Password",
		"would update 1Password latest_archive metadata",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry-run output missing %q\n%s", want, got)
		}
	}
}

func TestShellQuote(t *testing.T) {
	got := shellQuote([]string{"defaults", "write", "com.apple.finder", "FXPreferredViewStyle", "-string", "Nlsv"})
	want := "defaults write com.apple.finder FXPreferredViewStyle -string Nlsv"

	if got != want {
		t.Fatalf("shellQuote = %q, want %q", got, want)
	}

	got = shellQuote([]string{"brew", "bundle", "--file", "/Users/gus/Sites/mac os/Brewfile"})
	want = "brew bundle --file '/Users/gus/Sites/mac os/Brewfile'"

	if got != want {
		t.Fatalf("shellQuote with space = %q, want %q", got, want)
	}
}

func TestSanitizeDotfileRedactsMachineSpecificSettings(t *testing.T) {
	input := []byte("[coderabbit]\n\tmachineId = cli/example\n[core]\n\teditor = vim\n")
	got := string(sanitizeDotfile("/Users/gus/.gitconfig", "/Users/gus", input))

	if strings.Contains(got, "cli/example") {
		t.Fatalf("sanitizeDotfile leaked machine id: %s", got)
	}

	if !strings.Contains(got, "editor = vim") {
		t.Fatalf("sanitizeDotfile removed safe config: %s", got)
	}
}

func TestSanitizeDotfileRewritesHomePath(t *testing.T) {
	input := []byte(`export PATH="/Users/gus/bin:$PATH"`)
	got := string(sanitizeDotfile("/Users/gus/.zshrc", "/Users/gus", input))

	if strings.Contains(got, "/Users/gus") {
		t.Fatalf("sanitizeDotfile leaked absolute home path: %s", got)
	}

	if !strings.Contains(got, "$HOME/bin") {
		t.Fatalf("sanitizeDotfile did not rewrite home path: %s", got)
	}
}

func TestFindRepoRootWalksUp(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "Brewfile"), []byte("tap \"homebrew/bundle\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Mkdir(filepath.Join(dir, "stow"), 0o700); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(dir, "cmd", "mac-os")

	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}

	if got := findRepoRoot(nested); got != dir {
		t.Fatalf("findRepoRoot(%q) = %q, want %q", nested, got, dir)
	}
}
