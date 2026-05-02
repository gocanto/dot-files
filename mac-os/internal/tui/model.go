package tui

import (
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
	Name         string
	Description  string
	ChangesMac   string
	Phases       []Phase
	Confirmation *Confirmation
}

type Confirmation struct {
	Title   string
	Message string
	Options []ConfirmationOption
}

type ConfirmationOption struct {
	Label       string
	Description string
	Continue    bool
	Back        bool
	Phases      []Phase
	Run         func(io.Writer) error
}

type Result struct {
	ExitCode int
}

type Model struct {
	workflows []Workflow
	screen    string
	cursor    int
	phase     int
	choice    int
	log       string
	running   bool
	err       error
	exitCode  int
	message   string
	execPhase bool
}

type phaseDoneMsg struct {
	output string
	err    error
}

type confirmationDoneMsg struct {
	output  string
	proceed bool
	message string
	err     error
}

func New(workflows []Workflow) Model {
	return Model{
		workflows: workflows,
		screen:    "home",
	}
}

func Run(in io.Reader, out io.Writer, workflows []Workflow) (Result, error) {
	initial := New(workflows)
	initial.execPhase = true

	model, err := tea.NewProgram(initial, tea.WithInput(in), tea.WithOutput(out)).Run()

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
	case confirmationDoneMsg:
		return m.updateConfirmationDone(msg)
	case phaseDoneMsg:
		return m.updatePhaseDone(msg)
	}

	return m, nil
}
