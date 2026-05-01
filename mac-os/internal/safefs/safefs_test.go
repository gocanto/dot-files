package safefs

import (
	"strings"
	"testing"
)

func TestShouldSkipSensitive(t *testing.T) {
	cases := map[string]bool{
		"/Users/gus/.ssh/id_ed25519":                                                 true,
		"/Users/gus/.ssh/id_ed25519.pub":                                             false,
		"/Users/gus/.zsh_history":                                                    true,
		"/Users/gus/.config/gh/hosts.yml":                                            true,
		"/Users/gus/.config/ghostty/config":                                          false,
		"/Users/gus/Library/Application Support/Code/User/settings.json":             false,
		"/Users/gus/Library/Application Support/Code/User/globalStorage/state.vscdb": true,
		"/Users/gus/Library/Application Support/Google/Chrome/Default/Cookies":       true,
		"/Users/gus/Library/Keychains/login.keychain-db":                             true,
		"/Users/gus/.claude/file-history/abc/def":                                    true,
	}

	for path, want := range cases {
		if got := ShouldSkipSensitive(path); got != want {
			t.Fatalf("ShouldSkipSensitive(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestSanitizeDotfileRedactsMachineSpecificSettings(t *testing.T) {
	input := []byte("[coderabbit]\n\tmachineId = cli/example\n[core]\n\teditor = vim\n")
	got := string(SanitizeDotfile("/Users/gus/.gitconfig", "/Users/gus", input))

	if strings.Contains(got, "cli/example") {
		t.Fatalf("SanitizeDotfile leaked machine id: %s", got)
	}

	if !strings.Contains(got, "editor = vim") {
		t.Fatalf("SanitizeDotfile removed safe config: %s", got)
	}
}

func TestSanitizeDotfileRewritesHomePath(t *testing.T) {
	input := []byte(`export PATH="/Users/gus/bin:$PATH"`)
	got := string(SanitizeDotfile("/Users/gus/.zshrc", "/Users/gus", input))

	if strings.Contains(got, "/Users/gus") {
		t.Fatalf("SanitizeDotfile leaked absolute home path: %s", got)
	}

	if !strings.Contains(got, "$HOME/bin") {
		t.Fatalf("SanitizeDotfile did not rewrite home path: %s", got)
	}
}
