package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/gocanto/mac-os/internal/storage"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

func (a app) ui(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "missing ui command")
		a.uiUsage()

		return 2
	}

	switch args[0] {
	case "workflows":
		return a.uiWorkflows()
	case "run":
		return a.uiRun()
	case "runs":
		return a.uiRuns(args[1:])
	case "run-log":
		return a.uiRunLog(args[1:])
	default:
		fmt.Fprintf(a.stderr, "unknown ui command %q\n\n", args[0])
		a.uiUsage()

		return 2
	}
}

func (a app) uiUsage() {
	fmt.Fprintln(a.stderr, `Usage:
  mac-os ui workflows
  mac-os ui run
  mac-os ui runs
  mac-os ui run-log --run-id <id>`)
}

func (a app) uiWorkflows() int {
	return writeJSON(a.stdout, workflowdomain.Metadata(a.workflows()))
}

func (a app) uiRun() int {
	var req workflowdomain.RunRequest

	if err := json.NewDecoder(a.stdin).Decode(&req); err != nil {
		fmt.Fprintf(a.stderr, "decode run request: %v\n", err)

		return 2
	}

	plan, err := workflowdomain.BuildRunPlan(a.workflows(), req)

	if err != nil {
		fmt.Fprintf(a.stderr, "build run plan: %v\n", err)

		return 2
	}

	ctx := context.Background()
	store, err := storage.Open(ctx, storage.DefaultPath(a.home))

	if err != nil {
		fmt.Fprintf(a.stderr, "open workflow log database: %v\n", err)

		return 1
	}

	defer store.Close()

	runID := uuid.NewString()
	optionID, optionLabel := confirmationSelection(plan)

	if err := store.CreateRun(ctx, storage.RunStart{
		ID:                      runID,
		WorkflowID:              plan.Workflow.ID,
		WorkflowName:            plan.Workflow.Name,
		ConfirmationOptionID:    optionID,
		ConfirmationOptionLabel: optionLabel,
		Mode:                    plan.Mode,
		Status:                  workflowdomain.RunStatusRunning,
	}); err != nil {
		fmt.Fprintf(a.stderr, "create workflow run: %v\n", err)

		return 1
	}

	encoder := json.NewEncoder(a.stdout)
	recorder := storage.NewRecorder(store, runID, func(event workflowdomain.Event) error {
		return encoder.Encode(event)
	})

	if err := recorder.Emit(workflowdomain.Event{
		Type:    "run_started",
		Status:  string(workflowdomain.RunStatusRunning),
		Message: plan.Workflow.Name,
	}); err != nil {
		fmt.Fprintf(a.stderr, "record workflow start: %v\n", err)

		return 1
	}

	err = workflowdomain.Executor{Sink: recorder}.Execute(runID, plan)

	status := workflowdomain.RunStatusCompleted
	message := ""

	if plan.Mode == workflowdomain.RunModeStopBeforeRun && err == nil {
		status = workflowdomain.RunStatusStopped
	}

	if err != nil {
		status = workflowdomain.RunStatusFailed
		message = err.Error()

		_ = recorder.Emit(workflowdomain.Event{Type: "run_failed", Status: string(status), Message: message})
	}

	if completeErr := store.CompleteRun(ctx, runID, status, message); completeErr != nil {
		fmt.Fprintf(a.stderr, "complete workflow run: %v\n", completeErr)

		return 1
	}

	if err != nil {
		return 1
	}

	return 0
}

func (a app) uiRuns(args []string) int {
	fs := flag.NewFlagSet("runs", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	limit := fs.Int64("limit", 50, "maximum runs to return")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	store, err := storage.Open(context.Background(), storage.DefaultPath(a.home))

	if err != nil {
		fmt.Fprintf(a.stderr, "open workflow log database: %v\n", err)

		return 1
	}

	defer store.Close()

	runs, err := store.ListRuns(context.Background(), *limit)

	if err != nil {
		fmt.Fprintf(a.stderr, "list workflow runs: %v\n", err)

		return 1
	}

	return writeJSON(a.stdout, runs)
}

func (a app) uiRunLog(args []string) int {
	fs := flag.NewFlagSet("run-log", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	runID := fs.String("run-id", "", "workflow run id")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *runID == "" {
		fmt.Fprintln(a.stderr, "missing --run-id")

		return 2
	}

	store, err := storage.Open(context.Background(), storage.DefaultPath(a.home))

	if err != nil {
		fmt.Fprintf(a.stderr, "open workflow log database: %v\n", err)

		return 1
	}

	defer store.Close()

	log, err := store.RunLog(context.Background(), *runID)

	if err != nil {
		fmt.Fprintf(a.stderr, "read workflow run log: %v\n", err)

		return 1
	}

	return writeJSON(a.stdout, log)
}

func writeJSON(w io.Writer, value any) int {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return 1
	}

	return 0
}

func confirmationSelection(plan workflowdomain.RunPlan) (string, string) {
	if plan.ConfirmationOption == nil {
		return "", ""
	}

	return plan.ConfirmationOption.ID, plan.ConfirmationOption.Label
}
