package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gocanto/dot-files/internal/converge/appstore"
	"github.com/gocanto/dot-files/internal/snapshot"
)

func (s Service) CaptureArchive(opts Options) error {
	return snapshot.Service{Home: s.Home, Repo: s.Repo, Stdout: s.Stdout, Stderr: s.Stderr, Runner: s.Runner}.Capture(snapshot.Options{
		DryRun:      opts.DryRun,
		Encrypt:     opts.Encrypt,
		Apps:        opts.Apps,
		ArchiveRoot: opts.ArchiveRoot,
		ConfigPath:  opts.ConfigPath,
		OPVault:     opts.OPVault,
		OPItem:      opts.OPItem,
	})
}

func (s Service) RestoreAppConfigs(opts Options) error {
	archivePath := opts.ArchivePath

	if opts.UseLatestArchive && archivePath == "" {
		archiveRoot := opts.ArchiveRoot

		if archiveRoot == "~" {
			archiveRoot = s.Home
		}

		if strings.HasPrefix(archiveRoot, "~/") {
			archiveRoot = filepath.Join(s.Home, strings.TrimPrefix(archiveRoot, "~/"))
		}

		if archiveRoot == "" {
			archiveRoot = snapshot.DefaultLocalRoot(s.Home)
		}

		latest, ok, err := snapshot.LatestSnapshot(archiveRoot)

		if err != nil {
			return err
		}

		if !ok {
			fmt.Fprintf(s.Stdout, "skipped: no local app settings snapshot found under %s\n", archiveRoot)

			return nil
		}

		archivePath = latest
		fmt.Fprintf(s.Stdout, "using latest local app settings snapshot: %s\n", archivePath)
	}

	return s.appstore().RestoreConfigs(appstore.Options{DryRun: opts.DryRun, Apps: opts.Apps, ArchivePath: archivePath, ConfigPath: opts.ConfigPath})
}
