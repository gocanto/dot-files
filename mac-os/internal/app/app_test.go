package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stubRunner struct {
	outputs map[string][]byte
	errors  map[string]error
	calls   *[]string
}

func (r stubRunner) Run(name string, args ...string) ([]byte, error) {
	key := shellQuote(append([]string{name}, args...))

	if r.calls != nil {
		*r.calls = append(*r.calls, key)
	}

	return r.outputs[key], r.errors[key]
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

func TestShouldSkipSensitive(t *testing.T) {
	cases := map[string]bool{
		"/Users/gus/.ssh/id_ed25519":                                                 true,
		"/Users/gus/.ssh/id_ed25519.pub":                                             false,
		"/Users/gus/.zsh_history":                                                    true,
		"/Users/gus/.config/gh/hosts.yml":                                            true,
		"/Users/gus/.config/ghostty/config":                                          false,
		"/Users/gus/Library/Application Support/Code/User/settings.json":             false,
		"/Users/gus/Library/Application Support/Code/User/globalStorage/state.vscdb": true,
		"/Users/gus/Library/Application Support/Google/Chrome/Default/Cookies":       true,
		"/Users/gus/Library/Keychains/login.keychain-db":                             true,
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
		`brew "mas"`,
		`brew "opencode"`,
		`brew "node@24"`,
		`brew "go"`,
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

func TestRequireSudoValidatesWithSudoV(t *testing.T) {
	var calls []string
	a := app{runner: stubRunner{calls: &calls}}

	if err := a.requireSudo(); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 1 || calls[0] != "sudo -v" {
		t.Fatalf("calls = %v, want sudo -v", calls)
	}
}

func TestRequireSudoReportsAuthFailure(t *testing.T) {
	a := app{
		runner: stubRunner{
			outputs: map[string][]byte{"sudo -v": []byte("not in sudoers\n")},
			errors:  map[string]error{"sudo -v": errors.New("exit status 1")},
		},
	}

	err := a.requireSudo()

	if err == nil {
		t.Fatal("expected sudo failure")
	}

	if !strings.Contains(err.Error(), "sudo -v") || !strings.Contains(err.Error(), "not in sudoers") {
		t.Fatalf("error = %v, want sudo command and output", err)
	}
}

func TestEnsurePrerequisitesOnlyRequiresCommandLineTools(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	a := app{
		goos:   "darwin",
		stdout: &stdout,
		runner: stubRunner{calls: &calls},
	}

	if err := a.ensurePrerequisites(options{}); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.HasPrefix(call, "brew ") {
			t.Fatalf("ensurePrerequisites called Homebrew: %v", calls)
		}
	}
}

func TestEnsurePrerequisitesReportsMissingCommandLineTools(t *testing.T) {
	a := app{
		goos: "darwin",
		runner: stubRunner{
			outputs: map[string][]byte{"xcode-select -p": []byte("unable to get active developer directory\n")},
			errors:  map[string]error{"xcode-select -p": errors.New("exit status 2")},
		},
	}

	err := a.ensurePrerequisites(options{})

	if err == nil {
		t.Fatal("expected missing CLT error")
	}

	if !strings.Contains(err.Error(), "xcode-select --install") {
		t.Fatalf("error = %v, want setup guidance", err)
	}
}

func TestEnsurePrerequisitesRejectsNonDarwin(t *testing.T) {
	a := app{goos: "linux"}

	err := a.ensurePrerequisites(options{})

	if err == nil {
		t.Fatal("expected unsupported OS error")
	}

	if !strings.Contains(err.Error(), "only supports darwin") {
		t.Fatalf("error = %v, want darwin guidance", err)
	}
}

func TestLoadAppConfigValidatesModesAndPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apps.yaml")
	content := []byte(`
apps:
  - name: Ghostty
    bundle_id: com.mitchellh.ghostty
    install_method: brew
    package: ghostty
    config_mode: auto
    config_paths:
      - source: ~/.config/ghostty/config
        target: apps/ghostty/config
`)

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	a := app{home: "/Users/gus", repo: dir}
	cfg, err := a.loadAppConfig("")

	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Apps) != 1 {
		t.Fatalf("loaded %d apps, want 1", len(cfg.Apps))
	}

	if got := cfg.Apps[0].Package; got != "ghostty" {
		t.Fatalf("package = %q, want ghostty", got)
	}
}

func TestLoadAppConfigRejectsInvalidMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "apps.yaml")
	content := []byte(`
apps:
  - name: Broken
    install_method: curl
    config_mode: auto
    config_paths:
      - source: ~/.config/broken
        target: apps/broken
`)

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	a := app{home: "/Users/gus", repo: dir}
	_, err := a.loadAppConfig("")

	if err == nil {
		t.Fatal("expected invalid install mode error")
	}

	if !strings.Contains(err.Error(), "install_method") {
		t.Fatalf("error = %v, want install_method validation", err)
	}
}

