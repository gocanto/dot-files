package tui

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testWorkflow(err error) []Workflow {
	return []Workflow{
		{
			Name:        "Set Up This Mac",
			Description: "Run setup.",
			ChangesMac:  "Yes",
			Phases: []Phase{
				{Name: "one", Enabled: true, Run: func(io.Writer) error { return err }},
				{Name: "two", Enabled: true, Run: func(io.Writer) error { return nil }},
			},
		},
		{
			Name:        "Check Setup",
			Description: "Run health checks.",
			ChangesMac:  "No",
			Phases: []Phase{
				{Name: "doctor", Enabled: true, Run: func(io.Writer) error { return nil }},
			},
		},
	}
}

func viewString(m Model) string {
	return m.View().Content
}

func key(code rune) tea.KeyPressMsg {
	text := ""

	if code >= 32 && code != tea.KeySpace {
		text = string(code)
	}

	if code == tea.KeySpace {
		text = " "
	}

	return tea.KeyPressMsg(tea.Key{Code: code, Text: text})
}

func ctrl(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Mod: tea.ModCtrl})
}

func TestInitialMenuState(t *testing.T) {
	m := New(testWorkflow(nil))

	for _, want := range []string{"Set Up This Mac", "Check Setup", "Run setup.", "Yes"} {
		if !strings.Contains(viewString(m), want) {
			t.Fatalf("initial view missing %q:\n%s", want, viewString(m))
		}
	}
}

func TestConfirmationCanReplacePhasesForPreview(t *testing.T) {
	var liveRan bool

	var previewRan bool
	workflows := []Workflow{
		{
			Name:       "Set Up This Mac",
			ChangesMac: "Yes",
			Phases: []Phase{{Name: "live", Enabled: true, Run: func(io.Writer) error {
				liveRan = true

				return nil
			}}},
			Confirmation: &Confirmation{
				Title:   "Set Up This Mac",
				Message: "Choose how to proceed.",
				Options: []ConfirmationOption{
					{
						Label:    "Preview only",
						Continue: true,
						Phases: []Phase{{Name: "preview", Enabled: true, Run: func(io.Writer) error {
							previewRan = true

							return nil
						}}},
					},
				},
			},
		},
	}

	m := New(workflows)
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)
	model, _ = m.Update(key(tea.KeyEnter))
	m = model.(Model)

	if m.screen != "confirm" {
		t.Fatalf("screen = %q, want confirm", m.screen)
	}

	model, cmd := m.Update(key(tea.KeyEnter))
	m = model.(Model)
	msg := cmd().(confirmationDoneMsg)
	model, cmd = m.Update(msg)
	m = model.(Model)

	if cmd == nil || !m.running || m.workflows[0].Phases[0].Name != "preview" {
		t.Fatalf("expected preview phase to run, running = %v phase = %#v cmd nil = %v", m.running, m.workflows[0].Phases[0], cmd == nil)
	}

	msg2 := cmd().(phaseDoneMsg)
	model, _ = m.Update(msg2)
	m = model.(Model)

	if !previewRan || liveRan {
		t.Fatalf("previewRan = %v liveRan = %v", previewRan, liveRan)
	}
}

func TestConfirmationBackReturnsHome(t *testing.T) {
	workflows := []Workflow{
		{
			Name: "Set Up This Mac",
			Confirmation: &Confirmation{
				Title:   "Set Up This Mac",
				Message: "Choose how to proceed.",
				Options: []ConfirmationOption{{Label: "Back", Back: true}},
			},
			Phases: []Phase{{Name: "one", Enabled: true, Run: func(io.Writer) error { return nil }}},
		},
	}

	m := New(workflows)
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)
	model, _ = m.Update(key(tea.KeyEnter))
	m = model.(Model)
	model, cmd := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	if cmd != nil || m.screen != "home" {
		t.Fatalf("screen = %q cmd nil = %v, want home and no command", m.screen, cmd == nil)
	}
}

func TestConfirmationViewExplainsWorkflow(t *testing.T) {
	m := New([]Workflow{
		{
			Name:        "Set Up This Mac",
			Description: "Run setup.",
			ChangesMac:  "Yes",
			Confirmation: &Confirmation{
				Title:   "Set Up This Mac",
				Message: "Choose how to proceed.",
				Options: []ConfirmationOption{{Label: "Preview only", Continue: true}},
			},
			Phases: []Phase{{Name: "Check/install prerequisites", Enabled: true, Run: func(io.Writer) error { return nil }}},
		},
	})
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)
	model, _ = m.Update(key(tea.KeyEnter))
	m = model.(Model)

	view := stripANSI(viewString(m))

	for _, want := range []string{"Changes this Mac: Yes", "Steps", "Check/install prerequisites", "Preview only"} {
		if !strings.Contains(view, want) {
			t.Fatalf("confirmation view missing %q:\n%s", want, view)
		}
	}
}

func TestWorkflowViewUsesPlainPhaseStates(t *testing.T) {
	m := New(testWorkflow(nil))
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	view := stripANSI(viewString(m))

	if !strings.Contains(view, "will run") {
		t.Fatalf("workflow view missing will run:\n%s", view)
	}

	model, _ = m.Update(key(tea.KeySpace))
	m = model.(Model)
	view = stripANSI(viewString(m))

	if !strings.Contains(view, "skipped") {
		t.Fatalf("workflow view missing skipped:\n%s", view)
	}
}

func TestInitialMenuStateIncludesBothWorkflows(t *testing.T) {
	m := New(testWorkflow(nil))

	if !strings.Contains(viewString(m), "Set Up This Mac") || !strings.Contains(viewString(m), "Check Setup") {
		t.Fatalf("initial view missing workflows:\n%s", viewString(m))
	}
}

