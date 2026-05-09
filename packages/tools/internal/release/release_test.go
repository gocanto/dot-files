package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("RELEASE_REPO", "env/repo")

	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "env repo fallback",
			args: []string{"--notes-file", "notes.md"},
			want: Config{NotesFile: "notes.md", Repo: "env/repo"},
		},
		{
			name: "flag repo wins",
			args: []string{"--notes-file=notes.md", "--repo", "flag/repo"},
			want: Config{NotesFile: "notes.md", Repo: "flag/repo"},
		},
		{
			name: "tag flag",
			args: []string{"--notes-file", "notes.md", "--tag", "v0.1.0-main"},
			want: Config{NotesFile: "notes.md", Repo: "env/repo", Tag: "v0.1.0-main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadConfig(tt.args)

			if err != nil {
				t.Fatal(err)
			}

			if got != tt.want {
				t.Fatalf("LoadConfig() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLoadConfigRejectsUnknownArgument(t *testing.T) {
	_, err := LoadConfig([]string{"--notes-file", "notes.md", "extra"})

	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadUIVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.json")

	if err := os.WriteFile(path, []byte(`{"version":"1.2.3"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readUIVersion(path)

	if err != nil {
		t.Fatal(err)
	}

	if got != "1.2.3" {
		t.Fatalf("readUIVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestFindArtifacts(t *testing.T) {
	releaseDir := t.TempDir()
	dmg := filepath.Join(releaseDir, "macOS Manager_0.1.0_x64.dmg")
	zip := filepath.Join(releaseDir, "macOS Manager_0.1.0_x64.zip")

	if err := os.WriteFile(dmg, []byte("dmg"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(zip, []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := findArtifacts(releaseDir)

	if err != nil {
		t.Fatal(err)
	}

	if got.DMG != dmg || got.ZIP != zip {
		t.Fatalf("findArtifacts() = %#v, want dmg=%q zip=%q", got, dmg, zip)
	}
}

func TestFindArtifactsRequiresDMGAndZIP(t *testing.T) {
	_, err := findArtifacts(t.TempDir())

	if err == nil {
		t.Fatal("expected error")
	}
}
