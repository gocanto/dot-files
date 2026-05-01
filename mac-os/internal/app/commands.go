package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/gocanto/mac-os/internal/brewfile"
	"github.com/gocanto/mac-os/internal/secrets"
)

func (a app) bootstrap(args []string) int {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show commands without changing the machine")
	fs.BoolVar(&opts.yes, "yes", false, "run all phases without prompting")
	fs.BoolVar(&opts.apps, "apps", false, "include app install/config phases from apps.yaml")
	fs.StringVar(&opts.archivePath, "archive", "", "restore app configuration from this archive during bootstrap")
	fs.StringVar(&opts.configPath, "config", "", "app restore config path")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	for _, phase := range a.bootstrapPhases(opts) {
		if err := a.confirmAndRun(phase.Name, opts, func() error { return phase.Run(a.stdout) }); err != nil {
			fmt.Fprintf(a.stderr, "%s failed: %v\n", phase.Name, err)

			return 1
		}
	}

	return 0
}

func (a app) adopt(args []string) int {
	fs := flag.NewFlagSet("adopt", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show files without importing them")
	fs.BoolVar(&opts.yes, "yes", false, "import without prompting")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.confirmAndRun("adopt safe dotfiles", opts, func() error { return a.adoptDotfiles(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "adopt failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) capture(args []string) int {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show capture plan without writing files")
	fs.BoolVar(&opts.yes, "yes", false, "capture without prompting")
	fs.BoolVar(&opts.encrypt, "encrypt", false, "package and encrypt the archive with Age using 1Password metadata")
	fs.BoolVar(&opts.apps, "apps", false, "include allowlisted app configuration from apps.yaml")
	fs.StringVar(&opts.archiveRoot, "archive-root", "", "directory where timestamped archives are stored")
	fs.StringVar(&opts.configPath, "config", "", "app restore config path")
	fs.StringVar(&opts.opVault, "op-vault", defaultOPVault, "1Password vault containing archive metadata")
	fs.StringVar(&opts.opItem, "op-item", defaultOPItem, "1Password item containing archive metadata")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.confirmAndRun("private archive capture", opts, func() error { return a.captureArchive(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "capture failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) restore(args []string) int {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show restore plan without writing files")
	fs.BoolVar(&opts.yes, "yes", false, "restore without prompting")
	fs.BoolVar(&opts.apps, "apps", false, "restore allowlisted app configuration from apps.yaml")
	fs.StringVar(&opts.archivePath, "archive", "", "archive directory to restore from")
	fs.StringVar(&opts.configPath, "config", "", "app restore config path")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if opts.archivePath == "" {
		fmt.Fprintln(a.stderr, "restore requires --archive PATH")

		return 2
	}

	if !opts.apps {
		fmt.Fprintln(a.stderr, "restore currently requires --apps")

		return 2
	}

	if err := a.confirmAndRun("app config restore", opts, func() error { return a.restoreAppConfigs(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "restore failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) doctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.runDoctor(options{}); err != nil {
		fmt.Fprintf(a.stderr, "doctor failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) brewfile(args []string) int {
	fs := flag.NewFlagSet("brewfile", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	writePath := fs.String("write", "", "write Brewfile to this path")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	content := brewfile.Content()

	if *writePath == "" {
		fmt.Fprint(a.stdout, content)

		return 0
	}

	if err := os.WriteFile(*writePath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(a.stderr, "write Brewfile: %v\n", err)

		return 1
	}

	fmt.Fprintf(a.stdout, "wrote %s\n", *writePath)

	return 0
}

func (a app) macos(args []string) int {
	fs := flag.NewFlagSet("macos", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show defaults without applying")
	fs.BoolVar(&opts.yes, "yes", false, "apply without prompting")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := a.confirmAndRun("macOS defaults", opts, func() error { return a.applyMacOSDefaults(opts) }); err != nil {
		fmt.Fprintf(a.stderr, "macOS defaults failed: %v\n", err)

		return 1
	}

	return 0
}

func (a app) secrets(args []string) int {
	if len(args) == 0 {
		a.secretsUsage()

		return 0
	}

	fs := flag.NewFlagSet("secrets "+args[0], flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	opts := options{}
	fs.BoolVar(&opts.dryRun, "dry-run", false, "show secret workflow without writing files")
	fs.StringVar(&opts.secretTarget, "target", "", "secret target name from secrets.yaml")
	fs.StringVar(&opts.secretsPath, "secrets-config", "", "secret manifest config path")
	fs.StringVar(&opts.opVault, "op-vault", defaultOPVault, "1Password vault containing secret metadata")
	fs.StringVar(&opts.opItem, "op-item", defaultOPItem, "1Password item containing secret metadata")

	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	var err error

	switch args[0] {
	case "encrypt":
		err = a.encryptSecrets(opts)
	case "decrypt":
		err = a.decryptSecrets(opts)
	case "sync":
		err = a.syncSecrets(opts)
	case "encrypt-gitconfig":
		opts.secretTarget = secrets.GitconfigSecret
		err = a.encryptSecrets(opts)
	case "decrypt-gitconfig":
		opts.secretTarget = secrets.GitconfigSecret
		err = a.decryptSecrets(opts)
	case "sync-gitconfig":
		opts.secretTarget = secrets.GitconfigSecret
		err = a.syncSecrets(opts)
	case "help", "-h", "--help":
		a.secretsUsage()

		return 0
	default:
		fmt.Fprintf(a.stderr, "unknown secrets command %q\n\n", args[0])
		a.secretsUsage()

		return 2
	}

	if err != nil {
		fmt.Fprintf(a.stderr, "secrets %s failed: %v\n", args[0], err)

		return 1
	}

	return 0
}

func (a app) tui(args []string) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	result, err := a.tuiRunner(a.stdin, a.stdout, a.tuiWorkflows())

	if err != nil {
		fmt.Fprintf(a.stderr, "tui failed: %v\n", err)

		return 1
	}

	return result.ExitCode
}
