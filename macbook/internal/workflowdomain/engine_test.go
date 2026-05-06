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

func TestExecutorStreamsPhaseOutputBeforePhaseFinishes(t *testing.T) {
	collector := &eventCollector{}
	plan := RunPlan{
		Workflow: Workflow{Name: "Sample"},
		Phases: []Phase{
			{
				Name:    "stream",
				Enabled: true,
				Run: func(w io.Writer) error {
					if _, err := io.WriteString(w, "first\n"); err != nil {
						return err
					}

					_, err := io.WriteString(w, "second\n")

					return err
				},
			},
		},
	}
	plan.Workflow = Normalize([]Workflow{plan.Workflow})[0]

	executor := Executor{Sink: collector}

	if err := executor.Execute("run-1", plan); err != nil {
		t.Fatal(err)
	}

	var outputs []Event
	finishedIndex := -1

	for index, event := range collector.events {
		if event.Type == "phase_output" {
			outputs = append(outputs, event)
		}

		if event.Type == "phase_finished" {
			finishedIndex = index
		}
	}

	if len(outputs) != 2 {
		t.Fatalf("outputs = %#v, want two streamed chunks", outputs)
	}

	if outputs[0].Message != "first\n" || outputs[1].Message != "second\n" {
		t.Fatalf("output messages = %#v", outputs)
	}

	if finishedIndex < 0 {
		t.Fatalf("missing phase_finished in %#v", collector.events)
	}

	for _, output := range outputs {
		for index, event := range collector.events {
			if event == output && index > finishedIndex {
				t.Fatalf("output emitted after finish: events = %#v", collector.events)
			}
		}
	}
}
