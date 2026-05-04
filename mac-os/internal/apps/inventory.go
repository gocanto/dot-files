package apps

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/gocanto/mac-os/internal/appconfig"
	"github.com/gocanto/mac-os/internal/safefs"
	"go.yaml.in/yaml/v3"
)

type appInventory struct {
	Bundles []installedBundle
	Casks   []string
	MAS     []masInstall
}

type installedBundle struct {
	Name     string
	BundleID string
	Path     string
	System   bool
}

type masInstall struct {
	ID   string
	Name string
}

type detectedApp struct {
	Name          string
	BundleID      string
	InstallMethod string
	Package       string
	Path          string
	System        bool
}

type mergeSummary struct {
	Generated []appconfig.ManagedApp
	Matched   []string
	Missing   []string
	Added     []appconfig.ManagedApp
	Inventory appInventory
	Warnings  []string
	Output    string
}

type plist struct {
	Dict plistDict `xml:"dict"`
}

type plistDict struct {
	Items []plistItem `xml:",any"`
}

type plistItem struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func (s Service) GenerateInstalledList(opts Options) error {
	cfg, err := s.loader().Load(opts.ConfigPath)

	if err != nil {
		return err
	}

	summary, err := s.generateInstalledList(cfg, opts)

	if err != nil {
		return err
	}

	if opts.DryRun {
		printMergeSummary(s.Stdout, summary, true)

		return nil
	}

	out, err := yaml.Marshal(appconfig.Config{Apps: summary.Generated})

	if err != nil {
		return fmt.Errorf("encode generated app list: %w", err)
	}

	if err := safefs.WriteFile(summary.Output, out, 0o600); err != nil {
		return fmt.Errorf("write generated app list %s: %w", summary.Output, err)
	}

	printMergeSummary(s.Stdout, summary, false)

	return nil
}

func (s Service) generateInstalledList(cfg appconfig.Config, opts Options) (mergeSummary, error) {
	inventory, warnings, err := s.scanInventory(opts)

	if err != nil {
		return mergeSummary{}, err
	}

	summary := mergeApps(cfg, inventory)
	summary.Warnings = warnings
	summary.Output = s.generatedPath(opts.GeneratedPath)

	return summary, nil
}

func (s Service) scanInventory(opts Options) (appInventory, []string, error) {
	var warnings []string

	bundles, err := scanAppBundles(appRoots(opts, s.Home))

	if err != nil {
		return appInventory{}, nil, err
	}

	casksOut, err := s.Runner.Run("brew", "list", "--cask")
	casks := parseBrewCasks(casksOut)

	if err != nil {
		warnings = append(warnings, fmt.Sprintf("brew cask inventory failed: %v", err))
	}

	masOut, err := s.Runner.Run("mas", "list")
	masApps := parseMASList(masOut)

	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Mac App Store inventory failed: %v", err))
	}

	return appInventory{Bundles: bundles, Casks: casks, MAS: masApps}, warnings, nil
}

func appRoots(opts Options, home string) []string {
	if len(opts.AppRoots) > 0 {
		roots := make([]string, 0, len(opts.AppRoots))

		for _, root := range opts.AppRoots {
			roots = append(roots, safefs.ExpandHome(root, home))
		}

		return roots
	}

	return []string{"/Applications", filepath.Join(home, "Applications"), "/System/Applications"}
}

func (s Service) generatedPath(path string) string {
	if path == "" {
		return filepath.Join(s.Repo, "apps.generated.yaml")
	}

	return s.loader().Path(path)
}

func scanAppBundles(roots []string) ([]installedBundle, error) {
	var bundles []installedBundle
	seen := map[string]bool{}

	for _, root := range roots {
		info, err := os.Stat(root)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, err
		}

		if !info.IsDir() {
			continue
		}

		systemRoot := strings.HasPrefix(filepath.Clean(root), "/System/")

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".app" {
				return nil
			}

			metadata, err := readBundleMetadata(path)

			if err != nil {
				name := strings.TrimSuffix(filepath.Base(path), ".app")
				metadata = installedBundle{Name: name}
			}

			metadata.Path = path
			metadata.System = systemRoot || strings.HasPrefix(filepath.Clean(path), "/System/")

			key := metadata.BundleID

			if key == "" {
				key = normalize(metadata.Name)
			}

			if key == "" || seen[key] {
				return filepath.SkipDir
			}

			seen[key] = true
			bundles = append(bundles, metadata)

			return filepath.SkipDir
		})

		if err != nil {
			return nil, err
		}
	}

	sort.Slice(bundles, func(i, j int) bool {
		return strings.ToLower(bundles[i].Name) < strings.ToLower(bundles[j].Name)
	})

	return bundles, nil
}

