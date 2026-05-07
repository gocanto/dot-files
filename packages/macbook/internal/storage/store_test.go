package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gocanto/mac-os/internal/workflowdomain"
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
		Mode:                    workflowdomain.RunModeLive,
		Status:                  workflowdomain.RunStatusRunning,
	}); err != nil {
		t.Fatal(err)
	}

	recorder := NewRecorder(store, "run-1", nil)

	for _, event := range []workflowdomain.Event{
		{Type: "phase_started", PhaseID: "doctor", PhaseName: "Run health checks", Status: "running"},
		{Type: "phase_output", PhaseID: "doctor", PhaseName: "Run health checks", Message: "ok"},
	} {
		if err := recorder.Emit(ctx, event); err != nil {
			t.Fatal(err)
		}
	}

	if err := store.CompleteRun(ctx, "run-1", workflowdomain.RunStatusCompleted, ""); err != nil {
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
