package turbocache

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPackageDirs(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "packages", "ui"))
	mustMkdir(t, filepath.Join(root, "packages", "macbook"))
	mustMkdir(t, filepath.Join(root, "packages", "notes"))
	mustWrite(t, filepath.Join(root, "packages", "ui", "package.json"), "{}")
	mustWrite(t, filepath.Join(root, "packages", "macbook", "package.json"), "{}")

	got, err := PackageDirs(root)

	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, "packages", "macbook"),
		filepath.Join(root, "packages", "ui"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PackageDirs() = %#v, want %#v", got, want)
	}
}

func TestMovePackageLogsMovesDirectoryContents(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "packages", "ui")
	turboDir := filepath.Join(packageDir, ".turbo")
	destinationDir := filepath.Join(root, "storage", ".cache", "turbo-logs", "ui")

	mustMkdir(t, turboDir)
	mustWrite(t, filepath.Join(packageDir, "package.json"), "{}")
	mustWrite(t, filepath.Join(turboDir, "turbo-build.log"), "new")
	mustMkdir(t, destinationDir)
	mustWrite(t, filepath.Join(destinationDir, "turbo-build.log"), "old")

	if err := MovePackageLogs(root); err != nil {
		t.Fatal(err)
	}

	assertMissing(t, turboDir)
	assertFile(t, filepath.Join(destinationDir, "turbo-build.log"), "new")
}

func TestMovePackageLogsRemovesBrokenSymlink(t *testing.T) {
	root := t.TempDir()
	packageDir := filepath.Join(root, "packages", "ui")
	turboDir := filepath.Join(packageDir, ".turbo")

	mustMkdir(t, packageDir)
	mustWrite(t, filepath.Join(packageDir, "package.json"), "{}")

	if err := os.Symlink(filepath.Join(root, "missing"), turboDir); err != nil {
		t.Fatal(err)
	}

	if err := MovePackageLogs(root); err != nil {
		t.Fatal(err)
	}

	assertMissing(t, turboDir)
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path string, want string) {
	t.Helper()
	got, err := os.ReadFile(path)

	if err != nil {
		t.Fatal(err)
	}

	if string(got) != want {
		t.Fatalf("%s = %q, want %q", path, string(got), want)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists, err=%v", path, err)
	}
}
