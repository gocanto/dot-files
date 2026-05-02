package tui

import (
	"bytes"
	"fmt"
	"strings"
)

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
