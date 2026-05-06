package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	a := newApp(home, repo, strings.NewReader(""), nil, nil, stubRunner{})
	files, err := a.listTemplateFiles()

	if err != nil {
		t.Fatal(err)
	}

	if !hasTemplateFile(files, "apps.yaml") || !hasTemplateFile(files, "stow/shell/.zshrc") {
		t.Fatalf("files = %#v", files)
	}

	content, err := a.readTemplateFile("stow/shell/.zshrc")

	if err != nil {
		t.Fatal(err)
	}

	if content.Content != "export EDITOR=vim\n" {
		t.Fatalf("content = %q", content.Content)
	}

	saved, err := a.saveTemplateFile(stowFile, "export EDITOR=nvim\n")

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
	a := newApp(home, repo, strings.NewReader(""), nil, nil, stubRunner{})

	outside := filepath.Join(t.TempDir(), "outside.txt")

	if err := os.WriteFile(outside, []byte("outside\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := a.readTemplateFile(outside); err == nil {
		t.Fatal("expected outside path to be rejected")
	}

	if _, err := a.readTemplateFile("../go.mod"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}

	binary := filepath.Join(repo, "stow", "git", ".config", "git", "binary")

	if err := os.WriteFile(binary, []byte{0, 1, 2}, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := a.readTemplateFile(binary); err == nil {
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
