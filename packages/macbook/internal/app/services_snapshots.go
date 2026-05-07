package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gocanto/mac-os/internal/converge/appstore"
	"github.com/gocanto/mac-os/internal/snapshot"
)

func (a app) captureArchive(opts options) error {
	return snapshot.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout, Stderr: a.stderr, Runner: a.runner}.Capture(snapshot.Options{
		DryRun:      opts.dryRun,
		Encrypt:     opts.encrypt,
		Apps:        opts.apps,
		ArchiveRoot: opts.archiveRoot,
		ConfigPath:  opts.configPath,
		OPVault:     opts.opVault,
		OPItem:      opts.opItem,
	})
}

func (a app) restoreAppConfigs(opts options) error {
	archivePath := opts.archivePath

	if opts.useLatestArchive && archivePath == "" {
		archiveRoot := opts.archiveRoot

		if archiveRoot == "~" {
			archiveRoot = a.home
		}

		if strings.HasPrefix(archiveRoot, "~/") {
			archiveRoot = filepath.Join(a.home, strings.TrimPrefix(archiveRoot, "~/"))
		}

		if archiveRoot == "" {
			archiveRoot = snapshot.DefaultLocalRoot(a.home)
		}

		latest, ok, err := snapshot.LatestSnapshot(archiveRoot)

		if err != nil {
			return err
		}

		if !ok {
			fmt.Fprintf(a.stdout, "skipped: no local app settings snapshot found under %s\n", archiveRoot)

			return nil
		}

		archivePath = latest
		fmt.Fprintf(a.stdout, "using latest local app settings snapshot: %s\n", archivePath)
	}

	return a.appstore().RestoreConfigs(appstore.Options{DryRun: opts.dryRun, Apps: opts.apps, ArchivePath: archivePath, ConfigPath: opts.configPath})
}
