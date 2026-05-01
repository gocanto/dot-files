package doctor

import (
	"bytes"
	"errors"
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

func TestEnsurePrerequisitesOnlyRequiresCommandLineTools(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	s := Service{
		GOOS:   "darwin",
		Stdout: &stdout,
		Runner: stubRunner{calls: &calls},
	}

	if err := s.EnsurePrerequisites(false); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.HasPrefix(call, "brew ") {
			t.Fatalf("EnsurePrerequisites called Homebrew: %v", calls)
		}
	}
}

func TestEnsurePrerequisitesReportsMissingCommandLineTools(t *testing.T) {
	s := Service{
		GOOS: "darwin",
		Runner: stubRunner{
			outputs: map[string][]byte{"xcode-select -p": []byte("unable to get active developer directory\n")},
			errors:  map[string]error{"xcode-select -p": errors.New("exit status 2")},
		},
	}

	err := s.EnsurePrerequisites(false)

	if err == nil {
		t.Fatal("expected missing CLT error")
	}

	if !strings.Contains(err.Error(), "xcode-select --install") {
		t.Fatalf("error = %v, want setup guidance", err)
	}
}

func TestEnsurePrerequisitesRejectsNonDarwin(t *testing.T) {
	err := Service{GOOS: "linux"}.EnsurePrerequisites(false)

	if err == nil {
		t.Fatal("expected unsupported OS error")
	}

	if !strings.Contains(err.Error(), "only supports darwin") {
		t.Fatalf("error = %v, want darwin guidance", err)
	}
}
