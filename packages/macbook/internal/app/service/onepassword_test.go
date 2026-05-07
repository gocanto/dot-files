package service

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gocanto/dot-files/internal/command"
)

type onePasswordSigninRunner struct {
	calls          *[]string
	vaultListRuns  int
	vaultListError error
}

func TestOnePasswordSessionUsesActiveSession(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	session := onePasswordSession{
		stdout:   &stdout,
		runner:   stubRunner{outputs: outputsByQuotedKey(map[string][]byte{"op vault list --format=json": []byte(`[]`)}), calls: &calls},
		lookPath: fakeLookPath(nil),
	}

	if err := session.Ensure(); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 1 || calls[0] != "op vault list --format=json" {
		t.Fatalf("calls = %#v, want vault list access check only", calls)
	}

	if !strings.Contains(stdout.String(), "1Password CLI access is active") {
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
			calls:          &calls,
			vaultListError: expired,
		},
		lookPath: fakeLookPath(nil),
	}

	if err := session.Ensure(); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"op vault list --format=json", "op account list", "op signin"} {
		if !appTestContainsCall(calls, want) {
			t.Fatalf("calls missing %q: %#v", want, calls)
		}
	}

	if !strings.Contains(stdout.String(), "op signin") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestListVaultsParsesJSON(t *testing.T) {
	calls := []string{}
	outputs := map[string][]byte{
		"op vault list --format=json": []byte(`[{"id":"v1","name":"Private"},{"id":"v2","name":"Shared"}]`),
	}

	session := onePasswordSession{
		stdout:   &bytes.Buffer{},
		runner:   stubRunner{outputs: outputsByQuotedKey(outputs), calls: &calls},
		lookPath: fakeLookPath(nil),
	}

	vaults, err := session.ListVaults()

	if err != nil {
		t.Fatal(err)
	}

	if len(vaults) != 2 || vaults[0].Name != "Private" || vaults[1].Name != "Shared" {
		t.Fatalf("vaults = %#v", vaults)
	}
}

func TestListVaultsReturnsUnavailableWhenCLIMissing(t *testing.T) {
	session := onePasswordSession{
		stdout:   &bytes.Buffer{},
		runner:   stubRunner{},
		lookPath: fakeLookPath(errors.New("missing")),
	}

	_, err := session.ListVaults()

	var unavailable ErrOpUnavailable

	if !errors.As(err, &unavailable) {
		t.Fatalf("expected ErrOpUnavailable, got %v", err)
	}

	if !strings.Contains(unavailable.Reason, "1Password CLI (op) not found") {
		t.Fatalf("reason = %s", unavailable.Reason)
	}
}

func TestListVaultsReturnsUnavailableWhenSignedOut(t *testing.T) {
	errs := map[string]error{"op vault list --format=json": errors.New("not signed in")}

	session := onePasswordSession{
		stdout:   &bytes.Buffer{},
		runner:   stubRunner{outputs: outputsByQuotedKey(map[string][]byte{"op vault list --format=json": []byte("account is not signed in\n")}), errors: outputsByErrorKey(errs)},
		lookPath: fakeLookPath(nil),
	}

	_, err := session.ListVaults()

	var unavailable ErrOpUnavailable

	if !errors.As(err, &unavailable) {
		t.Fatalf("expected ErrOpUnavailable, got %v", err)
	}

	if !strings.Contains(unavailable.Reason, "not signed in") {
		t.Fatalf("reason = %s", unavailable.Reason)
	}
}

func TestListItemsRejectsEmptyVault(t *testing.T) {
	session := onePasswordSession{
		stdout:   &bytes.Buffer{},
		runner:   stubRunner{},
		lookPath: fakeLookPath(nil),
	}

	_, err := session.ListItems(" ")

	if err == nil || !strings.Contains(err.Error(), "vault name is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestListItemsParsesJSON(t *testing.T) {
	calls := []string{}
	outputs := map[string][]byte{
		"op item list --vault=Private --format=json": []byte(`[{"id":"i1","title":"GitHub"},{"id":"i2","title":"Mac Migration Archive"}]`),
	}

	session := onePasswordSession{
		stdout:   &bytes.Buffer{},
		runner:   stubRunner{outputs: outputsByQuotedKey(outputs), calls: &calls},
		lookPath: fakeLookPath(nil),
	}

	items, err := session.ListItems("Private")

	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 || items[0].Title != "GitHub" || items[1].Title != "Mac Migration Archive" {
		t.Fatalf("items = %#v", items)
	}
}

func outputsByQuotedKey(m map[string][]byte) map[string][]byte {
	quoted := make(map[string][]byte, len(m))

	for raw, value := range m {
		parts := strings.Split(raw, " ")
		quoted[command.ShellQuote(parts)] = value
	}

	return quoted
}

func outputsByErrorKey(m map[string]error) map[string]error {
	quoted := make(map[string]error, len(m))

	for raw, value := range m {
		parts := strings.Split(raw, " ")
		quoted[command.ShellQuote(parts)] = value
	}

	return quoted
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
	case "op vault list --format=json":
		r.vaultListRuns++

		if r.vaultListRuns == 1 {
			return []byte("account is not signed in\n"), r.vaultListError
		}

		return []byte(`[]`), nil
	case "op account list":
		return []byte("account\n"), nil
	case "op signin":
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected runner call: %s", call)
	}
}