func TestAppCapturePlanSkipsManualConfig(t *testing.T) {
	cfg := appConfig{Apps: []managedApp{
		{
			Name:          "Ghostty",
			InstallMethod: "brew",
			Package:       "ghostty",
			ConfigMode:    "auto",
			ConfigPaths: []appConfigPath{
				{Source: "~/.config/ghostty/config", Target: "apps/ghostty/config"},
			},
		},
		{
			Name:          "Slack",
			InstallMethod: "mas",
			Package:       "803453959",
			ConfigMode:    "manual",
			ConfigPaths: []appConfigPath{
				{Source: "~/Library/Application Support/Slack", Target: "apps/slack"},
			},
		},
	}}

	got := appCapturePlan(cfg)

	if len(got) != 1 {
		t.Fatalf("appCapturePlan returned %d items, want 1", len(got))
	}

	if got[0].target != "apps/ghostty/config" {
		t.Fatalf("target = %q", got[0].target)
	}
}

func TestLoadSecretConfigValidatesViperManifest(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)

	a := app{repo: dir}
	cfg, err := a.loadSecretConfig("")

	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Secrets) != 1 {
		t.Fatalf("loaded %d secrets, want 1", len(cfg.Secrets))
	}

	if got := cfg.Secrets[0].Name; got != gitconfigSecret {
		t.Fatalf("secret name = %q, want %q", got, gitconfigSecret)
	}
}

func TestLoadSecretConfigRejectsDuplicateNames(t *testing.T) {
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

	a := app{repo: dir}
	_, err := a.loadSecretConfig("")

	if err == nil {
		t.Fatal("expected duplicate secret name error")
	}

	if !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("error = %v, want duplicated", err)
	}
}

func TestLoadSecretConfigRejectsUnsafePaths(t *testing.T) {
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

	a := app{repo: dir}
	_, err := a.loadSecretConfig("")

	if err == nil {
		t.Fatal("expected unsafe path error")
	}

	if !strings.Contains(err.Error(), "repo-relative") {
		t.Fatalf("error = %v, want repo-relative", err)
	}
}

func TestLoadSecretConfigRejectsMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
secrets:
  - name: gitconfig
    plaintext_path: stow/git/.config/git/private.gitconfig
    encrypted_path: stow/git/.config/git/private.gitconfig.age
    mode: age-file
