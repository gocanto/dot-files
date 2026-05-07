package workflowdomain

type RunRequest struct {
	WorkflowID           string   `json:"workflowId"`
	ConfirmationOptionID string   `json:"confirmationOptionId"`
	EnabledPhaseIDs      []string `json:"enabledPhaseIds"`
}

type RunMode string

type RunStatus string

type Event struct {
	RunID     string `json:"runId"`
	Seq       int64  `json:"seq"`
	Type      string `json:"type"`
	PhaseID   string `json:"phaseId,omitempty"`
	PhaseName string `json:"phaseName,omitempty"`
	Status    string `json:"status,omitempty"`
	Message   string `json:"message,omitempty"`
}

type RunPlan struct {
	Workflow           Workflow
	ConfirmationOption *ConfirmationOption
	Phases             []Phase
	Mode               RunMode
}

type RunState struct {
	Place string
}

const ConfirmationOptionPreviewOnly = "preview-only"

const (
	RunModeLive          RunMode = "live"
	RunModePreview       RunMode = "preview"
	RunModeStopBeforeRun RunMode = "stop-before-run"
)

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusStopped   RunStatus = "stopped"
	RunStatusCancelled RunStatus = "cancelled"
)
