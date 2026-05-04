package workflowdomain

import (
	"errors"
	"io"
	"testing"
)

type eventCollector struct {
	events []Event
}

func (c *eventCollector) Emit(event Event) error {
	c.events = append(c.events, event)

	return nil
}

func TestEngineAllowsExpectedTransitions(t *testing.T) {
	flow, err := NewEngine()

	if err != nil {
		t.Fatal(err)
	}

	state := &RunState{Place: "pending"}
	executor := Executor{}

	for _, transition := range []string{TransitionConfirm, TransitionStart, TransitionSucceed, TransitionFinish} {
		if err := executor.apply(flow, state, transition); err != nil {
			t.Fatalf("apply %q: %v", transition, err)
		}
	}

	if state.Place != "completed" {
		t.Fatalf("place = %q, want completed", state.Place)
	}
}

func TestBuildRunPlanSelectsPreviewPhases(t *testing.T) {
	workflows := []Workflow{
		{
			Name:   "Sample",
			Phases: []Phase{{Name: "live", Enabled: true}},
			Confirmation: &Confirmation{Options: []ConfirmationOption{
				{Label: "Preview only", Continue: true, Phases: []Phase{{Name: "preview", Enabled: true}}},
				{Label: "Run now", Continue: true, Phases: []Phase{{Name: "live", Enabled: true}}},
			}},
		},
	}

	plan, err := BuildRunPlan(workflows, RunRequest{WorkflowID: "sample", ConfirmationOptionID: "preview-only"})

	if err != nil {
		t.Fatal(err)
	}

	if plan.Mode != RunModePreview || plan.Phases[0].Name != "preview" {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestExecutorStopsOnFirstFailingPhase(t *testing.T) {
	collector := &eventCollector{}
	plan := RunPlan{
		Workflow: Workflow{Name: "Sample"},
		Phases: []Phase{
			{Name: "one", Enabled: true, Run: func(io.Writer) error { return errors.New("boom") }},
			{Name: "two", Enabled: true, Run: func(io.Writer) error { return nil }},
		},
	}
	plan.Workflow = Normalize([]Workflow{plan.Workflow})[0]

	err := Executor{Sink: collector}.Execute("run-1", plan)

	if err == nil {
		t.Fatal("expected error")
	}

	for _, event := range collector.events {
		if event.PhaseName == "two" {
			t.Fatalf("second phase should not run: %#v", collector.events)
		}
	}
}
