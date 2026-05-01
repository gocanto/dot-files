package brewfile

import (
	"fmt"
	"strconv"
	"strings"
)

func Content() string {
	formulae := []string{
		"1password-cli",
		"age",
		"agent-browser",
		"autossh",
		"bruno",
		"claude-code",
		"codex",
		"csvlens",
		"fd",
		"ffmpeg",
		"fzf",
		"gh",
		"git",
		"glow",
		"go",
		"gnupg",
		"jq",
		"libavif",
		"libpq",
		"mas",
		"mysql",
		"nginx",
		"node@24",
		"opencode",
		"pinentry-mac",
		"portless",
		"sevenzip",
		"stow",
		"vim",
		"yazi",
		"zsh-syntax-highlighting",
	}
	casks := []string{
		"1password",
		"bruno",
		"claude",
		"codex",
		"codexbar",
		"dbeaver-community",
		"discord",
		"docker",
		"ghostty",
		"google-chrome",
		"iterm2",
		"jetbrains-toolbox",
		"jordanbaird-ice",
		"latest",
		"linearmouse",
		"markedit",
		"microsoft-teams",
		"notion",
		"postman",
		"raycast",
		"spotify",
		"stats",
		"sublime-text",
		"visual-studio-code",
		"zoom",
	}

	var b strings.Builder

	fmt.Fprintln(&b, `tap "homebrew/bundle"`)
	fmt.Fprintln(&b)

	for _, name := range formulae {
		fmt.Fprintf(&b, "brew %s\n", strconv.Quote(name))
	}

	fmt.Fprintln(&b)

	for _, name := range casks {
		fmt.Fprintf(&b, "cask %s\n", strconv.Quote(name))
	}

	return b.String()
}