func TestNavigationAndSelection(t *testing.T) {
	model, _ := New(testWorkflow(nil)).Update(key(tea.KeyDown))
	m := model.(Model)

	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}

	model, _ = m.Update(key(tea.KeyEnter))
	m = model.(Model)

	if m.screen != "workflow" {
		t.Fatalf("screen = %q, want workflow", m.screen)
	}
}

func TestPhaseToggling(t *testing.T) {
	m := New(testWorkflow(nil))
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	model, _ = m.Update(key(tea.KeySpace))
	m = model.(Model)

	if m.workflows[0].Phases[0].Enabled {
		t.Fatal("expected first phase disabled")
	}
}

func TestCommandSuccessHandling(t *testing.T) {
	m := New(testWorkflow(nil))
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	model, cmd := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	if !m.running {
		t.Fatal("expected running state")
	}

	msg := cmd().(phaseDoneMsg)
	model, cmd = m.Update(msg)
	m = model.(Model)

	if m.workflows[0].Phases[0].Status != "ok" {
		t.Fatalf("phase status = %q", m.workflows[0].Phases[0].Status)
	}

	msg = cmd().(phaseDoneMsg)
	model, _ = m.Update(msg)
	m = model.(Model)

	if m.exitCode != 0 || m.running {
		t.Fatalf("exitCode = %d running = %v", m.exitCode, m.running)
	}
}

func TestRunViewShowsNumberedCurrentStep(t *testing.T) {
	m := New(testWorkflow(nil))
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	model, _ = m.Update(key(tea.KeyEnter))
	m = model.(Model)

	view := stripANSI(viewString(m))

	for _, want := range []string{"Step 1/2: one", "1/2", "[RUN]", "2/2", "[WAIT]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("run view missing %q:\n%s", want, view)
		}
	}
}

func TestCommandFailureStopsRun(t *testing.T) {
	m := New(testWorkflow(errors.New("boom")))
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	model, cmd := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	msg := cmd().(phaseDoneMsg)
	model, next := m.Update(msg)
	m = model.(Model)

	if next != nil {
		t.Fatal("expected run to stop after failure")
	}

	if m.exitCode != 1 || m.err == nil {
		t.Fatalf("exitCode = %d err = %v", m.exitCode, m.err)
	}
}

func TestQuitBeforeRun(t *testing.T) {
	m := New(testWorkflow(nil))
	model, cmd := m.Update(key('q'))
	m = model.(Model)

	if m.exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", m.exitCode)
	}

	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestCancelWhileRunningReturnsNonZero(t *testing.T) {
	m := New(testWorkflow(nil))
	m.running = true

	model, cmd := m.Update(ctrl('c'))
	m = model.(Model)

	if m.exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", m.exitCode)
	}

	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestConfirmationProceedStartsWorkflow(t *testing.T) {
	var confirmationLog bytes.Buffer
	workflows := []Workflow{
		{
			Name: "Factory Install",
			Confirmation: &Confirmation{
				Title:   "Confirm erase state",
				Message: "Choose how to proceed.",
				Options: []ConfirmationOption{
					{
						Label:    "Already erased",
						Continue: true,
						Run: func(w io.Writer) error {
							_, _ = confirmationLog.WriteString("confirmed")
							_, _ = w.Write([]byte("confirmed: already erased\n"))

							return nil
						},
					},
				},
			},
			Phases: []Phase{{Name: "one", Enabled: true, Run: func(io.Writer) error { return nil }}},
		},
	}

	m := New(workflows)
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	model, _ = m.Update(key(tea.KeyEnter))
	m = model.(Model)

	if m.screen != "confirm" {
		t.Fatalf("screen = %q, want confirm", m.screen)
	}

	model, cmd := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	msg := cmd().(confirmationDoneMsg)
	model, next := m.Update(msg)
	m = model.(Model)

	if confirmationLog.String() != "confirmed" {
		t.Fatal("expected confirmation callback")
	}

	if !strings.Contains(m.log, "confirmed: already erased") {
		t.Fatalf("log = %q", m.log)
	}

	if next == nil || !m.running || m.phase != 0 {
		t.Fatalf("expected workflow to start, running = %v phase = %d next nil = %v", m.running, m.phase, next == nil)
	}
}

func TestConfirmationStopDoesNotRunWorkflow(t *testing.T) {
	ran := false
	workflows := []Workflow{
		{
			Name: "Factory Install",
			Confirmation: &Confirmation{
				Title:   "Confirm erase state",
				Message: "Choose how to proceed.",
				Options: []ConfirmationOption{
					{
						Label:    "Erase first",
						Continue: false,
						Run: func(w io.Writer) error {
							_, _ = w.Write([]byte("opening reset settings\n"))

							return nil
						},
					},
				},
			},
			Phases: []Phase{{Name: "one", Enabled: true, Run: func(io.Writer) error {
				ran = true

				return nil
			}}},
		},
	}

	m := New(workflows)
	model, _ := m.Update(key(tea.KeyEnter))
	m = model.(Model)
	model, _ = m.Update(key(tea.KeyEnter))
	m = model.(Model)
	model, cmd := m.Update(key(tea.KeyEnter))
	m = model.(Model)

	msg := cmd().(confirmationDoneMsg)
	model, next := m.Update(msg)
	m = model.(Model)

	if next != nil || m.running || ran {
		t.Fatalf("expected stop before phases, next nil = %v running = %v ran = %v", next == nil, m.running, ran)
	}

	if m.exitCode != 0 || !strings.Contains(m.log, "opening reset settings") {
		t.Fatalf("exitCode = %d log = %q", m.exitCode, m.log)
	}
}