func readBundleMetadata(appPath string) (installedBundle, error) {
	infoPath := filepath.Join(appPath, "Contents", "Info.plist")
	data, err := os.ReadFile(infoPath)

	if err != nil {
		return installedBundle{}, err
	}

	values := parsePlistStrings(data)

	if values["CFBundleIdentifier"] == "" {
		if converted, err := exec.Command("plutil", "-convert", "xml1", "-o", "-", infoPath).Output(); err == nil {
			values = parsePlistStrings(converted)
		}
	}

	name := firstNonEmpty(values["CFBundleDisplayName"], values["CFBundleName"], strings.TrimSuffix(filepath.Base(appPath), ".app"))

	return installedBundle{
		Name:     name,
		BundleID: values["CFBundleIdentifier"],
	}, nil
}

func parsePlistStrings(data []byte) map[string]string {
	values := map[string]string{}

	if !bytes.Contains(data, []byte("<plist")) {
		return values
	}

	var p plist

	if err := xml.Unmarshal(data, &p); err != nil {
		return values
	}

	var key string

	for _, item := range p.Dict.Items {
		switch item.XMLName.Local {
		case "key":
			key = strings.TrimSpace(item.Value)
		case "string":
			if key != "" {
				values[key] = strings.TrimSpace(item.Value)
			}

			key = ""
		default:
			key = ""
		}
	}

	return values
}

func parseBrewCasks(out []byte) []string {
	lines := strings.Split(string(out), "\n")
	casks := make([]string, 0, len(lines))
	seen := map[string]bool{}

	for _, line := range lines {
		name := strings.TrimSpace(line)

		if name == "" || seen[name] {
			continue
		}

		seen[name] = true
		casks = append(casks, name)
	}

	sort.Strings(casks)

	return casks
}

func parseMASList(out []byte) []masInstall {
	lines := strings.Split(string(out), "\n")
	apps := make([]masInstall, 0, len(lines))
	seen := map[string]bool{}

	for _, line := range lines {
		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		id := fields[0]
		rest := strings.TrimSpace(strings.TrimPrefix(line, id))
		version := strings.LastIndex(rest, " (")

		if version >= 0 {
			rest = strings.TrimSpace(rest[:version])
		}

		if id == "" || rest == "" || seen[id] {
			continue
		}

		seen[id] = true
		apps = append(apps, masInstall{ID: id, Name: rest})
	}

	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps
}

func mergeApps(cfg appconfig.Config, inventory appInventory) mergeSummary {
	detected := detectApps(inventory)
	detectedByBundle := map[string]detectedApp{}
	detectedByName := map[string]detectedApp{}
	detectedByPackage := map[string]detectedApp{}

	for _, app := range detected {
		if app.BundleID != "" {
			detectedByBundle[app.BundleID] = app
		}

		if key := normalize(app.Name); key != "" {
			detectedByName[key] = app
		}

		if key := normalize(app.Package); key != "" {
			detectedByPackage[key] = app
		}
	}

	generated := append([]appconfig.ManagedApp{}, cfg.Apps...)

	var matched []string

	var missing []string
	matchedDetected := map[string]bool{}

	for _, app := range cfg.Apps {
		detectedApp, ok := findDetectedMatch(app, detectedByBundle, detectedByName, detectedByPackage)

		if !ok {
			missing = append(missing, app.Name)

			continue
		}

		matched = append(matched, app.Name)
		matchedDetected[detectedKey(detectedApp)] = true
	}

	var added []appconfig.ManagedApp

	for _, app := range detected {
		if matchedDetected[detectedKey(app)] {
			continue
		}

		if existingAppMatchesDetected(cfg.Apps, app) {
			continue
		}

		managed := appconfig.ManagedApp{
			Name:          app.Name,
			BundleID:      app.BundleID,
			InstallMethod: app.InstallMethod,
			Package:       app.Package,
			ConfigMode:    "manual",
		}
		generated = append(generated, managed)
		added = append(added, managed)
	}

	sortManagedApps(generated)
	sortManagedApps(added)

	sort.Strings(matched)

	sort.Strings(missing)

	return mergeSummary{Generated: generated, Matched: matched, Missing: missing, Added: added, Inventory: inventory}
}

