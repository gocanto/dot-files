package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

type stubRunner struct {
	outputs map[string][]byte
	errors  map[string]error
	calls   *[]string
}

type errRunner struct {
	err error
}

func (r stubRunner) Run(name string, args ...string) ([]byte, error) {
	key := command.ShellQuote(append([]string{name}, args...))

	if r.calls != nil {
		*r.calls = append(*r.calls, key)
	}

	return r.outputs[key], r.errors[key]
}

func TestNoArgsShowsUsage(t *testing.T) {
	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	a.stdout = &stdout

	if got := a.run(nil); got != 0 {
		t.Fatalf("exit = %d, want 0", got)
	}

	if !strings.Contains(stdout.String(), "mac-os serve-http --socket") {
		t.Fatalf("usage = %s", stdout.String())
	}
}

func TestWorkflowsUsePlainMenuLabels(t *testing.T) {
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	workflows := a.workflows()

	wantNames := []string{
		"Set Up This Mac",
		"Update This Mac",
		"Save App Settings Snapshot",
		"Restore App Settings",
		"Update Installed App List",
		"Apply macOS Settings",
		"Check Setup",
		"Show Homebrew Packages",
	}

	if len(workflows) != len(wantNames) {
		t.Fatalf("workflow count = %d, want %d: %#v", len(workflows), len(wantNames), workflows)
	}

	for i, want := range wantNames {
		if workflows[i].Name != want {
			t.Fatalf("workflow[%d] = %q, want %q", i, workflows[i].Name, want)
		}
	}

	for _, workflow := range workflows {
		if strings.Contains(workflow.Name, "Dry Run") || workflow.Name == "Bootstrap" {
			t.Fatalf("unexpected technical workflow name: %#v", workflow)
		}

		if workflow.Description == "" || workflow.ChangesMac == "" || workflow.Confirmation == nil {
			t.Fatalf("workflow missing explanation metadata: %#v", workflow)
		}
	}
}

func TestUpdateThisMacWorkflowUsesFullHostUpdatePhases(t *testing.T) {
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	workflows := a.workflows()

	var workflow *workflowdomain.Workflow

	for i := range workflows {
		if workflows[i].Name == "Update This Mac" {
			workflow = &workflows[i]

			break
		}
	}

	if workflow == nil {
		t.Fatalf("missing Update This Mac workflow: %#v", workflows)
	}

	if workflow.Confirmation == nil || len(workflow.Confirmation.Options) != 3 {
		t.Fatalf("workflow confirmation = %#v", workflow.Confirmation)
	}

	want := []string{
		"Check/install prerequisites",
		"Install Homebrew packages",
		"Set up GitHub access and signing",
		"Install App Store apps",
		"Show manual app install notes",
		"Restore private secrets from 1Password",
		"Install oh-my-zsh",
		"Link dotfiles",
		"Restore supported app configs from latest snapshot",
		"Apply macOS settings",
		"Run health checks",
	}

	for _, optionIndex := range []int{0, 1} {
		phases := workflow.Confirmation.Options[optionIndex].Phases

		if len(phases) != len(want) {
			t.Fatalf("option %d phase count = %d, want %d: %#v", optionIndex, len(phases), len(want), phases)
		}

		for i, wantName := range want {
			if phases[i].Name != wantName {
				t.Fatalf("option %d phase[%d] = %q, want %q", optionIndex, i, phases[i].Name, wantName)
			}
		}
	}

	for _, phase := range workflow.Confirmation.Options[1].Phases {
		if phase.Name == "Prepare existing dotfiles" {
			t.Fatalf("Update This Mac must not import host dotfiles into the repo: %#v", workflow.Confirmation.Options[1].Phases)
		}
	}
}

func TestSetUpThisMacConfirmationOptions(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{calls: &calls})
	a.goos = "linux"
	confirmation := a.setupConfirmation(options{dryRun: true, apps: true}, options{apps: true})

	if confirmation == nil || len(confirmation.Options) != 5 {
		t.Fatalf("confirmation = %#v", confirmation)
	}

	if !confirmation.Options[0].Continue || !confirmation.Options[2].Continue || !confirmation.Options[3].Continue {
		t.Fatalf("expected preview and run options to continue: %#v", confirmation.Options)
	}

	if confirmation.Options[1].Continue {
		t.Fatal("expected erase-first option to stop before install phases")
	}

	if !confirmation.Options[4].Back {
		t.Fatalf("expected final option to go back: %#v", confirmation.Options)
	}

	if err := confirmation.Options[1].Run(&stdout); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stdout.String(), "Erase Assistant") {
		t.Fatalf("stdout = %s", stdout.String())
	}

	if len(calls) != 0 {
		t.Fatalf("non-darwin should not open settings, calls = %#v", calls)
	}
}

