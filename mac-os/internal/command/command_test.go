package command

import "testing"

func TestShellQuote(t *testing.T) {
	got := ShellQuote([]string{"defaults", "write", "com.apple.finder", "FXPreferredViewStyle", "-string", "Nlsv"})
	want := "defaults write com.apple.finder FXPreferredViewStyle -string Nlsv"

	if got != want {
		t.Fatalf("ShellQuote = %q, want %q", got, want)
	}

	got = ShellQuote([]string{"brew", "bundle", "--file", "/Users/gus/Sites/mac os/Brewfile"})
	want = "brew bundle --file '/Users/gus/Sites/mac os/Brewfile'"

	if got != want {
		t.Fatalf("ShellQuote with space = %q, want %q", got, want)
	}
}
