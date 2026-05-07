package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/gocanto/mac-os/internal/storage"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

type runWorkflowRequest struct {
	WorkflowID           string   `json:"workflowId"`
	ConfirmationOptionID string   `json:"confirmationOptionId"`
	EnabledPhaseIDs      []string `json:"enabledPhaseIds"`
}

type runEndFrame struct {
	ExitCode int    `json:"exitCode"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

func (s httpServer) listWorkflows(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"workflows": workflowdomain.Metadata(s.app.workflows()),
	})
}

func (s httpServer) runWorkflow(w http.ResponseWriter, r *http.Request) {
	var req runWorkflowRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	plan, err := workflowdomain.BuildRunPlan(s.app.workflows(), workflowdomain.RunRequest{
		WorkflowID:           req.WorkflowID,
		ConfirmationOptionID: req.ConfirmationOptionID,
		EnabledPhaseIDs:      req.EnabledPhaseIDs,
	})

	if err != nil {
		writeError(w, http.StatusBadRequest, err)

		return
	}

	store, closeStore, err := s.app.workflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	_ = rc.Flush()

	runID := uuid.NewString()
	optionID, optionLabel := confirmationSelection(plan)

	if err := store.CreateRun(r.Context(), storage.RunStart{
		ID:                      runID,
		WorkflowID:              plan.Workflow.ID,
		WorkflowName:            plan.Workflow.Name,
		ConfirmationOptionID:    optionID,
		ConfirmationOptionLabel: optionLabel,
		Mode:                    plan.Mode,
		Status:                  workflowdomain.RunStatusRunning,
	}); err != nil {
		writeSSE(w, rc, "error", map[string]string{"message": fmt.Sprintf("create workflow run: %v", err)})

		return
	}

	recorder := storage.NewRecorder(store, runID, func(event workflowdomain.Event) error {
		return writeSSE(w, rc, "workflow", event)
	})

	if err := recorder.Emit(r.Context(), workflowdomain.Event{
		Type:    "run_started",
		Status:  string(workflowdomain.RunStatusRunning),
		Message: plan.Workflow.Name,
	}); err != nil {
		writeSSE(w, rc, "error", map[string]string{"message": fmt.Sprintf("record workflow start: %v", err)})

		return
	}

	runErr := workflowdomain.Executor{Sink: recorder}.Execute(r.Context(), runID, plan)
	statusValue, message := finalRunStatus(plan, runErr)

	if runErr != nil {
		_ = recorder.Emit(r.Context(), workflowdomain.Event{Type: "run_failed", Status: string(statusValue), Message: message})
	}

	if err := store.CompleteRun(r.Context(), runID, statusValue, message); err != nil {
		writeSSE(w, rc, "error", map[string]string{"message": fmt.Sprintf("complete workflow run: %v", err)})

		return
	}

	exitCode := 0

	if runErr != nil {
		exitCode = 1
	}

	_ = writeSSE(w, rc, "end", runEndFrame{
		ExitCode: exitCode,
		Status:   string(statusValue),
		Message:  message,
	})
}

func confirmationSelection(plan workflowdomain.RunPlan) (string, string) {
	if plan.ConfirmationOption == nil {
		return "", ""
	}

	return plan.ConfirmationOption.ID, plan.ConfirmationOption.Label
}

func finalRunStatus(plan workflowdomain.RunPlan, runErr error) (workflowdomain.RunStatus, string) {
	if runErr != nil {
		return workflowdomain.RunStatusFailed, runErr.Error()
	}

	if plan.Mode == workflowdomain.RunModeStopBeforeRun {
		return workflowdomain.RunStatusStopped, ""
	}

	return workflowdomain.RunStatusCompleted, ""
}