`)

	if err := os.WriteFile(filepath.Join(dir, "secrets.yaml"), content, 0o600); err != nil {
		t.Fatal(err)
	}

	a := app{repo: dir}
	_, err := a.loadSecretConfig("")

	if err == nil {
		t.Fatal("expected missing op_field error")
	}

	if !strings.Contains(err.Error(), "op_field") {
		t.Fatalf("error = %v, want op_field", err)
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

func TestEncryptGitconfigSecretRequiresPlaintextField(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)
	a := app{
		repo: dir,
		runner: stubRunner{outputs: map[string][]byte{
			`op item get 'Mac Migration Archive' --vault Private --format json`: []byte(`{
				"fields": [
					{"id": "archive_age_recipient", "label": "archive_age_recipient", "value": "age1example"}
				]
			}`),
		}},
	}

	err := a.encryptGitconfigSecret(options{opVault: defaultOPVault, opItem: defaultOPItem})

	if err == nil {
		t.Fatal("expected missing gitconfig_plaintext error")
	}

	if !strings.Contains(err.Error(), gitconfigPlaintext) {
		t.Fatalf("error = %v, want %s", err, gitconfigPlaintext)
	}
}

func TestEncryptGitconfigSecretDoesNotPrintPlaintext(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)
	secret := "[user]\n\temail = private@example.com\n"

	var stdout bytes.Buffer

	var calls []string
	a := app{
		repo:   dir,
		stdout: &stdout,
		runner: stubRunner{
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

	if err := a.encryptGitconfigSecret(options{opVault: defaultOPVault, opItem: defaultOPItem}); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(stdout.String(), "private@example.com") {
		t.Fatalf("stdout leaked gitconfig plaintext: %s", stdout.String())
	}

	data, err := os.ReadFile(a.privateGitconfigPath())

	if err != nil {
		t.Fatal(err)
	}

	if got := string(data); got != secret {
		t.Fatalf("private gitconfig = %q, want %q", got, secret)
	}

	ageCall := shellQuote([]string{"age", "-r", "age1example", "-o", a.encryptedGitconfigPath(), a.privateGitconfigPath()})

	if !containsCall(calls, ageCall) {
		t.Fatalf("calls = %v, want %s", calls, ageCall)
	}
}

func TestSecretsEncryptTargetUsesManifest(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	var calls []string
	a := app{
		repo:   dir,
		stdout: &stdout,
		stderr: &stderr,
		runner: stubRunner{
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

	if got := a.secrets([]string{"encrypt", "--target", "gitconfig"}); got != 0 {
		t.Fatalf("secrets encrypt exit = %d, stderr = %s", got, stderr.String())
	}

	if strings.Contains(stdout.String(), "private@example.com") {
		t.Fatalf("stdout leaked gitconfig plaintext: %s", stdout.String())
	}

	ageCall := shellQuote([]string{"age", "-r", "age1example", "-o", a.encryptedGitconfigPath(), a.privateGitconfigPath()})

	if !containsCall(calls, ageCall) {
		t.Fatalf("calls = %v, want %s", calls, ageCall)
	}
}

func TestDecryptGitconfigFallsBackToPlaintextWhenEncryptedFileMissing(t *testing.T) {
	dir := t.TempDir()
	writeTestSecretConfig(t, dir)

	var calls []string

	var stdout bytes.Buffer
	a := app{
		repo:   dir,
		stdout: &stdout,
		runner: stubRunner{
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

	if err := a.decryptGitconfigSecret(options{opVault: defaultOPVault, opItem: defaultOPItem}); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.HasPrefix(call, "age ") {
			t.Fatalf("decrypt fallback called age unexpectedly: %v", calls)
		}
	}

	data, err := os.ReadFile(a.privateGitconfigPath())

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
	a := app{
		repo:   dir,
		stdout: &stdout,
		runner: stubRunner{
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

	if err := writeFile(a.privateGitconfigPath(), []byte("[user]\n\temail = private@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := a.syncGitconfigSecret(options{opVault: defaultOPVault, opItem: defaultOPItem}); err != nil {
		t.Fatal(err)
	}

	opPrefix := "op item edit 'Mac Migration Archive' --vault Private "
	ageCall := shellQuote([]string{"age", "-r", "age1example", "-o", a.encryptedGitconfigPath(), a.privateGitconfigPath()})

	if !containsCallPrefix(calls, opPrefix) {
		t.Fatalf("calls = %v, want prefix %s", calls, opPrefix)
	}

	if !containsCall(calls, ageCall) {
		t.Fatalf("calls = %v, want %s", calls, ageCall)
	}
}

func TestPrivateGitconfigIsIgnoredByGit(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", ".gitignore"))

	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "mac-os/stow/git/.config/git/private.gitconfig") {
		t.Fatal(".gitignore does not ignore decrypted private gitconfig")
	}
}

func TestCaptureDryRunShowsAppConfigPlan(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "apps.yaml")
	content := []byte(`
apps:
  - name: Ghostty
    install_method: brew
    package: ghostty
    config_mode: auto
    config_paths:
      - source: ~/.config/ghostty/config
        target: apps/ghostty/config
`)

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	a := app{
		home:   "/Users/gus",
		repo:   dir,
		stdout: &stdout,
		runner: stubRunner{},
	}

	if err := a.captureArchive(options{dryRun: true, apps: true}); err != nil {
		t.Fatal(err)
	}

	got := stdout.String()

	if !strings.Contains(got, "would capture app config: ~/.config/ghostty/config -> apps/ghostty/config") {
		t.Fatalf("dry-run output missing app config plan\n%s", got)
	}
}

func TestRestoreDryRunShowsAppConfigPlan(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "apps.yaml")
	archive := filepath.Join(dir, "archive")
	source := filepath.Join(archive, "apps/ghostty/config")
	content := []byte(`
apps:
  - name: Ghostty
    install_method: brew
    package: ghostty
    config_mode: auto
    config_paths:
      - source: ~/.config/ghostty/config
        target: apps/ghostty/config
`)

	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(source, []byte("font-size = 16\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	a := app{
		home:   "/Users/gus",
		repo:   dir,
		stdout: &stdout,
		runner: stubRunner{},
	}

	if err := a.restoreAppConfigs(options{dryRun: true, apps: true, archivePath: archive}); err != nil {
		t.Fatal(err)
	}

	got := stdout.String()

	if !strings.Contains(got, "would restore app config:") {
		t.Fatalf("dry-run output missing restore plan\n%s", got)
	}

	if !strings.Contains(got, "/Users/gus/.config/ghostty/config") {
		t.Fatalf("dry-run output missing expanded target\n%s", got)
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

func TestFindRepoRootFromOuterRepoUsesMacOSDir(t *testing.T) {
	dir := t.TempDir()
	macOSDir := filepath.Join(dir, "mac-os")

	if err := os.MkdirAll(filepath.Join(macOSDir, "stow"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(macOSDir, "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(macOSDir, "Brewfile"), []byte("tap \"homebrew/bundle\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := findRepoRoot(dir); got != macOSDir {
		t.Fatalf("findRepoRoot(%q) = %q, want %q", dir, got, macOSDir)
	}
}
