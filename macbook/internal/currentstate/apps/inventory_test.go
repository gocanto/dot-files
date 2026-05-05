package apps

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/template/appconfig"
)

type inventoryRunner struct {
	outputs map[string][]byte
	errors  map[string]error
}

func (r inventoryRunner) Run(name string, args ...string) ([]byte, error) {
	key := command.ShellQuote(append([]string{name}, args...))

	return r.outputs[key], r.errors[key]
}

func TestParseBrewCasks(t *testing.T) {
	got := parseBrewCasks([]byte("ghostty\nvisual-studio-code\n\nghostty\n"))
	want := []string{"ghostty", "visual-studio-code"}

	if !slices.Equal(got, want) {
		t.Fatalf("parseBrewCasks = %#v, want %#v", got, want)
	}
}

func TestParseMASList(t *testing.T) {
	got := parseMASList([]byte("1502839586 Hand Mirror (3.0.1)\n803453959 Slack (4.42.0)\n"))
	want := []masInstall{{ID: "1502839586", Name: "Hand Mirror"}, {ID: "803453959", Name: "Slack"}}

	if !slices.Equal(got, want) {
		t.Fatalf("parseMASList = %#v, want %#v", got, want)
	}
}

func TestScanAppBundlesReadsMetadata(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "Example.app", "Contents")

	if err := os.MkdirAll(app, 0o700); err != nil {
		t.Fatal(err)
	}

	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key>
  <string>com.example.Example</string>
  <key>CFBundleName</key>
  <string>Example App</string>
</dict>
</plist>
`

	if err := os.WriteFile(filepath.Join(app, "Info.plist"), []byte(plist), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := scanAppBundles([]string{root})

	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("bundle count = %d, want 1: %#v", len(got), got)
	}

	if got[0].Name != "Example App" || got[0].BundleID != "com.example.Example" {
		t.Fatalf("bundle metadata = %#v", got[0])
	}
}

func TestMergeAppsPreservesExistingMetadataAndAddsDetectedApps(t *testing.T) {
	cfg := appconfig.Config{Apps: []appconfig.ManagedApp{
		{
			Name:          "Ghostty",
			BundleID:      "com.mitchellh.ghostty",
			InstallMethod: "brew",
			Package:       "ghostty",
			ConfigMode:    "auto",
			ConfigPaths: []appconfig.ConfigPath{
				{Source: "~/.config/ghostty/config", Target: "ghostty/config"},
			},
		},
		{Name: "Missing App", InstallMethod: "manual", ConfigMode: "manual"},
	}}
	inventory := appInventory{
		Bundles: []installedBundle{
			{Name: "Ghostty", BundleID: "com.mitchellh.ghostty", Path: "/Applications/Ghostty.app"},
			{Name: "New Brew", BundleID: "com.example.NewBrew", Path: "/Applications/New Brew.app"},
			{Name: "Store App", BundleID: "com.example.Store", Path: "/Applications/Store App.app"},
			{Name: "Loose App", BundleID: "com.example.Loose", Path: "/Applications/Loose App.app"},
			{Name: "System Tool", BundleID: "com.apple.SystemTool", Path: "/System/Applications/System Tool.app", System: true},
		},
		Casks: []string{"ghostty", "new-brew"},
		MAS:   []masInstall{{ID: "1234567890", Name: "Store App"}},
	}

	summary := mergeApps(cfg, inventory)
	byName := map[string]appconfig.ManagedApp{}

	for _, app := range summary.Generated {
		byName[app.Name] = app
	}

	ghostty := byName["Ghostty"]

	if ghostty.ConfigMode != "auto" || len(ghostty.ConfigPaths) != 1 {
		t.Fatalf("existing metadata was not preserved: %#v", ghostty)
	}

	for name, want := range map[string]appconfig.ManagedApp{
		"New Brew":    {InstallMethod: "brew", Package: "new-brew"},
		"Store App":   {InstallMethod: "mas", Package: "1234567890"},
		"Loose App":   {InstallMethod: "manual"},
		"System Tool": {InstallMethod: "system"},
	} {
		got, ok := byName[name]

		if !ok {
			t.Fatalf("missing generated app %q in %#v", name, summary.Generated)
		}

		if got.InstallMethod != want.InstallMethod || got.Package != want.Package || got.ConfigMode != "manual" {
			t.Fatalf("%s = %#v, want method %q package %q manual config", name, got, want.InstallMethod, want.Package)
		}
	}

	if !slices.Contains(summary.Matched, "Ghostty") {
		t.Fatalf("matched = %#v, want Ghostty", summary.Matched)
	}

	if !slices.Contains(summary.Missing, "Missing App") {
		t.Fatalf("missing = %#v, want Missing App", summary.Missing)
	}
}

func TestGenerateInstalledListDryRunDoesNotWriteCandidate(t *testing.T) {
	dir := t.TempDir()
	appRoot := filepath.Join(dir, "Applications")
	configPath := filepath.Join(dir, "apps.yaml")
	outputPath := filepath.Join(dir, "apps.generated.yaml")

	if err := os.MkdirAll(filepath.Join(appRoot, "Existing.app", "Contents"), 0o700); err != nil {
		t.Fatal(err)
	}

	plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.example.Existing</string><key>CFBundleName</key><string>Existing</string></dict></plist>`

	if err := os.WriteFile(filepath.Join(appRoot, "Existing.app", "Contents", "Info.plist"), []byte(plist), 0o600); err != nil {
		t.Fatal(err)
	}

	config := []byte(`
apps:
  - name: Existing
    bundle_id: com.example.Existing
    install_method: manual
    config_mode: manual
`)

	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	service := Service{
		Home:   dir,
		Repo:   dir,
		Stdout: &stdout,
		Runner: inventoryRunner{outputs: map[string][]byte{
			"brew list --cask": []byte(""),
			"mas list":         []byte(""),
		}, errors: map[string]error{}},
	}

	if err := service.GenerateInstalledList(Options{DryRun: true, ConfigPath: configPath, GeneratedPath: outputPath, AppRoots: []string{appRoot}}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry run wrote output path, stat err = %v", err)
	}

	if !strings.Contains(stdout.String(), "would write generated app list: "+outputPath) {
		t.Fatalf("stdout missing dry-run output path:\n%s", stdout.String())
	}
}

