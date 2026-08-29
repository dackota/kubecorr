package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dackota/kubecorr/internal/summary"
)

var (
	styleRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

// renderSummary draws the pod summaries as a small colored table.
// Pod names use the same color as their log lines.
func renderSummary(pods []summary.PodSummary, width int) string {
	if len(pods) == 0 {
		return ""
	}
	nameWidth := 0
	for _, p := range pods {
		if n := lipgloss.Width(p.Pod); n > nameWidth {
			nameWidth = n
		}
	}
	var b strings.Builder
	for _, p := range pods {
		color := lipgloss.NewStyle().Foreground(lipgloss.Color(podColor(p.Pod))).Bold(true)
		fmt.Fprintf(&b, " %s %s   %s   %s\n",
			color.Render("●"), color.Render(padRight(p.Pod, nameWidth)),
			styleHint.Render("node "+p.Node), phaseStyle(p.Phase).Render(p.Phase))
		for _, c := range p.Containers {
			b.WriteString(containerLine(c) + "\n")
		}
		if len(p.Probes) > 0 {
			b.WriteString(probeLine(p.Probes) + "\n")
		}
	}
	return truncateLines(b.String(), width)
}

func containerLine(c summary.ContainerSummary) string {
	state := strings.TrimPrefix(strings.TrimPrefix(c.State, "Waiting: "), "Terminated: ")
	stateStyle := styleOK
	if !c.Ready {
		stateStyle = styleRed
	}
	line := fmt.Sprintf("     %-9s %s", c.Name, stateStyle.Render(padRight(state, 22)))
	restarts := fmt.Sprintf("restarts %d", c.Restarts)
	if c.Restarts > 0 {
		line += "   " + styleRed.Render(restarts)
	} else {
		line += "   " + styleHint.Render(restarts)
	}
	if c.LastExitReason != "" || c.LastExitCode != 0 {
		line += "   " + styleRed.Render(fmt.Sprintf("exit %s (%d)", c.LastExitReason, c.LastExitCode))
	}
	return line
}

func probeLine(probes []summary.ProbeSummary) string {
	parts := make([]string, 0, len(probes))
	for _, p := range probes {
		parts = append(parts, fmt.Sprintf("%s %dx (%s)", p.Probe, p.Failures, p.Last.Local().Format("15:04:05")))
	}
	return fmt.Sprintf("     %-9s %s", "probes", styleYellow.Render(strings.Join(parts, "   ")))
}

func phaseStyle(phase string) lipgloss.Style {
	if phase == "Running" || phase == "Succeeded" {
		return styleOK
	}
	return styleRed
}

// truncateLines cuts every line to width so the header never wraps.
func truncateLines(s string, width int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		if lipgloss.Width(l) > width {
			lines[i] = truncate(l, width)
		}
	}
	return strings.Join(lines, "\n")
}
