package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gocanto/dot-files/internal/domain"
)

func TestStorePersistsRunsAndEventsInOrder(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "runs.sqlite3"))

	if err != nil {
		t.Fatal(err)
	}

	defer store.Close()

	if err := store.Init(ctx); err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	if err := store.Init(ctx); err != nil {
		t.Fatalf("second init should be idempotent: %v", err)
	}

	if err := store.CreateRun(ctx, RunStart{
		ID:                      "run-1",
		WorkflowID:              "check-setup",
		WorkflowName:            "Check Setup",
		ConfirmationOptionID:    "run-now",
		ConfirmationOptionLabel: "Run now",
		Mode:                    domain.RunModeLive,
		Status:                  domain.RunStatusRunning,
	}); err != nil {
		t.Fatal(err)
	}

	recorder := NewRecorder(store, "run-1", nil)

	for _, event := range []domain.Event{
		{Type: "phase_started", PhaseID: "doctor", PhaseName: "Run health checks", Status: "running"},
		{Type: "phase_output", PhaseID: "doctor", PhaseName: "Run health checks", Message: "ok"},
	} {
		if err := recorder.Emit(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	if err := store.CompleteRun(ctx, "run-1", domain.RunStatusCompleted, ""); err != nil {
		t.Fatal(err)
	}

	runs, err := store.ListRuns(ctx, 10)

	if err != nil {
		t.Fatal(err)
	}

	if len(runs) != 1 || runs[0].Status != "completed" {
		t.Fatalf("runs = %#v", runs)
	}

	log, err := store.RunLog(ctx, "run-1")

	if err != nil {
		t.Fatal(err)
	}

	if len(log.Events) != 2 || log.Events[0].Seq != 1 || log.Events[1].Message != "ok" {
		t.Fatalf("log events = %#v", log.Events)
	}
}

func TestDefaultPathMigratesOldNamespaceDatabase(t *testing.T) {
	t.Setenv(envDBPath, "")

	home := t.TempDir()
	oldPath := filepath.Join(home, "Library", "Application Support", "mac-os", "workflows.sqlite3")

	if err := os.MkdirAll(filepath.Dir(oldPath), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(oldPath, []byte("old-db"), 0o600); err != nil {
		t.Fatal(err)
	}

	newPath := DefaultPath(home)
	wantPath := filepath.Join(home, "Library", "Application Support", "dot-files", "workflows.sqlite3")

	if newPath != wantPath {
		t.Fatalf("path = %q, want %q", newPath, wantPath)
	}

	got, err := os.ReadFile(newPath)

	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "old-db" {
		t.Fatalf("new database content = %q", got)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old database still exists or stat failed: %v", err)
	}
}

func TestDefaultPathDoesNotOverwriteNewNamespaceDatabase(t *testing.T) {
	t.Setenv(envDBPath, "")

	home := t.TempDir()
	oldPath := filepath.Join(home, "Library", "Application Support", "mac-os", "workflows.sqlite3")
	newPath := filepath.Join(home, "Library", "Application Support", "dot-files", "workflows.sqlite3")

	for _, path := range []string{oldPath, newPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.WriteFile(oldPath, []byte("old-db"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(newPath, []byte("new-db"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := DefaultPath(home); got != newPath {
		t.Fatalf("path = %q, want %q", got, newPath)
	}

	got, err := os.ReadFile(newPath)

	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "new-db" {
		t.Fatalf("new database was overwritten with %q", got)
	}
}