func TestFactoryInstallEraseFirstOpensResetSettingsOnDarwin(t *testing.T) {
	var calls []string
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{calls: &calls})
	a.goos = "darwin"

	if err := a.setupConfirmation(options{dryRun: true, apps: true}, options{apps: true}).Options[1].Run(io.Discard); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 2 {
		t.Fatalf("calls = %#v", calls)
	}

	if calls[0] != "sudo -v" || !strings.Contains(calls[1], "x-apple.systempreferences") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestSetUpThisMacPreviewDoesNotOpenResetSettings(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{calls: &calls})
	a.goos = "darwin"

	workflows := a.workflows()

	if len(workflows) == 0 || workflows[0].Name != "Set Up This Mac" {
		t.Fatalf("workflows = %#v", workflows)
	}

	if err := workflows[0].Confirmation.Options[0].Run(&stdout); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 0 {
		t.Fatalf("preview should not open settings, calls = %#v", calls)
	}

	for _, want := range []string{"preview selected"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %s", stdout.String())
		}
	}

	if workflows[0].Confirmation.Options[0].Phases[0].Name != "Check/install prerequisites" {
		t.Fatalf("preview phases = %#v", workflows[0].Confirmation.Options[0].Phases)
	}

	if workflows[0].Confirmation.Options[0].Phases[2].Name != "Set up GitHub access and signing" {
		t.Fatalf("preview phases = %#v", workflows[0].Confirmation.Options[0].Phases)
	}
}

func TestFactoryInstallEraseFirstStopsWhenAdminValidationFails(t *testing.T) {
	var calls []string
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{
		calls:  &calls,
		errors: map[string]error{"sudo -v": os.ErrPermission},
	})
	a.goos = "darwin"

	err := a.setupConfirmation(options{dryRun: true, apps: true}, options{apps: true}).Options[1].Run(io.Discard)

	if err == nil {
		t.Fatal("expected admin validation error")
	}

	if len(calls) != 1 || calls[0] != "sudo -v" {
		t.Fatalf("calls = %#v", calls)
	}

	if !strings.Contains(err.Error(), "validate administrator access") {
		t.Fatalf("error = %v", err)
	}
}

func TestCommandsAreRejected(t *testing.T) {
	for _, command := range []string{"doctor", "bootstrap", "brewfile"} {
		t.Run(command, func(t *testing.T) {
			var stderr bytes.Buffer
			a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, &stderr, stubRunner{})

			if got := a.run([]string{command}); got != 2 {
				t.Fatalf("exit = %d, want 2", got)
			}

			if !strings.Contains(stderr.String(), `unknown command "`+command+`"`) {
				t.Fatalf("stderr = %s", stderr.String())
			}
		})
	}
}

func TestHelpOnlyShowsElectronCommandUsage(t *testing.T) {
	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{})

	if got := a.run([]string{"help"}); got != 0 {
		t.Fatalf("exit = %d, want 0", got)
	}

	output := stdout.String()

	for _, want := range []string{"mac-os", "mac-os serve-http --socket", "HTTP backend"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output = %s, want %q", output, want)
		}
	}

	for _, old := range []string{"interactive legacy interface", "bootstrap", "adopt", "capture", "restore", "secrets", "doctor", "brewfile", "macos"} {
		if strings.Contains(output, old) {
			t.Fatalf("help output = %s, did not expect old command %q", output, old)
		}
	}
}

func TestUpdateInstalledAppListWorkflowUsesPreviewCandidate(t *testing.T) {
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	workflows := a.workflows()

	var workflow *workflowdomain.Workflow

	for i := range workflows {
		if workflows[i].Name == "Update Installed App List" {
			workflow = &workflows[i]

			break
		}
	}

	if workflow == nil {
		t.Fatalf("missing Update Installed App List workflow: %#v", workflows)
	}

	if workflow.Confirmation == nil || len(workflow.Confirmation.Options) < 2 {
		t.Fatalf("workflow confirmation = %#v", workflow.Confirmation)
	}

	if workflow.Confirmation.Options[0].Phases[0].Name != "Generate installed app list candidate" {
		t.Fatalf("preview phases = %#v", workflow.Confirmation.Options[0].Phases)
	}

	if workflow.Confirmation.Options[1].Phases[0].Name != "Generate installed app list candidate" {
		t.Fatalf("run phases = %#v", workflow.Confirmation.Options[1].Phases)
	}
}

