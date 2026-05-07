package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gocanto/mac-os/internal/command"
	"github.com/gocanto/mac-os/internal/safefs"
	"github.com/gocanto/mac-os/internal/template/dotfiles"
)

const ohMyZshInstallURL = "https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh"

func (a app) applyStow(opts options) error {
	return stowService{home: a.home, repo: a.repo, stdout: a.stdout, runner: a.runner}.Apply(opts.dryRun)
}

func (a app) adoptDotfiles(opts options) error {
	return dotfiles.Service{Home: a.home, Repo: a.repo, Stdout: a.stdout}.Adopt(opts.dryRun)
}

func (a app) ensureOhMyZsh(opts options) error {
	marker := filepath.Join(a.home, ".oh-my-zsh", "oh-my-zsh.sh")

	if _, err := os.Stat(marker); err == nil {
		fmt.Fprintln(a.stdout, "oh-my-zsh found")

		return nil
	}

	// Oh My Zsh documents the master-branch installer as the official install path.
	script := `set -e
RUNZSH=no CHSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL ` + ohMyZshInstallURL + `)"
`
	cmd := []string{"sh", "-c", script}

	if opts.dryRun {
		fmt.Fprintf(a.stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	out, err := a.runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(a.stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("install oh-my-zsh: %w", err)
	}

	return nil
}

func (a app) writeDotfileCandidates(opts options) error {
	dest := filepath.Join(a.repo, ".template-candidates", "current-mac")
	fmt.Fprintf(a.stdout, "dotfile review candidate destination: %s\n", dest)

	for _, item := range dotfiles.CapturePlan() {
		if opts.dryRun {
			fmt.Fprintf(a.stdout, "would write dotfile candidate: %s -> %s\n", item.Source, filepath.Join(dest, item.Target))

			continue
		}

		if err := safefs.CopyPlanItem(dest, a.home, item); err != nil {
			return fmt.Errorf("write dotfile candidate %s: %w", item.Target, err)
		}

		fmt.Fprintf(a.stdout, "wrote dotfile candidate: %s\n", filepath.Join(dest, item.Target))
	}

	return nil
}
