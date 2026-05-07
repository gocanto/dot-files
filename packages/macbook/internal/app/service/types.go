package service

import (
	"io"

	"github.com/gocanto/dot-files/internal/app/setting"
	"github.com/gocanto/dot-files/internal/command"
)

type Service struct {
	Home     string
	Repo     string
	GOOS     string
	GOARCH   string
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Runner   command.Runner
	Settings setting.RuntimeSettings
}

type Options struct {
	DryRun            bool
	Encrypt           bool
	Apps              bool
	AllowMasUninstall bool
	ArchiveRoot       string
	ArchivePath       string
	UseLatestArchive  bool
	ConfigPath        string
	GeneratedPath     string
	SecretsPath       string
	OPVault           string
	OPItem            string
}
