package command

import (
	"bytes"
	"errors"
	"testing"
)

type capturedRunner struct {
	out []byte
	err error
}

type interactiveTestRunner struct {
	called bool
	err    error
}

func (r capturedRunner) Run(string, ...string) ([]byte, error) {
	return r.out, r.err
}

func (r *interactiveTestRunner) Run(string, ...string) ([]byte, error) {
	return []byte("captured"), nil
}

func (r *interactiveTestRunner) RunInteractive(string, ...string) error {
	r.called = true

	return r.err
}

func TestShellQuote(t *testing.T) {
	got := ShellQuote([]string{"defaults", "write", "com.apple.finder", "FXPreferredViewStyle", "-string", "Nlsv"})
	want := "defaults write com.apple.finder FXPreferredViewStyle -string Nlsv"

	if got != want {
		t.Fatalf("ShellQuote = %q, want %q", got, want)
	}

	got = ShellQuote([]string{"brew", "bundle", "--file", "/Users/gus/Sites/mac os/Brewfile"})
	want = "brew bundle --file '/Users/gus/Sites/mac os/Brewfile'"

	if got != want {
		t.Fatalf("ShellQuote with space = %q, want %q", got, want)
	}
}

func TestRunInteractiveUsesInteractiveRunner(t *testing.T) {
	runner := &interactiveTestRunner{}

	if err := RunInteractive(runner, &bytes.Buffer{}, "sudo", "-v"); err != nil {
		t.Fatal(err)
	}

	if !runner.called {
		t.Fatal("expected interactive runner to be called")
	}
}

func TestRunInteractiveFallsBackToCapturedRunner(t *testing.T) {
	var stdout bytes.Buffer
	runner := capturedRunner{out: []byte("hello\n")}

	if err := RunInteractive(runner, &stdout, "echo", "hello"); err != nil {
		t.Fatal(err)
	}

	if stdout.String() != "hello\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunInteractiveReturnsFallbackError(t *testing.T) {
	want := errors.New("boom")
	runner := capturedRunner{err: want}

	if got := RunInteractive(runner, &bytes.Buffer{}, "false"); !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}
