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

type writerCommand struct {
	run    func(io.Writer) error
	output bytes.Buffer
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
	case "confirm":
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
	case "run":
		if key.Code == tea.KeyEnter && !m.running {
			return m, tea.Quit
		}
	}

	return m, nil
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

func statusRow(index, total int, label, status string) string {
	prefix := fmt.Sprintf("%2d/%-2d", index, total)
	row := fmt.Sprintf("  %s  %-36s %s", prefix, label, statusBadge(status))

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
		return "will run"
	}

	return "skipped"
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

func currentStep(workflow Workflow, phase int, running bool, message string) string {
	if running && phase >= 0 && phase < len(workflow.Phases) {
		return accent(fmt.Sprintf("Step %d/%d: %s", phase+1, len(workflow.Phases), workflow.Phases[phase].Name))
	}

	if message != "" {
		return muted(message)
	}

	return muted("No active step.")
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

func wrapLines(s string, width int) []string {
	if s == "" {
		return nil
	}

	words := strings.Fields(s)

	if len(words) == 0 {
		return nil
	}

	lines := []string{}
	line := words[0]

	for _, word := range words[1:] {
		if len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = word

			continue
		}

		line += " " + word
	}

	return append(lines, line)
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