func detectApps(inventory appInventory) []detectedApp {
	detectedByKey := map[string]detectedApp{}

	for _, bundle := range inventory.Bundles {
		app := detectedApp{
			Name:          bundle.Name,
			BundleID:      bundle.BundleID,
			InstallMethod: "manual",
			Path:          bundle.Path,
			System:        bundle.System,
		}

		if bundle.System {
			app.InstallMethod = "system"
		}

		detectedByKey[detectedKey(app)] = app
	}

	for _, cask := range inventory.Casks {
		key, app, ok := findDetectedByLooseName(detectedByKey, cask)

		if !ok {
			app = detectedApp{Name: titleFromPackage(cask)}
			key = detectedKey(app)
		}

		app.InstallMethod = "brew"
		app.Package = cask
		detectedByKey[key] = app
	}

	for _, masApp := range inventory.MAS {
		key, app, ok := findDetectedByLooseName(detectedByKey, masApp.Name)

		if !ok {
			app = detectedApp{Name: masApp.Name}
			key = detectedKey(app)
		}

		app.InstallMethod = "mas"
		app.Package = masApp.ID
		detectedByKey[key] = app
	}

	detected := make([]detectedApp, 0, len(detectedByKey))

	for _, app := range detectedByKey {
		detected = append(detected, app)
	}

	sort.Slice(detected, func(i, j int) bool {
		return strings.ToLower(detected[i].Name) < strings.ToLower(detected[j].Name)
	})

	return detected
}

func findDetectedByLooseName(apps map[string]detectedApp, name string) (string, detectedApp, bool) {
	want := normalize(name)

	for key, app := range apps {
		appName := normalize(app.Name)

		if appName == want || strings.Contains(want, appName) || strings.Contains(appName, want) {
			return key, app, true
		}
	}

	return "", detectedApp{}, false
}

func findDetectedMatch(app appconfig.ManagedApp, byBundle map[string]detectedApp, byName map[string]detectedApp, byPackage map[string]detectedApp) (detectedApp, bool) {
	if app.BundleID != "" {
		if detected, ok := byBundle[app.BundleID]; ok {
			return detected, true
		}
	}

	if app.Package != "" {
		if detected, ok := byPackage[normalize(app.Package)]; ok {
			return detected, true
		}
	}

	if detected, ok := byName[normalize(app.Name)]; ok {
		return detected, true
	}

	return detectedApp{}, false
}

func existingAppMatchesDetected(apps []appconfig.ManagedApp, detected detectedApp) bool {
	for _, app := range apps {
		if _, ok := findDetectedMatch(app, map[string]detectedApp{detected.BundleID: detected}, map[string]detectedApp{normalize(detected.Name): detected}, map[string]detectedApp{normalize(detected.Package): detected}); ok {
			return true
		}
	}

	return false
}

func detectedKey(app detectedApp) string {
	if app.BundleID != "" {
		return "bundle:" + app.BundleID
	}

	if app.Name != "" {
		return "name:" + normalize(app.Name)
	}

	return "package:" + normalize(app.Package)
}

func sortManagedApps(apps []appconfig.ManagedApp) {
	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})
}

func printMergeSummary(stdout interface{ Write([]byte) (int, error) }, summary mergeSummary, dryRun bool) {
	fmt.Fprintf(stdout, "installed app inventory: %d app bundles, %d Homebrew casks, %d App Store apps\n", len(summary.Inventory.Bundles), len(summary.Inventory.Casks), len(summary.Inventory.MAS))

	for _, warning := range summary.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}

	fmt.Fprintf(stdout, "matched tracked apps: %d\n", len(summary.Matched))
	fmt.Fprintf(stdout, "added detected apps: %d\n", len(summary.Added))
	fmt.Fprintf(stdout, "missing tracked apps: %d\n", len(summary.Missing))

	for _, app := range summary.Added {
		if app.Package != "" {
			fmt.Fprintf(stdout, "added app: %s (%s %s)\n", app.Name, app.InstallMethod, app.Package)
		} else {
			fmt.Fprintf(stdout, "added app: %s (%s)\n", app.Name, app.InstallMethod)
		}
	}

	for _, name := range summary.Missing {
		fmt.Fprintf(stdout, "missing tracked app: %s\n", name)
	}

	if dryRun {
		fmt.Fprintf(stdout, "would write generated app list: %s\n", summary.Output)
	} else {
		fmt.Fprintf(stdout, "wrote generated app list: %s\n", summary.Output)
	}
}

func normalize(value string) string {
	var b strings.Builder

	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func titleFromPackage(pkg string) string {
	parts := strings.FieldsFunc(pkg, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})

	for i, part := range parts {
		if part == "" {
			continue
		}

		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}

	return strings.Join(parts, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
