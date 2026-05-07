package domain

import (
	"fmt"

	"github.com/oullin/workflow/store"
	engine "github.com/oullin/workflow/workflow"
)

const (
	TransitionConfirm = "confirm"
	TransitionStart   = "start"
	TransitionSkip    = "skip"
	TransitionSucceed = "succeed"
	TransitionFail    = "fail"
	TransitionStop    = "stop"
	TransitionFinish  = "finish"
)

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
