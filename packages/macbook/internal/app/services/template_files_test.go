package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gocanto/dot-files/internal/app/setting"
)

func TestTemplateFilesListReadAndSaveAllowlistedFiles(t *testing.T) {
	home := t.TempDir()
	repo := writeSettingsRepo(t)
	stowFile := filepath.Join(repo, "stow", "shell", ".zshrc")

	if err := os.MkdirAll(filepath.Dir(stowFile), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(stowFile, []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := Service{Home: home, Repo: repo, Settings: setting.DefaultRuntimeSettings(home, repo), Runner: stubRunner{}}
	files, err := a.ListTemplateFiles()

	if err != nil {
		t.Fatal(err)
	}

	if !hasTemplateFile(files, "apps.yaml") || !hasTemplateFile(files, "stow/shell/.zshrc") {
		t.Fatalf("files = %#v", files)
	}

	content, err := a.ReadTemplateFile("stow/shell/.zshrc")

	if err != nil {
		t.Fatal(err)
	}

	if content.Content != "export EDITOR=vim\n" {
		t.Fatalf("content = %q", content.Content)
	}

	saved, err := a.SaveTemplateFile("stow/shell/.zshrc", "export EDITOR=nvim\n")

	if err != nil {
		t.Fatal(err)
	}

	if saved.Content != "export EDITOR=nvim\n" {
		t.Fatalf("saved = %#v", saved)
	}
}

func TestTemplateFilesRejectUnsafeFiles(t *testing.T) {
	home := t.TempDir()
	repo := writeSettingsRepo(t)
	a := Service{Home: home, Repo: repo, Settings: setting.DefaultRuntimeSettings(home, repo), Runner: stubRunner{}}

	outside := filepath.Join(t.TempDir(), "outside.txt")

	if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := a.ReadTemplateFile(outside); err == nil {
		t.Fatal("expected outside path to be rejected")
	}

	if _, err := a.ReadTemplateFile("../go.mod"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}

	if _, err := a.SaveTemplateFile(outside, "changed\n"); err == nil {
		t.Fatal("expected outside save path to be rejected")
	}

	if _, err := a.SaveTemplateFile("../go.mod", "changed\n"); err == nil {
		t.Fatal("expected traversal save path to be rejected")
	}

	binary := filepath.Join(repo, "stow", "git", ".config", "git", "binary")

	if err := os.WriteFile(binary, []byte{0, 1, 2}, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := a.ReadTemplateFile(binary); err == nil {
		t.Fatal("expected binary file to be rejected")
	}
}

func hasTemplateFile(files []templateFileSummary, relative string) bool {
	for _, file := range files {
		if file.Relative == relative {
			return true
		}
	}

	return false
}
