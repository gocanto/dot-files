package command

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestAugmentPathPrependsMissingDirs(t *testing.T) {
	exists := func(string) bool { return true }
	got := augmentPath("/usr/bin:/bin", []string{"/opt/homebrew/bin", "/usr/local/bin"}, exists)
	want := "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"

	if got != want {
		t.Fatalf("augmentPath = %q, want %q", got, want)
	}
}

func TestAugmentPathSkipsNonexistent(t *testing.T) {
	exists := func(p string) bool { return p == "/opt/homebrew/bin" }
	got := augmentPath("/usr/bin", []string{"/opt/homebrew/bin", "/does/not/exist"}, exists)
	want := "/opt/homebrew/bin:/usr/bin"

	if got != want {
		t.Fatalf("augmentPath = %q, want %q", got, want)
	}
}

func TestAugmentPathSkipsAlreadyPresent(t *testing.T) {
	exists := func(string) bool { return true }
	got := augmentPath("/opt/homebrew/bin:/usr/bin", []string{"/opt/homebrew/bin", "/usr/local/bin"}, exists)
	want := "/usr/local/bin:/opt/homebrew/bin:/usr/bin"

	if got != want {
		t.Fatalf("augmentPath = %q, want %q", got, want)
	}
}

func TestAugmentPathReturnsEmptyWhenNothingToAdd(t *testing.T) {
	exists := func(string) bool { return true }
	got := augmentPath("/opt/homebrew/bin:/usr/bin", []string{"/opt/homebrew/bin"}, exists)

	if got != "" {
		t.Fatalf("augmentPath = %q, want empty (no changes needed)", got)
	}
}

func TestAugmentPathHandlesEmptyCurrent(t *testing.T) {
	exists := func(string) bool { return true }
	got := augmentPath("", []string{"/opt/homebrew/bin"}, exists)
	want := "/opt/homebrew/bin"

	if got != want {
		t.Fatalf("augmentPath = %q, want %q", got, want)
	}
}

func TestEnsureSystemPathIsIdempotent(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("EnsureSystemPath only mutates PATH on darwin")
	}

	original := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", original) })

	EnsureSystemPath()
	first := os.Getenv("PATH")

	EnsureSystemPath()
	second := os.Getenv("PATH")

	if first != second {
		t.Fatalf("EnsureSystemPath not idempotent:\nfirst:  %q\nsecond: %q", first, second)
	}

	for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
		if !isDir(dir) {
			continue
		}

		if !strings.Contains(first, dir) {
			t.Errorf("expected PATH to contain %q after EnsureSystemPath, got %q", dir, first)
		}
	}
}

func TestEnsureSystemPathNoOpOnNonDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("non-darwin behavior")
	}

	original := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", original) })

	EnsureSystemPath()

	if os.Getenv("PATH") != original {
		t.Fatalf("EnsureSystemPath should be a no-op on %s", runtime.GOOS)
	}
}
