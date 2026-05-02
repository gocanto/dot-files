package brewfile

import (
	"strings"
	"testing"
)

func TestContentIncludesDevToolsAndStow(t *testing.T) {
	content := Content()

	for _, want := range []string{
		`brew "stow"`,
		`brew "age"`,
		`brew "agent-browser"`,
		`cask "codex"`,
		`cask "claude-code"`,
		`cask "1password-cli"`,
		`brew "mas"`,
		`brew "opencode"`,
		`brew "node@24"`,
		`brew "go"`,
		`brew "mysql"`,
		`brew "libpq"`,
		`cask "docker-desktop"`,
		`cask "visual-studio-code"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("Brewfile missing %s\n%s", want, content)
		}
	}
}