func TestGenerateInstalledListWritesCandidateWithoutChangingSource(t *testing.T) {
	dir := t.TempDir()
	appRoot := filepath.Join(dir, "Applications")
	configPath := filepath.Join(dir, "apps.yaml")
	outputPath := filepath.Join(dir, "apps.generated.yaml")

	if err := os.MkdirAll(filepath.Join(appRoot, "New App.app", "Contents"), 0o700); err != nil {
		t.Fatal(err)
	}

	plist := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>com.example.NewApp</string><key>CFBundleName</key><string>New App</string></dict></plist>`

	if err := os.WriteFile(filepath.Join(appRoot, "New App.app", "Contents", "Info.plist"), []byte(plist), 0o600); err != nil {
		t.Fatal(err)
	}

	source := []byte(`
apps:
  - name: Existing
    install_method: manual
    config_mode: manual
`)

	if err := os.WriteFile(configPath, source, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	service := Service{
		Home:   dir,
		Repo:   dir,
		Stdout: &stdout,
		Runner: inventoryRunner{outputs: map[string][]byte{
			"brew list --cask": []byte(""),
			"mas list":         []byte(""),
		}, errors: map[string]error{}},
	}

	if err := service.GenerateInstalledList(Options{ConfigPath: configPath, GeneratedPath: outputPath, AppRoots: []string{appRoot}}); err != nil {
		t.Fatal(err)
	}

	unchanged, err := os.ReadFile(configPath)

	if err != nil {
		t.Fatal(err)
	}

	if string(unchanged) != string(source) {
		t.Fatalf("source config changed:\n%s", string(unchanged))
	}

	generated, err := os.ReadFile(outputPath)

	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"name: New App", "bundle_id: com.example.NewApp", "install_method: manual", "wrote generated app list: " + outputPath} {
		if !strings.Contains(string(generated)+stdout.String(), want) {
			t.Fatalf("generated output missing %q\ngenerated:\n%s\nstdout:\n%s", want, string(generated), stdout.String())
		}
	}
}
