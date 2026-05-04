package app

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type onePasswordSigninRunner struct {
	calls       *[]string
	whoamiRuns  int
	whoamiError error
}

func TestOnePasswordSessionUsesActiveSession(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	session := onePasswordSession{
		stdout:   &stdout,
		runner:   stubRunner{calls: &calls},
		lookPath: fakeLookPath(nil),
	}

	if err := session.Ensure(); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 1 || calls[0] != "op whoami" {
		t.Fatalf("calls = %#v, want active session check only", calls)
	}

	if !strings.Contains(stdout.String(), "1Password CLI session is active") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestOnePasswordSessionRejectsMissingCLI(t *testing.T) {
	session := onePasswordSession{
		stdout:   &bytes.Buffer{},
		runner:   stubRunner{},
		lookPath: fakeLookPath(errors.New("missing")),
	}

	err := session.Ensure()

	if err == nil {
		t.Fatal("expected missing CLI error")
	}

	if !strings.Contains(err.Error(), "1Password CLI (op) not found") {
		t.Fatalf("error = %v", err)
	}
}

func TestOnePasswordSessionSignsInWhenAccountExists(t *testing.T) {
	var calls []string
	expired := errors.New("not signed in")

	var stdout bytes.Buffer
	session := onePasswordSession{
		stdout: &stdout,
		runner: &onePasswordSigninRunner{
			calls:       &calls,
			whoamiError: expired,
		},
		lookPath: fakeLookPath(nil),
	}

	if err := session.Ensure(); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"op whoami", "op account list", "op signin", "op whoami"} {
		if !appTestContainsCall(calls, want) {
			t.Fatalf("calls missing %q: %#v", want, calls)
		}
	}

	if !strings.Contains(stdout.String(), "op signin") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func appTestContainsCall(calls []string, want string) bool {
	for _, call := range calls {
		if strings.Contains(call, want) {
			return true
		}
	}

	return false
}

func fakeLookPath(err error) func(string) (string, error) {
	return func(name string) (string, error) {
		if err != nil {
			return "", err
		}

		return "/usr/local/bin/" + name, nil
	}
}

func (r *onePasswordSigninRunner) Run(name string, args ...string) ([]byte, error) {
	call := name

	for _, arg := range args {
		call += " " + arg
	}

	if r.calls != nil {
		*r.calls = append(*r.calls, call)
	}

	switch call {
	case "op whoami":
		r.whoamiRuns++

		if r.whoamiRuns == 1 {
			return nil, r.whoamiError
		}

		return []byte("signed-in\n"), nil
	case "op account list":
		return []byte("account\n"), nil
	default:
		return nil, nil
	}
}
