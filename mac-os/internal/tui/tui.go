package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

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

const (
	reset      = "\x1b[0m"
	bold       = "\x1b[1m"
	fgText     = "\x1b[38;2;218;224;244m"
	fgMuted    = "\x1b[38;2;126;135;158m"
	fgAccent   = "\x1b[38;2;118;214;196m"
	fgSuccess  = "\x1b[38;2;137;220;142m"
	fgDanger   = "\x1b[38;2;239;118;122m"
	bgSelected = "\x1b[48;2;31;42;50m"
	leftPad    = "   "
	rowWidth   = 76
	footerLine = 16
	logLines   = 8
)

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
	var content string

	switch m.screen {
	case "workflow":
		content = m.workflowView()
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
		phaseCount := enabledPhaseCount(workflow)
		summary := fmt.Sprintf("%d/%d phases enabled", phaseCount, len(workflow.Phases))
		row := menuRow("  ", workflow.Name, summary, false)

		if i == m.cursor {
			row = menuRow(">", workflow.Name, summary, true)
		}

		fmt.Fprintln(&b, inset(row))
	}

	padToLine(&b, footerLine)
	fmt.Fprintln(&b, inset(help("enter", "open", "j/k", "move", "q/esc", "quit")))

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
	fmt.Fprintln(&b)

	for _, phase := range workflow.Phases {
		status := phase.Status

		if status == "" {
			status = "pending"
		}

		fmt.Fprintln(&b, inset(statusRow(phase.Name, status)))
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

func banner(title string) string {
	title = strings.ToUpper(title)
	rule := strings.Repeat("-", max(16, rowWidth-len(title)-2))

	return accent(title) + " " + muted(rule)
}

func menuRow(marker, label, meta string, selected bool) string {
	contentWidth := rowWidth - 4
	labelWidth := 38

	if len(label) > labelWidth {
		labelWidth = len(label)
	}

	if !selected {
		meta = muted(meta)
	}

	row := fmt.Sprintf(" %s %-*s %s", marker, labelWidth, label, meta)
	row = padRight(row, contentWidth)

	if selected {
		return bgSelected + accent(row) + reset
	}

	return fgText + row + reset
}

func statusRow(label, status string) string {
	row := fmt.Sprintf("  %-42s %s", label, statusBadge(status))

	return padRight(row, rowWidth)
}

func help(parts ...string) string {
	var b strings.Builder

	for i := 0; i+1 < len(parts); i += 2 {
		if i > 0 {
			b.WriteString("  ")
		}

		b.WriteString(accent(parts[i]))
		b.WriteString(": ")
		b.WriteString(parts[i+1])
	}

	return b.String()
}

func enabledPhaseCount(workflow Workflow) int {
	count := 0

	for _, phase := range workflow.Phases {
		if phase.Enabled {
			count++
		}
	}

	return count
}

func phaseState(phase Phase) string {
	if phase.Enabled {
		return "armed"
	}

	return "off"
}

func runSummary(workflow Workflow, running bool, err error) string {
	if err != nil {
		return danger("Run stopped on the first failing phase.")
	}

	if running {
		return accent("Running enabled phases now.")
	}

	return success(fmt.Sprintf("Complete: %d phases processed.", len(workflow.Phases)))
}

func statusBadge(status string) string {
	switch status {
	case "ok":
		return success("[OK]")
	case "failed":
		return danger("[FAIL]")
	case "running":
		return accent("[RUN]")
	case "skipped":
		return muted("[SKIP]")
	default:
		return muted("[WAIT]")
	}
}

func accent(s string) string {
	return fgAccent + bold + s + reset
}

func success(s string) string {
	return fgSuccess + bold + s + reset
}

func danger(s string) string {
	return fgDanger + bold + s + reset
}

func muted(s string) string {
	return fgMuted + s + reset
}

func inset(s string) string {
	return leftPad + s
}

func indentBlock(s string) string {
	lines := strings.SplitAfter(s, "\n")

	var b strings.Builder

	for _, line := range lines {
		if line == "" {
			continue
		}

		b.WriteString(leftPad)
		b.WriteString("  ")
		b.WriteString(line)
	}

	return b.String()
}

func outputPanel(log string, height int) string {
	lines := tailLines(log, height)

	var b strings.Builder

	for i := 0; i < height; i++ {
		line := ""

		if i < len(lines) {
			line = lines[i]
		}

		b.WriteString(leftPad)
		b.WriteString("  ")
		b.WriteString(muted(padRight(line, rowWidth-4)))
		b.WriteString("\n")
	}

	return b.String()
}

func tailLines(log string, limit int) []string {
	if log == "" {
		return nil
	}

	lines := strings.Split(strings.TrimRight(log, "\n"), "\n")

	if len(lines) <= limit {
		return lines
	}

	return lines[len(lines)-limit:]
}

func padToLine(b *bytes.Buffer, target int) {
	for strings.Count(b.String(), "\n") < target {
		fmt.Fprintln(b)
	}
}

func padRight(s string, width int) string {
	plain := stripANSI(s)

	if len(plain) >= width {
		return s
	}

	return s + strings.Repeat(" ", width-len(plain))
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func stripANSI(s string) string {
	replacer := strings.NewReplacer(reset, "", bold, "", fgText, "", fgMuted, "", fgAccent, "", fgSuccess, "", fgDanger, "", bgSelected, "")

	return replacer.Replace(s)
}
