package dotfiles

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gocanto/mac-os/internal/safefs"
)

type Service struct {
	Home   string
	Repo   string
	Stdout io.Writer
}

func AdoptPlan(home, repo string) []safefs.Item {
	return []safefs.Item{
		{Source: filepath.Join(home, ".zshrc"), Target: filepath.Join(repo, "stow/shell/.zshrc")},
		{Source: filepath.Join(home, ".zprofile"), Target: filepath.Join(repo, "stow/shell/.zprofile")},
		{Source: filepath.Join(home, ".bash_profile"), Target: filepath.Join(repo, "stow/shell/.bash_profile")},
		{Source: filepath.Join(home, ".gitconfig"), Target: filepath.Join(repo, "stow/git/.gitconfig")},
		{Source: filepath.Join(home, ".vimrc"), Target: filepath.Join(repo, "stow/vim/.vimrc")},
		{Source: filepath.Join(home, ".config/git/ignore"), Target: filepath.Join(repo, "stow/git/.config/git/ignore")},
		{Source: filepath.Join(home, ".config/ghostty/config"), Target: filepath.Join(repo, "stow/ghostty/.config/ghostty/config")},
	}
}

func CapturePlan() []safefs.Item {
	return []safefs.Item{
		{Source: "~/.zshrc", Target: "dotfiles/.zshrc"},
		{Source: "~/.zprofile", Target: "dotfiles/.zprofile"},
		{Source: "~/.bash_profile", Target: "dotfiles/.bash_profile"},
		{Source: "~/.gitconfig", Target: "dotfiles/.gitconfig"},
		{Source: "~/.vimrc", Target: "dotfiles/.vimrc"},
		{Source: "~/.config/git/ignore", Target: "dotfiles/.config/git/ignore"},
		{Source: "~/.config/ghostty/config", Target: "dotfiles/.config/ghostty/config"},
		{Source: "~/.vscode/extensions/extensions.json", Target: "editors/vscode/extensions.json"},
		{Source: "~/Library/Application Support/Code/User/settings.json", Target: "editors/vscode/settings.json"},
	}
}

func (s Service) Adopt(dryRun bool) error {
	for _, item := range AdoptPlan(s.Home, s.Repo) {
		if dryRun {
			fmt.Fprintf(s.Stdout, "would import: %s -> %s\n", item.Source, item.Target)

			continue
		}

		if safefs.ShouldSkipSensitive(item.Source) {
			fmt.Fprintf(s.Stdout, "skipped sensitive path: %s\n", item.Source)

			continue
		}

		data, err := os.ReadFile(item.Source)

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(s.Stdout, "missing, skipped: %s\n", item.Source)

				continue
			}

			return err
		}

		data = safefs.SanitizeDotfile(item.Source, s.Home, data)

		if err := safefs.WriteFile(item.Target, data, 0o600); err != nil {
			return err
		}

		fmt.Fprintf(s.Stdout, "imported: %s\n", item.Target)
	}

	return nil
}
