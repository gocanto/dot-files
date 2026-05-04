package tui

import (
	"bytes"
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	var content string

	switch m.screen {
	case "workflow":
		content = m.workflowView()
	case "confirm":
		content = m.confirmView()
	case "run":
		content = m.runView()
	default:
		content = m.homeView()
	}

	view := tea.NewView(content)
	view.AltScreen = true

	return view
}

func (m Model) homeView() string {
	var b bytes.Buffer

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, inset(banner("mac-os")))
	fmt.Fprintln(&b, inset(muted(fmt.Sprintf("%d workflows ready", len(m.workflows)))))
	fmt.Fprintln(&b)

	for i, workflow := range m.workflows {
		summary := workflow.ChangesMac

		if summary == "" {
			phaseCount := enabledPhaseCount(workflow)
			summary = fmt.Sprintf("%d/%d phases enabled", phaseCount, len(workflow.Phases))
		}

		row := menuRow("  ", workflow.Name, summary, false)

		if i == m.cursor {
			row = menuRow(">", workflow.Name, summary, true)
		}

		fmt.Fprintln(&b, inset(row))
	}

	if len(m.workflows) > 0 {
		fmt.Fprintln(&b)

		for _, line := range wrapLines(m.workflows[m.cursor].Description, rowWidth) {
			fmt.Fprintln(&b, inset(muted(line)))
		}
	}

	padToLine(&b, footerLine)
	fmt.Fprintln(&b, inset(help("enter", "open", "j/k", "move", "q/esc", "quit")))

	return b.String()
}

func (m Model) confirmView() string {
	var b bytes.Buffer
	workflow := m.workflows[m.cursor]
	confirmation := workflow.Confirmation

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, inset(banner(workflow.Name)))

	if confirmation == nil {
		fmt.Fprintln(&b, inset(muted("No confirmation is required.")))

		return b.String()
	}

	title := confirmation.Title

	if title == "" {
		title = "Confirm before running"
	}

	fmt.Fprintln(&b, inset(accent(title)))

	for _, line := range wrapLines(confirmation.Message, rowWidth-6) {
		fmt.Fprintln(&b, inset(muted(line)))
	}

	if workflow.ChangesMac != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, inset(muted("Changes this Mac: ")+accent(workflow.ChangesMac)))
	}

	if len(workflow.Phases) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, inset(accent("Steps")))

		for i, phase := range workflow.Phases {
			fmt.Fprintln(&b, inset(muted(fmt.Sprintf("%2d. %s", i+1, phase.Name))))
		}
	}

	fmt.Fprintln(&b)

	for i, option := range confirmation.Options {
		label := option.Label
		meta := option.Description
		row := menuRow("  ", label, meta, false)

		if i == m.choice {
			row = menuRow(">", label, meta, true)
		}

		fmt.Fprintln(&b, inset(row))
	}

	padToLine(&b, footerLine)
	fmt.Fprintln(&b, inset(help("enter", "select", "j/k", "move", "backspace", "workflow", "q/esc", "quit")))

	return b.String()
}

func (m Model) workflowView() string {
	var b bytes.Buffer
	workflow := m.workflows[m.cursor]
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, inset(banner(workflow.Name)))
	fmt.Fprintln(&b, inset(muted("Toggle phases, then launch the selected workflow.")))
	fmt.Fprintln(&b)

	for i, phase := range workflow.Phases {
		checked := "[ ]"

		if phase.Enabled {
			checked = "[x]"
		}

		label := fmt.Sprintf("%s %s", checked, phase.Name)
		row := menuRow("  ", label, stripANSI(phaseState(phase)), false)

		if i == m.phase {
			row = menuRow(">", label, stripANSI(phaseState(phase)), true)
		}

		fmt.Fprintln(&b, inset(row))
	}

	padToLine(&b, footerLine)
	fmt.Fprintln(&b, inset(help("space", "toggle", "enter", "run", "backspace", "home", "q/esc", "quit")))

	return b.String()
}

func (m Model) runView() string {
	var b bytes.Buffer
	workflow := m.workflows[m.cursor]
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, inset(banner(workflow.Name)))
	fmt.Fprintln(&b, inset(runSummary(workflow, m.running, m.err)))
	fmt.Fprintln(&b, inset(currentStep(workflow, m.phase, m.running, m.message)))
	fmt.Fprintln(&b)

	for i, phase := range workflow.Phases {
		status := phase.Status

		if status == "" {
			status = "pending"
		}

		fmt.Fprintln(&b, inset(statusRow(i+1, len(workflow.Phases), phase.Name, status)))
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, inset(accent("output")))
	fmt.Fprint(&b, outputPanel(m.log, logLines))

	if m.err != nil {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, inset(danger(fmt.Sprintf("failed: %v", m.err))))
	}

	if !m.running {
		padToLine(&b, footerLine+logLines+3)
		fmt.Fprintln(&b, inset(help("enter", "exit")))
	}

	return b.String()
}
