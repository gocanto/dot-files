package apps

import (
	"sort"
	"strings"
	"unicode"

	"github.com/gocanto/mac-os/internal/template/appconfig"
)

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
	const minLenForSubstring = 4

	for key, app := range apps {
		appName := normalize(app.Name)

		if appName == want ||
			(len(want) >= minLenForSubstring && len(appName) >= minLenForSubstring &&
				(strings.Contains(want, appName) || strings.Contains(appName, want))) {
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
