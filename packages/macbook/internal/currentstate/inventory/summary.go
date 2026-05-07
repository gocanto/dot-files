package inventory

import "fmt"

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
