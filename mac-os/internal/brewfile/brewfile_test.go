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
		`brew "codex"`,
		`brew "claude-code"`,
		`brew "mas"`,
		`brew "opencode"`,
		`brew "node@24"`,
		`brew "go"`,
		`brew "mysql"`,
		`brew "libpq"`,
		`cask "docker"`,
		`cask "visual-studio-code"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("Brewfile missing %s\n%s", want, content)
		}
	}
}