func TestRestoreAppConfigsUsesLatestLocalSnapshot(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")
	oldSnapshot := filepath.Join(home, ".local", "state", "macos-settings-archives", "20260102-030405")
	latestSnapshot := filepath.Join(home, ".local", "state", "macos-settings-archives", "20260103-030405")

	for _, dir := range []string{
		repo,
		filepath.Join(oldSnapshot, "ghostty"),
		filepath.Join(latestSnapshot, "ghostty"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	config := []byte(`
apps:
  - name: Ghostty
    install_method: brew
    package: ghostty
    config_mode: auto
    config_paths:
      - source: ~/.config/ghostty/config
        target: ghostty/config
`)

	if err := os.WriteFile(filepath.Join(repo, "apps.yaml"), config, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(oldSnapshot, "ghostty", "config"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(latestSnapshot, "ghostty", "config"), []byte("latest\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	a := newApp(home, repo, strings.NewReader(""), &stdout, io.Discard, stubRunner{})

	if err := a.restoreAppConfigs(options{dryRun: true, apps: true, useLatestArchive: true}); err != nil {
		t.Fatal(err)
	}

	got := stdout.String()

	for _, want := range []string{
		"using latest local app settings snapshot: " + latestSnapshot,
		"would restore app config: " + filepath.Join(latestSnapshot, "ghostty", "config"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout missing %q\n%s", want, got)
		}
	}
}

func TestRestoreAppConfigsSkipsWhenLatestLocalSnapshotIsMissing(t *testing.T) {
	var stdout bytes.Buffer
	a := newApp(t.TempDir(), "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{})

	if err := a.restoreAppConfigs(options{dryRun: true, apps: true, useLatestArchive: true}); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(stdout.String(), "skipped: no local app settings snapshot found under") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestFindRepoRootWalksUp(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o600); err != nil {
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

func TestApplyStowDetectsStaleSymlinksFromAnotherTree(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")
	otherRepo := filepath.Join(tmp, "other")

	for _, dir := range []string{
		filepath.Join(home, ".config"),
		filepath.Join(repo, "stow", "shell"),
		filepath.Join(repo, "stow", "git", ".config", "git"),
		filepath.Join(otherRepo, "stow", "shell"),
		filepath.Join(otherRepo, "stow", "git", ".config", "git"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range []string{
		filepath.Join(repo, "stow", "shell", ".zshrc"),
		filepath.Join(repo, "stow", "git", ".config", "git", "ignore"),
		filepath.Join(otherRepo, "stow", "shell", ".zshrc"),
		filepath.Join(otherRepo, "stow", "git", ".config", "git", "ignore"),
	} {
		if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Symlink(filepath.Join(otherRepo, "stow", "shell", ".zshrc"), filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(filepath.Join(otherRepo, "stow", "git", ".config", "git"), filepath.Join(home, ".config", "git")); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	a := newApp(home, repo, strings.NewReader(""), &stdout, io.Discard, stubRunner{})

	err := a.applyStow(options{})

	if err == nil {
		t.Fatal("expected stale-link error, got nil")
	}

	if !strings.Contains(err.Error(), "stale stow link") {
		t.Fatalf("error = %v", err)
	}

	out := stdout.String()

	for _, want := range []string{
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".config", "git"),
		filepath.Join(otherRepo, "stow", "shell", ".zshrc"),
		filepath.Join(otherRepo, "stow", "git", ".config", "git"),
		"unstowing from the old tree",
		filepath.Join(otherRepo),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q\n%s", want, out)
		}
	}
}

func TestApplyStowAllowsLinksIntoCurrentTree(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	repo := filepath.Join(tmp, "repo")

	for _, dir := range []string{
		home,
		filepath.Join(repo, "stow", "shell"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(filepath.Join(repo, "stow", "shell", ".zshrc"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(filepath.Join(repo, "stow", "shell", ".zshrc"), filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}

	var calls []string
	a := newApp(home, repo, strings.NewReader(""), io.Discard, io.Discard, stubRunner{calls: &calls})

	if err := a.applyStow(options{dryRun: true}); err != nil {
		t.Fatalf("applyStow returned error: %v", err)
	}

	if len(calls) != 0 {
		t.Fatalf("dry-run should not invoke runner, calls = %#v", calls)
	}
}

func TestEnsureOhMyZshSkipsWhenInstalled(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")

	if err := os.MkdirAll(filepath.Join(home, ".oh-my-zsh"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(home, ".oh-my-zsh", "oh-my-zsh.sh"), []byte("# oh-my-zsh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var calls []string

	var stdout bytes.Buffer
	a := newApp(home, "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{calls: &calls})

	if err := a.ensureOhMyZsh(options{}); err != nil {
		t.Fatalf("ensureOhMyZsh returned error: %v", err)
	}

	if len(calls) != 0 {
		t.Fatalf("expected no runner calls when oh-my-zsh is installed, got %#v", calls)
	}

	if !strings.Contains(stdout.String(), "oh-my-zsh found") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestEnsureOhMyZshDryRunPrintsCommand(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []string

	var stdout bytes.Buffer
	a := newApp(home, "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{calls: &calls})

	if err := a.ensureOhMyZsh(options{dryRun: true}); err != nil {
		t.Fatalf("ensureOhMyZsh returned error: %v", err)
	}

	if len(calls) != 0 {
		t.Fatalf("dry-run should not invoke runner, calls = %#v", calls)
	}

	for _, want := range []string{"would run:", "RUNZSH=no", "CHSH=no", "KEEP_ZSHRC=yes", "ohmyzsh/ohmyzsh"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout.String())
		}
	}
}

func TestEnsureOhMyZshInstallsWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls []string
	a := newApp(home, "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{calls: &calls})

	if err := a.ensureOhMyZsh(options{}); err != nil {
		t.Fatalf("ensureOhMyZsh returned error: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected one runner call, got %#v", calls)
	}

	if !strings.HasPrefix(calls[0], "sh -c ") {
		t.Fatalf("call = %q, want sh -c ...", calls[0])
	}

	for _, want := range []string{"RUNZSH=no", "CHSH=no", "KEEP_ZSHRC=yes", "ohmyzsh/ohmyzsh"} {
		if !strings.Contains(calls[0], want) {
			t.Fatalf("call missing %q: %s", want, calls[0])
		}
	}
}

func TestEnsureOhMyZshSurfacesInstallError(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	a := newApp(home, "/repo", strings.NewReader(""), io.Discard, io.Discard, errRunner{err: os.ErrPermission})

	err := a.ensureOhMyZsh(options{})

	if err == nil {
		t.Fatal("expected install error")
	}

	if !strings.Contains(err.Error(), "install oh-my-zsh") {
		t.Fatalf("error = %v", err)
	}
}

func (r errRunner) Run(name string, args ...string) ([]byte, error) {
	return nil, r.err
}

func TestFactoryInstallIncludesOhMyZshBeforeStow(t *testing.T) {
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	phases := a.factoryInstallPhases(options{dryRun: true, apps: true})

	var stowIdx, ohmyzshIdx = -1, -1

	for i, phase := range phases {
		if phase.Name == "Link dotfiles" {
			stowIdx = i
		}

		if phase.Name == "Install oh-my-zsh" {
			ohmyzshIdx = i
		}
	}

	if ohmyzshIdx < 0 {
		t.Fatalf("missing oh-my-zsh phase: %#v", phases)
	}

	if stowIdx < 0 {
		t.Fatalf("missing stow links phase: %#v", phases)
	}

	if ohmyzshIdx >= stowIdx {
		t.Fatalf("oh-my-zsh phase (%d) must run before stow links (%d)", ohmyzshIdx, stowIdx)
	}
}

func TestFindRepoRootFromOuterRepoUsesMacOSDir(t *testing.T) {
	dir := t.TempDir()
	macOSDir := filepath.Join(dir, "macbook")

	if err := os.MkdirAll(filepath.Join(macOSDir, "stow"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(macOSDir, "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := findRepoRoot(dir); got != macOSDir {
		t.Fatalf("findRepoRoot(%q) = %q, want %q", dir, got, macOSDir)
	}
}
