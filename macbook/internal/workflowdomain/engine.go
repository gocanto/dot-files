package workflowdomain

import (
	"fmt"
	"io"
	"slices"

	"github.com/oullin/workflow/store"
	engine "github.com/oullin/workflow/workflow"
)

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

type EventSink interface {
	Emit(Event) error
}

type Executor struct {
	Sink EventSink
}

type eventWriter struct {
	emit     func(string) error
	writeErr error
	written  int
}

const (
	TransitionConfirm = "confirm"
	TransitionStart   = "start"
	TransitionSkip    = "skip"
	TransitionSucceed = "succeed"
	TransitionFail    = "fail"
	TransitionStop    = "stop"
	TransitionFinish  = "finish"
)

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

func BuildRunPlan(workflows []Workflow, req RunRequest) (RunPlan, error) {
	workflow, err := Find(workflows, req.WorkflowID)

	if err != nil {
		return RunPlan{}, err
	}

	plan := RunPlan{Workflow: *workflow, Phases: clonePhases(workflow.Phases), Mode: RunModeLive}

	if workflow.Confirmation != nil {
		option, err := findOption(workflow.Confirmation.Options, req.ConfirmationOptionID)

		if err != nil {
			return RunPlan{}, err
		}

		plan.ConfirmationOption = option

		if option.Phases != nil {
			plan.Phases = clonePhases(option.Phases)
		}

		switch {
		case option.Back:
			return RunPlan{}, fmt.Errorf("confirmation option %q goes back and cannot run", option.ID)
		case !option.Continue:
			plan.Mode = RunModeStopBeforeRun
		case option.ID == "preview-only":
			plan.Mode = RunModePreview
		default:
			plan.Mode = RunModeLive
		}
	}

	enabledIDs := map[string]bool{}

	for _, id := range req.EnabledPhaseIDs {
		enabledIDs[id] = true
	}

	if len(enabledIDs) > 0 {
		for index := range plan.Phases {
			plan.Phases[index].Enabled = enabledIDs[plan.Phases[index].ID]
		}
	}

	return plan, nil
}

func (e Executor) Execute(runID string, plan RunPlan) error {
	state := &RunState{Place: "pending"}
	flow, err := NewEngine()

	if err != nil {
		return err
	}

	if plan.ConfirmationOption != nil {
		if err := e.apply(flow, state, TransitionConfirm); err != nil {
			return err
		}

		if plan.ConfirmationOption.RequiresApproval {
			if err := e.approve(runID, plan.ConfirmationOption); err != nil {
				_ = e.apply(flow, state, TransitionFail)

				return err
			}
		}

		if plan.ConfirmationOption.Run != nil {
			if err := e.runConfirmation(runID, plan.ConfirmationOption); err != nil {
				_ = e.apply(flow, state, TransitionFail)

				return err
			}
		}

		if !plan.ConfirmationOption.Continue {
			if err := e.apply(flow, state, TransitionStop); err != nil {
				return err
			}

			return e.emit(Event{RunID: runID, Type: "run_stopped", Status: string(RunStatusStopped), Message: "Workflow stopped before phases."})
		}
	}

	for _, phase := range plan.Phases {
		if !phase.Enabled {
			if err := e.apply(flow, state, TransitionSkip); err != nil {
				return err
			}

			if err := e.emit(Event{RunID: runID, Type: "phase_skipped", PhaseID: phase.ID, PhaseName: phase.Name, Status: "skipped"}); err != nil {
				return err
			}

			continue
		}

		if err := e.apply(flow, state, TransitionStart); err != nil {
			return err
		}

		if err := e.emit(Event{RunID: runID, Type: "phase_started", PhaseID: phase.ID, PhaseName: phase.Name, Status: "running"}); err != nil {
			return err
		}

		err := e.runPhase(runID, phase)

		if err != nil {
			if applyErr := e.apply(flow, state, TransitionFail); applyErr != nil {
				return applyErr
			}

			_ = e.emit(Event{RunID: runID, Type: "phase_finished", PhaseID: phase.ID, PhaseName: phase.Name, Status: "failed", Message: err.Error()})

			return err
		}

		if err := e.apply(flow, state, TransitionSucceed); err != nil {
			return err
		}

		if err := e.emit(Event{RunID: runID, Type: "phase_finished", PhaseID: phase.ID, PhaseName: phase.Name, Status: "ok"}); err != nil {
			return err
		}
	}

	if err := e.apply(flow, state, TransitionFinish); err != nil {
		return err
	}

	return e.emit(Event{RunID: runID, Type: "run_finished", Status: string(RunStatusCompleted), Message: "Workflow completed."})
}

