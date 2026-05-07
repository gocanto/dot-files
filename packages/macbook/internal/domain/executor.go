package domain

import (
	"context"
	"io"
)

type EventSink interface {
	Emit(context.Context, Event) error
}

type Executor struct {
	Sink EventSink
}

func (e Executor) Execute(ctx context.Context, runID string, plan RunPlan) error {
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
			if err := e.approve(ctx, runID, plan.ConfirmationOption); err != nil {
				_ = e.apply(flow, state, TransitionFail)

				return err
			}
		}

		if plan.ConfirmationOption.Run != nil {
			if err := e.runConfirmation(ctx, runID, plan.ConfirmationOption); err != nil {
				_ = e.apply(flow, state, TransitionFail)

				return err
			}
		}

		if !plan.ConfirmationOption.Continue {
			if err := e.apply(flow, state, TransitionStop); err != nil {
				return err
			}

			return e.emit(ctx, Event{RunID: runID, Type: "run_stopped", Status: string(RunStatusStopped), Message: "Workflow stopped before phases."})
		}
	}

	for _, phase := range plan.Phases {
		if !phase.Enabled {
			if err := e.apply(flow, state, TransitionSkip); err != nil {
				return err
			}

			if err := e.emit(ctx, Event{RunID: runID, Type: "phase_skipped", PhaseID: phase.ID, PhaseName: phase.Name, Status: "skipped"}); err != nil {
				return err
			}

			continue
		}

		if err := e.apply(flow, state, TransitionStart); err != nil {
			return err
		}

		if err := e.emit(ctx, Event{RunID: runID, Type: "phase_started", PhaseID: phase.ID, PhaseName: phase.Name, Status: "running"}); err != nil {
			return err
		}

		err := e.runPhase(ctx, runID, phase)

		if err != nil {
			if applyErr := e.apply(flow, state, TransitionFail); applyErr != nil {
				return applyErr
			}

			if emitErr := e.emit(ctx, Event{RunID: runID, Type: "phase_finished", PhaseID: phase.ID, PhaseName: phase.Name, Status: "failed", Message: err.Error()}); emitErr != nil {
				return emitErr
			}

			return err
		}

		if err := e.apply(flow, state, TransitionSucceed); err != nil {
			return err
		}

		if err := e.emit(ctx, Event{RunID: runID, Type: "phase_finished", PhaseID: phase.ID, PhaseName: phase.Name, Status: "ok"}); err != nil {
			return err
		}
	}

	if err := e.apply(flow, state, TransitionFinish); err != nil {
		return err
	}

	return e.emit(ctx, Event{RunID: runID, Type: "run_finished", Status: string(RunStatusCompleted), Message: "Workflow completed."})
}

func (e Executor) runConfirmation(ctx context.Context, runID string, option *ConfirmationOption) error {
	return e.runWriter(ctx, option.Run, func(message string) Event {
		return Event{RunID: runID, Type: "confirmation_output", Message: message}
	})
}

func (e Executor) approve(ctx context.Context, runID string, option *ConfirmationOption) error {
	if err := e.emit(ctx, Event{RunID: runID, Type: "permission_status", Status: "needs_approval", Message: "Host password approval required."}); err != nil {
		return err
	}

	err := e.runWriter(ctx, option.Approve, func(message string) Event {
		return Event{RunID: runID, Type: "permission_status", Message: message}
	})

	if err != nil {
		_ = e.emit(ctx, Event{RunID: runID, Type: "permission_status", Status: "failed", Message: "Host password approval failed."})

		return err
	}

	return e.emit(ctx, Event{RunID: runID, Type: "permission_status", Status: "ok", Message: "Host password approval accepted."})
}

func (e Executor) runPhase(ctx context.Context, runID string, phase Phase) error {
	return e.runWriter(ctx, phase.Run, func(message string) Event {
		return Event{RunID: runID, Type: "phase_output", PhaseID: phase.ID, PhaseName: phase.Name, Message: message}
	})
}

func (e Executor) emit(ctx context.Context, event Event) error {
	if e.Sink == nil {
		return nil
	}

	return e.Sink.Emit(ctx, event)
}

func (e Executor) runWriter(ctx context.Context, run func(io.Writer) error, event func(string) Event) error {
	if run == nil {
		return nil
	}

	writer := &eventWriter{
		emit: func(message string) error {
			return e.emit(ctx, event(message))
		},
	}

	err := run(writer)

	if writer.writeErr != nil {
		return writer.writeErr
	}

	return err
}
