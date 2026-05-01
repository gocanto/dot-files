package app

import (
	"bytes"
	"errors"
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

func TestTUICommandLaunchesTUI(t *testing.T) {
	var launched bool
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, io.Discard, stubRunner{})
	a.tuiRunner = func(io.Reader, io.Writer, []tui.Workflow) (tui.Result, error) {
		launched = true

		return tui.Result{ExitCode: 7}, nil
	}

	if got := a.run([]string{"tui"}); got != 7 {
		t.Fatalf("exit = %d, want 7", got)
	}

	if !launched {
		t.Fatal("expected TUI launch")
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

func TestExistingCommandStillRequiresSudo(t *testing.T) {
	var stderr bytes.Buffer
	a := newApp("/Users/gus", "/repo", strings.NewReader(""), io.Discard, &stderr, stubRunner{
		outputs: map[string][]byte{"sudo -v": []byte("nope\n")},
		errors:  map[string]error{"sudo -v": errors.New("exit status 1")},
	})

	if got := a.run([]string{"doctor"}); got != 1 {
		t.Fatalf("exit = %d, want 1", got)
	}

	if !strings.Contains(stderr.String(), "sudo access required") {
		t.Fatalf("stderr = %s", stderr.String())
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
