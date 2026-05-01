package tui

import (
	"bytes"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
)

type Phase struct {
	Name    string
	Run     func(io.Writer) error
	Enabled bool
	Status  string
}

type Workflow struct {
	Name   string
	Phases []Phase
}

type Result struct {
	ExitCode int
}

type Model struct {
	workflows []Workflow
	screen    string
	cursor    int
	phase     int
	log       string
	running   bool
	err       error
	exitCode  int
}

type phaseDoneMsg struct {
	output string
	err    error
}

func New(workflows []Workflow) Model {
	return Model{
		workflows: workflows,
		screen:    "home",
	}
}

func Run(in io.Reader, out io.Writer, workflows []Workflow) (Result, error) {
	model, err := tea.NewProgram(New(workflows), tea.WithInput(in), tea.WithOutput(out)).Run()

	if err != nil {
		return Result{ExitCode: 1}, err
	}

	m, ok := model.(Model)

	if !ok {
		return Result{ExitCode: 1}, fmt.Errorf("unexpected TUI model %T", model)
	}

	return Result{ExitCode: m.exitCode}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case phaseDoneMsg:
		return m.updatePhaseDone(msg)
	}

	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()

	if key.Code == 'c' && key.Mod == tea.ModCtrl {
		if m.running {
			m.exitCode = 1
		}

		return m, tea.Quit
	}

	switch {
	case key.Code == 'q' || key.Code == tea.KeyEscape:
		if m.running {
			return m, nil
		}

		return m, tea.Quit
	}

	if m.running {
		return m, nil
	}

	switch m.screen {
	case "home":
		switch {
		case key.Code == tea.KeyUp || key.Code == 'k':
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Code == tea.KeyDown || key.Code == 'j':
			if m.cursor < len(m.workflows)-1 {
				m.cursor++
			}
		case key.Code == tea.KeyEnter:
			m.screen = "workflow"
			m.phase = 0
		}
	case "workflow":
		workflow := m.workflows[m.cursor]

		switch {
		case key.Code == tea.KeyUp || key.Code == 'k':
			if m.phase > 0 {
				m.phase--
			}
		case key.Code == tea.KeyDown || key.Code == 'j':
			if m.phase < len(workflow.Phases)-1 {
				m.phase++
			}
		case key.Code == tea.KeySpace:
			m.workflows[m.cursor].Phases[m.phase].Enabled = !m.workflows[m.cursor].Phases[m.phase].Enabled
		case key.Code == tea.KeyEnter:
			m.screen = "run"
			m.log = ""
			m.phase = -1

			return m.startNextPhase()
		case key.Code == tea.KeyBackspace:
			m.screen = "home"
			m.phase = 0
		}
	case "run":
		if key.Code == tea.KeyEnter && !m.running {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) updatePhaseDone(msg phaseDoneMsg) (tea.Model, tea.Cmd) {
	m.running = false
	m.log += msg.output

	if msg.err != nil {
		m.err = msg.err
		m.exitCode = 1
		m.workflows[m.cursor].Phases[m.phase].Status = "failed"

		return m, nil
	}

	m.workflows[m.cursor].Phases[m.phase].Status = "ok"

	return m.startNextPhase()
}

func (m Model) startNextPhase() (tea.Model, tea.Cmd) {
	phases := m.workflows[m.cursor].Phases

	for i := m.phase + 1; i < len(phases); i++ {
		if !phases[i].Enabled {
			m.workflows[m.cursor].Phases[i].Status = "skipped"

			continue
		}

		m.phase = i
		m.running = true
		m.workflows[m.cursor].Phases[i].Status = "running"

		phase := m.workflows[m.cursor].Phases[i]

		return m, func() tea.Msg {
			var b bytes.Buffer
			err := phase.Run(&b)

			return phaseDoneMsg{output: b.String(), err: err}
		}
	}

	m.exitCode = 0

	return m, nil
}

func (m Model) View() tea.View {
	switch m.screen {
	case "workflow":
		return tea.NewView(m.workflowView())
	case "run":
		return tea.NewView(m.runView())
	default:
		return tea.NewView(m.homeView())
	}
}

func (m Model) homeView() string {
	var b bytes.Buffer

	fmt.Fprintln(&b, "mac-os")
	fmt.Fprintln(&b)

	for i, workflow := range m.workflows {
		cursor := " "

		if i == m.cursor {
			cursor = ">"
		}

		fmt.Fprintf(&b, "%s %s\n", cursor, workflow.Name)
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "enter: open  q/esc: quit")

	return b.String()
}

func (m Model) workflowView() string {
	var b bytes.Buffer
	workflow := m.workflows[m.cursor]
	fmt.Fprintln(&b, workflow.Name)
	fmt.Fprintln(&b)

	for i, phase := range workflow.Phases {
		cursor := " "

		if i == m.phase {
			cursor = ">"
		}

		checked := " "

		if phase.Enabled {
			checked = "x"
		}

		fmt.Fprintf(&b, "%s [%s] %s\n", cursor, checked, phase.Name)
	}

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "space: toggle  enter: run  backspace: home  q/esc: quit")

	return b.String()
}

func (m Model) runView() string {
	var b bytes.Buffer
	workflow := m.workflows[m.cursor]
	fmt.Fprintln(&b, workflow.Name)
	fmt.Fprintln(&b)

	for _, phase := range workflow.Phases {
		status := phase.Status

		if status == "" {
			status = "pending"
		}

		fmt.Fprintf(&b, "%-24s %s\n", phase.Name, status)
	}

	if m.log != "" {
		fmt.Fprintln(&b)
		fmt.Fprint(&b, m.log)
	}

	if m.err != nil {
		fmt.Fprintf(&b, "\nfailed: %v\n", m.err)
	}

	if !m.running {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "enter: exit")
	}

	return b.String()
}
