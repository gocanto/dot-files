package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/google/uuid"

	"github.com/gocanto/mac-os/internal/storage"
	"github.com/gocanto/mac-os/internal/workflowdomain"
)

type httpServer struct {
	app app
}

type runWorkflowRequest struct {
	WorkflowID           string   `json:"workflowId"`
	ConfirmationOptionID string   `json:"confirmationOptionId"`
	EnabledPhaseIDs      []string `json:"enabledPhaseIds"`
}

type validateSettingsRequest struct {
	Settings runtimeSettings `json:"settings"`
}

type savePreferencesRequest struct {
	Theme string `json:"theme"`
}

type runEndFrame struct {
	ExitCode int    `json:"exitCode"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

func (s httpServer) buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthz", s.healthz)
	mux.HandleFunc("GET /v1/workflows", s.listWorkflows)
	mux.HandleFunc("POST /v1/workflows/run", s.runWorkflow)
	mux.HandleFunc("GET /v1/template-files", s.listTemplateFiles)
	mux.HandleFunc("GET /v1/template-files/content", s.readTemplateFile)
	mux.HandleFunc("PUT /v1/template-files/content", s.saveTemplateFile)
	mux.HandleFunc("GET /v1/runs", s.listRuns)
	mux.HandleFunc("GET /v1/runs/{id}/log", s.runLog)
	mux.HandleFunc("GET /v1/settings", s.getSettings)
	mux.HandleFunc("POST /v1/settings/validate", s.validateSettings)
	mux.HandleFunc("GET /v1/preferences", s.getPreferences)
	mux.HandleFunc("POST /v1/preferences", s.savePreferences)
	mux.HandleFunc("GET /v1/onepassword/vaults", s.listOpVaults)
	mux.HandleFunc("GET /v1/onepassword/items", s.listOpItems)

	return mux
}

func (s httpServer) listTemplateFiles(w http.ResponseWriter, _ *http.Request) {
	files, err := s.app.listTemplateFiles()

	if err != nil {
		writeError(w, http.StatusInternalServerError, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s httpServer) readTemplateFile(w http.ResponseWriter, r *http.Request) {
	content, err := s.app.readTemplateFile(r.URL.Query().Get("path"))

	if err != nil {
		writeError(w, http.StatusBadRequest, err)

		return
	}

	writeJSON(w, http.StatusOK, content)
}

func (s httpServer) saveTemplateFile(w http.ResponseWriter, r *http.Request) {
	var req saveTemplateFileRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	content, err := s.app.saveTemplateFile(req.Path, req.Content)

	if err != nil {
		writeError(w, http.StatusBadRequest, err)

		return
	}

	writeJSON(w, http.StatusOK, content)
}

func (s httpServer) listOpVaults(w http.ResponseWriter, _ *http.Request) {
	vaults, err := s.app.listOpVaults()

	if err != nil {
		writeOpError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"vaults": vaults})
}

func (s httpServer) listOpItems(w http.ResponseWriter, r *http.Request) {
	vault := r.URL.Query().Get("vault")

	if vault == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("query parameter 'vault' is required"))

		return
	}

	items, err := s.app.listOpItems(vault)

	if err != nil {
		writeOpError(w, err)

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func writeOpError(w http.ResponseWriter, err error) {
	var unavailable ErrOpUnavailable

	if errors.As(err, &unavailable) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": unavailable.Reason,
			"code":  "op_unavailable",
		})

		return
	}

	writeError(w, http.StatusInternalServerError, err)
}

func (s httpServer) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s httpServer) listWorkflows(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"workflows": workflowdomain.Metadata(s.app.workflows()),
	})
}

func (s httpServer) listRuns(w http.ResponseWriter, r *http.Request) {
	limit := int64(0)

	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)

		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid limit: %w", err))

			return
		}

		limit = parsed
	}

	store, closeStore, err := s.app.workflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	runs, err := store.ListRuns(r.Context(), limit)

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("list workflow runs: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s httpServer) runLog(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")

	store, closeStore, err := s.app.workflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	log, err := store.RunLog(r.Context(), runID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("read workflow run log: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, log)
}

func (s httpServer) getSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, validateRuntimeSettings(s.app.home, s.app.repo, s.app.settings))
}

func (s httpServer) getPreferences(w http.ResponseWriter, r *http.Request) {
	store, closeStore, err := s.app.workflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	prefs, err := store.GetUserPreferences(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("read user preferences: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, prefs)
}

func (s httpServer) savePreferences(w http.ResponseWriter, r *http.Request) {
	var req savePreferencesRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	store, closeStore, err := s.app.workflowStore(r.Context())

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("open workflow log database: %w", err))

		return
	}

	defer closeStore()

	prefs, err := store.SaveUserPreferences(r.Context(), storage.UserPreferences{Theme: req.Theme})

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("save user preferences: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, prefs)
}

func (s httpServer) validateSettings(w http.ResponseWriter, r *http.Request) {
	var req validateSettingsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))

		return
	}

	writeJSON(w, http.StatusOK, validateRuntimeSettings(s.app.home, s.app.repo, req.Settings))
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		fmt.Fprintf(os.Stderr, "json encode error: %v\n", err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeSSE(w http.ResponseWriter, rc *http.ResponseController, event string, data any) error {
	payload, err := json.Marshal(data)

	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}

	return rc.Flush()
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
