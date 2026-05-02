package doctor

import (
	"bytes"
	"encoding/json"
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

func TestEnsurePrerequisitesOnlyRequiresCommandLineTools(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	s := Service{
		GOOS:   "darwin",
		GOARCH: "amd64",
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

func TestEnsurePrerequisitesSkipsRosettaOnIntel(t *testing.T) {
	var calls []string

	s := Service{
		GOOS:   "darwin",
		GOARCH: "amd64",
		Stdout: new(bytes.Buffer),
		Runner: stubRunner{calls: &calls},
	}

	if err := s.EnsurePrerequisites(false); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.Contains(call, "softwareupdate") {
			t.Fatalf("intel should not invoke softwareupdate: %#v", calls)
		}
	}
}

func TestEnsurePrerequisitesSkipsRosettaWhenInstalled(t *testing.T) {
	var calls []string

	s := Service{
		GOOS:         "darwin",
		GOARCH:       "arm64",
		Home:         t.TempDir(),
		Stdout:       new(bytes.Buffer),
		Runner:       stubRunner{calls: &calls},
		rosettaCheck: func() bool { return true },
	}

	if err := s.EnsurePrerequisites(false); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.Contains(call, "softwareupdate") {
			t.Fatalf("installed Rosetta should not invoke softwareupdate: %#v", calls)
		}
	}
}

func TestEnsurePrerequisitesInstallsRosettaOnAppleSilicon(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	home := t.TempDir()
	s := Service{
		GOOS:         "darwin",
		GOARCH:       "arm64",
		Home:         home,
		Stdout:       &stdout,
		Runner:       stubRunner{calls: &calls},
		rosettaCheck: func() bool { return false },
	}

	if err := s.EnsurePrerequisites(false); err != nil {
		t.Fatal(err)
	}

	want := "softwareupdate --install-rosetta --agree-to-license"

	var found bool

	for _, call := range calls {
		if call == want {
			found = true

			break
		}
	}

	if !found {
		t.Fatalf("expected runner call %q, got %#v", want, calls)
	}

	settings := readDockerSettings(t, home)

	if settings["UseVirtualizationFrameworkRosetta"] != false || settings["useVirtualizationFrameworkRosetta"] != false {
		t.Fatalf("settings = %#v, want Docker Desktop Rosetta disabled", settings)
	}
}

func TestEnsurePrerequisitesDryRunReportsRosettaInstall(t *testing.T) {
	var calls []string

	var stdout bytes.Buffer
	home := t.TempDir()
	s := Service{
		GOOS:         "darwin",
		GOARCH:       "arm64",
		Home:         home,
		Stdout:       &stdout,
		Runner:       stubRunner{calls: &calls},
		rosettaCheck: func() bool { return false },
	}

	if err := s.EnsurePrerequisites(true); err != nil {
		t.Fatal(err)
	}

	for _, call := range calls {
		if strings.Contains(call, "softwareupdate") {
			t.Fatalf("dry-run should not invoke softwareupdate: %#v", calls)
		}
	}

	if !strings.Contains(stdout.String(), "would run: softwareupdate --install-rosetta --agree-to-license") {
		t.Fatalf("stdout = %s", stdout.String())
	}

	if !strings.Contains(stdout.String(), "would disable Docker Desktop Rosetta integration") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestEnsurePrerequisitesSurfacesRosettaInstallError(t *testing.T) {
	s := Service{
		GOOS:   "darwin",
		GOARCH: "arm64",
		Home:   t.TempDir(),
		Stdout: new(bytes.Buffer),
		Runner: stubRunner{
			errors: map[string]error{
				"softwareupdate --install-rosetta --agree-to-license": errors.New("boom"),
			},
		},
		rosettaCheck: func() bool { return false },
	}

	err := s.EnsurePrerequisites(false)

	if err == nil {
		t.Fatal("expected Rosetta install error")
	}

	if !strings.Contains(err.Error(), "Rosetta 2") {
		t.Fatalf("error = %v", err)
	}
}

func TestEnsurePrerequisitesPreservesDockerDesktopSettings(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, dockerDesktopSettingsPath)

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}

	original := `{"AutoStart":true,"SettingsVersion":43,"UseVirtualizationFrameworkRosetta":true}`

	if err := os.WriteFile(settingsPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	s := Service{
		GOOS:         "darwin",
		GOARCH:       "arm64",
		Home:         home,
		Stdout:       new(bytes.Buffer),
		Runner:       stubRunner{},
		rosettaCheck: func() bool { return true },
	}

	if err := s.EnsurePrerequisites(false); err != nil {
		t.Fatal(err)
	}

	settings := readDockerSettings(t, home)

	if settings["AutoStart"] != true || settings["SettingsVersion"] != float64(43) {
		t.Fatalf("settings = %#v, want existing settings preserved", settings)
	}

	if settings["UseVirtualizationFrameworkRosetta"] != false || settings["useVirtualizationFrameworkRosetta"] != false {
		t.Fatalf("settings = %#v, want Docker Desktop Rosetta disabled", settings)
	}
}

func TestEnsurePrerequisitesSkipsDockerDesktopSettingsOnIntel(t *testing.T) {
	home := t.TempDir()

	var calls []string

	s := Service{
		GOOS:   "darwin",
		GOARCH: "amd64",
		Home:   home,
		Stdout: new(bytes.Buffer),
		Runner: stubRunner{calls: &calls},
	}

	if err := s.EnsurePrerequisites(false); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(home, dockerDesktopSettingsPath)); !os.IsNotExist(err) {
		t.Fatalf("expected no Docker Desktop settings on intel, stat error = %v", err)
	}
}

func readDockerSettings(t *testing.T, home string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(home, dockerDesktopSettingsPath))

	if err != nil {
		t.Fatal(err)
	}

	var settings map[string]any

	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}

	return settings
}
