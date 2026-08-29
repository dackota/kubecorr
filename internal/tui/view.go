package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dackota/kubecorr/internal/timeline"
)

const timeLayout = "15:04:05.000"

var (
	styleHeader  = lipgloss.NewStyle().Bold(true)
	styleHint    = lipgloss.NewStyle().Faint(true)
	styleCursor  = lipgloss.NewStyle().Reverse(true)
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleNormal  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleFocus   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("2"))
	styleBlur    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8"))
	podColors    = []string{"2", "5", "4", "1", "10", "13", "12", "9"}
)

// View satisfies tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	header := styleHeader.Render(fmt.Sprintf(" kubecorr  ns=%s  ctx=%s", m.namespace, m.context))
	footer := styleHint.Render(" [q] quit  [tab] focus  [j/k] scroll  [g/G] top/end  [w] wrap")

	bodyHeight := m.height - chromeRows
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	logWidth := int(float64(m.width) * logPaneShare)
	eventWidth := m.width - logWidth

	left := m.pane("Logs", m.logs, m.logCursor, m.focus == focusLogs, logWidth, bodyHeight, m.logLine)
	right := m.pane("Events", m.events, m.eventCursor, m.focus == focusEvents, eventWidth, bodyHeight, m.eventLine)

	return header + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n" + footer
}

type lineFn func(timeline.Item, int) string

func (m Model) pane(title string, items []timeline.Item, cursor int, focused bool, width, height int, render lineFn) string {
	inner := width - 2 // borders
	if inner < 1 {
		inner = 1
	}
	title = fmt.Sprintf(" %s (%d) ", title, len(items))
	rows := make([]string, 0, height)
	start := windowStart(cursor, len(items), height)
	for i := start; i < len(items) && len(rows) < height; i++ {
		line := render(items[i], inner)
		if i == cursor {
			line = styleCursor.Render(padRight(stripANSI(line), inner))
		}
		rows = append(rows, line)
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	body := strings.Join(rows, "\n")
	style := styleBlur
	if focused {
		style = styleFocus
	}
	box := style.Width(inner).Height(height).Render(body)
	return placeTitle(box, title)
}

func (m Model) logLine(it timeline.Item, width int) string {
	src := lipgloss.NewStyle().Foreground(lipgloss.Color(podColor(it.Source))).Render(it.Source)
	plain := fmt.Sprintf("%s %s  %s", it.Time.Local().Format(timeLayout), it.Source, it.Text)
	if !m.wrap && lipgloss.Width(plain) > width {
		cut := width - lipgloss.Width(it.Time.Local().Format(timeLayout)) - lipgloss.Width(it.Source) - 3
		if cut < 0 {
			cut = 0
		}
		return fmt.Sprintf("%s %s  %s", styleHint.Render(it.Time.Local().Format(timeLayout)), src, truncate(it.Text, cut))
	}
	return fmt.Sprintf("%s %s  %s", styleHint.Render(it.Time.Local().Format(timeLayout)), src, it.Text)
}

func (m Model) eventLine(it timeline.Item, width int) string {
	style := styleNormal
	if it.Type == "Warning" {
		style = styleWarning
	}
	plain := fmt.Sprintf("%s %-7s %-12s %s  %s", it.Time.Local().Format("15:04:05"), it.Type, it.Reason, it.Source, it.Text)
	if !m.wrap {
		plain = truncate(plain, width)
	}
	return style.Render(plain)
}

func podColor(source string) string {
	pod, _, _ := strings.Cut(source, "/")
	var h uint32 = 2166136261
	for i := 0; i < len(pod); i++ {
		h ^= uint32(pod[i])
		h *= 16777619
	}
	return podColors[h%uint32(len(podColors))]
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	if width <= 1 {
		return string(r[:width])
	}
	return string(r[:width-1]) + "…"
}

func padRight(s string, width int) string {
	if n := width - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// stripANSI drops escape codes so a reversed cursor row has one flat style.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// placeTitle writes the title into the top border line of a box.
func placeTitle(box, title string) string {
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}
	top := []rune(stripANSI(lines[0]))
	t := []rune(title)
	if len(t)+2 > len(top) {
		return box
	}
	copy(top[2:], t)
	lines[0] = string(top)
	return strings.Join(lines, "\n")
}
