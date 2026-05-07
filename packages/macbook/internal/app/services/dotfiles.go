package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gocanto/dot-files/internal/command"
	"github.com/gocanto/dot-files/internal/safefs"
	"github.com/gocanto/dot-files/internal/template/dotfiles"
)

const ohMyZshInstallURL = "https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh"

func (s Service) ApplyStow(opts Options) error {
	return stowService{home: s.Home, repo: s.Repo, stdout: s.Stdout, runner: s.Runner}.Apply(opts.DryRun)
}

func (s Service) AdoptDotfiles(opts Options) error {
	return dotfiles.Service{Home: s.Home, Repo: s.Repo, Stdout: s.Stdout}.Adopt(opts.DryRun)
}

func (s Service) EnsureOhMyZsh(opts Options) error {
	marker := filepath.Join(s.Home, ".oh-my-zsh", "oh-my-zsh.sh")

	if _, err := os.Stat(marker); err == nil {
		fmt.Fprintln(s.Stdout, "oh-my-zsh found")

		return nil
	}

	// Oh My Zsh documents the master-branch installer as the official install path.
	script := `set -e
RUNZSH=no CHSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL ` + ohMyZshInstallURL + `)"
`
	cmd := []string{"sh", "-c", script}

	if opts.DryRun {
		fmt.Fprintf(s.Stdout, "would run: %s\n", command.ShellQuote(cmd))

		return nil
	}

	out, err := s.Runner.Run(cmd[0], cmd[1:]...)

	if len(out) > 0 {
		fmt.Fprint(s.Stdout, string(out))
	}

	if err != nil {
		return fmt.Errorf("install oh-my-zsh: %w", err)
	}

	return nil
}

func (s Service) WriteDotfileCandidates(opts Options) error {
	dest := filepath.Join(s.Repo, ".template-candidates", "current-mac")
	fmt.Fprintf(s.Stdout, "dotfile review candidate destination: %s\n", dest)

	for _, item := range dotfiles.CapturePlan() {
		if opts.DryRun {
			fmt.Fprintf(s.Stdout, "would write dotfile candidate: %s -> %s\n", item.Source, filepath.Join(dest, item.Target))

			continue
		}

		if err := safefs.CopyPlanItem(dest, s.Home, item); err != nil {
			return fmt.Errorf("write dotfile candidate %s: %w", item.Target, err)
		}

		fmt.Fprintf(s.Stdout, "wrote dotfile candidate: %s\n", filepath.Join(dest, item.Target))
	}

	return nil
}
