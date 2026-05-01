package app

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/tui"
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

func TestNoArgsLaunchesTUI(t *testing.T) {
	var launched bool
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	a.tuiRunner = func(io.Reader, io.Writer, []tui.Workflow) (tui.Result, error) {
		launched = true

		return tui.Result{ExitCode: 0}, nil
	}

	if got := a.run(nil); got != 0 {
		t.Fatalf("exit = %d, want 0", got)
	}

	if !launched {
		t.Fatal("expected TUI launch")
	}
}

func TestTUIWorkflowsStartWithFactoryInstall(t *testing.T) {
	var workflows []tui.Workflow
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	a.tuiRunner = func(_ io.Reader, _ io.Writer, got []tui.Workflow) (tui.Result, error) {
		workflows = got

		return tui.Result{ExitCode: 0}, nil
	}

	if got := a.run(nil); got != 0 {
		t.Fatalf("exit = %d, want 0", got)
	}

	if len(workflows) == 0 || workflows[0].Name != "Factory Install" {
		t.Fatalf("first workflow = %#v, want Factory Install", workflows)
	}

	if len(workflows[0].Phases) < 8 {
		t.Fatalf("factory phases = %#v", workflows[0].Phases)
	}

	if workflows[0].Phases[0].Name != "prerequisites" || workflows[0].Phases[1].Name != "homebrew bundle" {
		t.Fatalf("factory phases = %#v", workflows[0].Phases)
	}

	if workflows[0].Confirmation == nil || len(workflows[0].Confirmation.Options) != 3 {
		t.Fatalf("factory confirmation = %#v", workflows[0].Confirmation)
	}
}

func TestFactoryInstallEraseConfirmationOptions(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{calls: &calls})
	a.goos = "linux"
	confirmation := a.factoryInstallConfirmation(false)

	if confirmation == nil || len(confirmation.Options) != 3 {
		t.Fatalf("confirmation = %#v", confirmation)
	}

	if !confirmation.Options[1].Continue || !confirmation.Options[2].Continue {
		t.Fatalf("expected already-erased and proceed-without-erase options to continue: %#v", confirmation.Options)
	}

	if confirmation.Options[0].Continue {
		t.Fatal("expected erase-first option to stop before install phases")
	}

	if err := confirmation.Options[0].Run(&stdout); err != nil {
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

	if err := a.factoryInstallConfirmation(false).Options[0].Run(io.Discard); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 1 || !strings.Contains(calls[0], "x-apple.systempreferences") {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestFactoryInstallDryRunDoesNotOpenResetSettings(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{calls: &calls})
	a.goos = "darwin"

	workflows := a.tuiWorkflows()

	if len(workflows) < 2 || workflows[1].Name != "Factory Install Dry Run" {
		t.Fatalf("workflows = %#v", workflows)
	}

	if err := workflows[1].Confirmation.Options[0].Run(&stdout); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 0 {
		t.Fatalf("dry run should not open settings, calls = %#v", calls)
	}

	if !strings.Contains(stdout.String(), "would open reset settings") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestCommandsAreRejected(t *testing.T) {
	for _, command := range []string{"tui", "doctor", "bootstrap"} {
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

func TestHelpOnlyShowsTUIUsage(t *testing.T) {
	var stdout bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), &stdout, io.Discard, stubRunner{})

	if got := a.run([]string{"help"}); got != 0 {
		t.Fatalf("exit = %d, want 0", got)
	}

	output := stdout.String()

	for _, want := range []string{"mac-os", "interactive Bubble Tea workflow dashboard"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output = %s, want %q", output, want)
		}
	}

	for _, old := range []string{"mac-os tui", "bootstrap", "adopt", "capture", "restore", "secrets", "doctor", "brewfile", "macos"} {
		if strings.Contains(output, old) {
			t.Fatalf("help output = %s, did not expect old command %q", output, old)
		}
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