func NewEngine() (*engine.StateMachine[*RunState], error) {
	definition, err := engine.NewDefinitionBuilder().
		AddPlace("pending").
		AddPlace("confirmed").
		AddPlace("running").
		AddPlace("skipped").
		AddPlace("completed").
		AddPlace("failed").
		AddPlace("stopped").
		SetInitialPlaces("pending").
		AddTransition(TransitionConfirm, []string{"pending"}, []string{"confirmed"}).
		AddTransition("start_pending", []string{"pending"}, []string{"running"}).
		AddTransition("start_confirmed", []string{"confirmed"}, []string{"running"}).
		AddTransition("start_skipped", []string{"skipped"}, []string{"running"}).
		AddTransition("start_completed", []string{"completed"}, []string{"running"}).
		AddTransition("skip_pending", []string{"pending"}, []string{"skipped"}).
		AddTransition("skip_confirmed", []string{"confirmed"}, []string{"skipped"}).
		AddTransition("skip_skipped", []string{"skipped"}, []string{"skipped"}).
		AddTransition("skip_completed", []string{"completed"}, []string{"skipped"}).
		AddTransition(TransitionSucceed, []string{"running"}, []string{"completed"}).
		AddTransition("fail_pending", []string{"pending"}, []string{"failed"}).
		AddTransition("fail_confirmed", []string{"confirmed"}, []string{"failed"}).
		AddTransition("fail_running", []string{"running"}, []string{"failed"}).
		AddTransition(TransitionStop, []string{"confirmed"}, []string{"stopped"}).
		AddTransition("finish_pending", []string{"pending"}, []string{"completed"}).
		AddTransition("finish_confirmed", []string{"confirmed"}, []string{"completed"}).
		AddTransition("finish_skipped", []string{"skipped"}, []string{"completed"}).
		AddTransition("finish_completed", []string{"completed"}, []string{"completed"}).
		Build()

	if err != nil {
		return nil, err
	}

	markingStore := &store.SingleState[*RunState]{
		Getter: func(s *RunState) string { return s.Place },
		Setter: func(s *RunState, state string) { s.Place = state },
	}

	return engine.NewStateMachine("mac_os_workflow_run", definition, markingStore, nil)
}

func (e Executor) apply(flow *engine.StateMachine[*RunState], state *RunState, transition string) error {
	for _, candidate := range transitionCandidates(transition) {
		if !flow.Can(state, candidate) {
			continue
		}

		_, err := flow.Apply(state, candidate, nil)

		return err
	}

	return fmt.Errorf("invalid workflow transition %q from %q", transition, state.Place)
}

func transitionCandidates(transition string) []string {
	switch transition {
	case TransitionStart:
		return []string{"start_pending", "start_confirmed", "start_skipped", "start_completed"}
	case TransitionSkip:
		return []string{"skip_pending", "skip_confirmed", "skip_skipped", "skip_completed"}
	case TransitionFinish:
		return []string{"finish_pending", "finish_confirmed", "finish_skipped", "finish_completed"}
	case TransitionFail:
		return []string{"fail_pending", "fail_confirmed", "fail_running"}
	default:
		return []string{transition}
	}
}

func (e Executor) runConfirmation(runID string, option *ConfirmationOption) error {
	return e.runWriter(option.Run, func(message string) Event {
		return Event{RunID: runID, Type: "confirmation_output", Message: message}
	})
}

func (e Executor) approve(runID string, option *ConfirmationOption) error {
	if err := e.emit(Event{RunID: runID, Type: "permission_status", Status: "needs_approval", Message: "Host password approval required."}); err != nil {
		return err
	}

	err := e.runWriter(option.Approve, func(message string) Event {
		return Event{RunID: runID, Type: "permission_status", Message: message}
	})

	if err != nil {
		_ = e.emit(Event{RunID: runID, Type: "permission_status", Status: "failed", Message: "Host password approval failed."})

		return err
	}

	return e.emit(Event{RunID: runID, Type: "permission_status", Status: "ok", Message: "Host password approval accepted."})
}

func (e Executor) runPhase(runID string, phase Phase) error {
	return e.runWriter(phase.Run, func(message string) Event {
		return Event{RunID: runID, Type: "phase_output", PhaseID: phase.ID, PhaseName: phase.Name, Message: message}
	})
}

func (e Executor) emit(event Event) error {
	if e.Sink == nil {
		return nil
	}

	return e.Sink.Emit(event)
}

func findOption(options []ConfirmationOption, id string) (*ConfirmationOption, error) {
	if id == "" && len(options) > 0 {
		return &options[0], nil
	}

	for index := range options {
		if options[index].ID == id {
			return &options[index], nil
		}
	}

	return nil, fmt.Errorf("unknown confirmation option %q", id)
}

func clonePhases(phases []Phase) []Phase {
	return slices.Clone(phases)
}

func (e Executor) runWriter(run func(io.Writer) error, event func(string) Event) error {
	if run == nil {
		return nil
	}

	writer := &eventWriter{
		emit: func(message string) error {
			return e.emit(event(message))
		},
	}

	err := run(writer)

	if writer.writeErr != nil {
		return writer.writeErr
	}

	return err
}

func (w *eventWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if w.writeErr != nil {
		return 0, w.writeErr
	}

	if err := w.emit(string(p)); err != nil {
		w.writeErr = err

		return 0, err
	}

	w.written += len(p)

	return len(p), nil
}

func (w *eventWriter) Written() int {
	return w.written
}
