package tui

import tea "charm.land/bubbletea/v2"

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
		return m.updateHomeKey(key)
	case "workflow":
		return m.updateWorkflowKey(key)
	case "confirm":
		return m.updateConfirmKey(key)
	case "run":
		return m.updateRunKey(key)
	}

	return m, nil
}

func (m Model) updateHomeKey(key tea.Key) (tea.Model, tea.Cmd) {
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

	return m, nil
}

func (m Model) updateWorkflowKey(key tea.Key) (tea.Model, tea.Cmd) {
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
		if workflow.Confirmation != nil {
			m.screen = "confirm"
			m.choice = 0

			return m, nil
		}

		return m.startRun()
	case key.Code == tea.KeyBackspace:
		m.screen = "home"
		m.phase = 0
	}

	return m, nil
}

func (m Model) updateConfirmKey(key tea.Key) (tea.Model, tea.Cmd) {
	confirmation := m.workflows[m.cursor].Confirmation

	if confirmation == nil || len(confirmation.Options) == 0 {
		return m.startRun()
	}

	switch {
	case key.Code == tea.KeyUp || key.Code == 'k':
		if m.choice > 0 {
			m.choice--
		}
	case key.Code == tea.KeyDown || key.Code == 'j':
		if m.choice < len(confirmation.Options)-1 {
			m.choice++
		}
	case key.Code == tea.KeyBackspace:
		m.screen = "workflow"
		m.choice = 0
	case key.Code == tea.KeyEnter:
		option := confirmation.Options[m.choice]

		if option.Back {
			m.screen = "home"
			m.choice = 0

			return m, nil
		}

		if option.Phases != nil {
			m.workflows[m.cursor].Phases = option.Phases
		}

		m.screen = "run"
		m.log = ""
		m.phase = -1
		m.message = ""
		m.err = nil
		m.exitCode = 0
		m.running = true

		return m, m.runConfirmationOption(option)
	}

	return m, nil
}

func (m Model) updateRunKey(key tea.Key) (tea.Model, tea.Cmd) {
	if key.Code == tea.KeyEnter && !m.running {
		return m, tea.Quit
	}

	return m, nil
}
