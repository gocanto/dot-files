package tui

import (
	"bytes"
	"io"

	tea "charm.land/bubbletea/v2"
)

type writerCommand struct {
	run    func(io.Writer) error
	output bytes.Buffer
}

func (m Model) runConfirmationOption(option ConfirmationOption) tea.Cmd {
	run := func(w io.Writer) error {
		if option.Run == nil {
			return nil
		}

		return option.Run(w)
	}

	if m.execPhase {
		cmd := &writerCommand{run: run}

		return tea.Exec(cmd, func(err error) tea.Msg {
			return confirmationDoneMsg{
				output:  cmd.output.String(),
				proceed: option.Continue,
				message: option.Label,
				err:     err,
			}
		})
	}

	return func() tea.Msg {
		var b bytes.Buffer
		err := run(&b)

		return confirmationDoneMsg{
			output:  b.String(),
			proceed: option.Continue,
			message: option.Label,
			err:     err,
		}
	}
}

func (m Model) startRun() (tea.Model, tea.Cmd) {
	m.screen = "run"
	m.log = ""
	m.phase = -1
	m.choice = 0
	m.message = ""
	m.err = nil
	m.exitCode = 0

	return m.startNextPhase()
}

func (m Model) updateConfirmationDone(msg confirmationDoneMsg) (tea.Model, tea.Cmd) {
	m.running = false
	m.log += msg.output
	m.message = msg.message

	if msg.err != nil {
		m.err = msg.err
		m.exitCode = 1

		return m, nil
	}

	if !msg.proceed {
		m.exitCode = 0
		m.message = "Factory install stopped before install phases."

		return m, nil
	}

	return m.startNextPhase()
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

		return m, m.runPhase(phase)
	}

	m.exitCode = 0

	return m, nil
}

func (m Model) runPhase(phase Phase) tea.Cmd {
	if m.execPhase {
		cmd := &writerCommand{run: phase.Run}

		return tea.Exec(cmd, func(err error) tea.Msg {
			return phaseDoneMsg{output: cmd.output.String(), err: err}
		})
	}

	return func() tea.Msg {
		var b bytes.Buffer
		err := phase.Run(&b)

		return phaseDoneMsg{output: b.String(), err: err}
	}
}

func (c *writerCommand) Run() error {
	return c.run(&c.output)
}

func (c *writerCommand) SetStdin(io.Reader) {}

func (c *writerCommand) SetStdout(io.Writer) {}

func (c *writerCommand) SetStderr(io.Writer) {}
