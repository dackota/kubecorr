package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dackota/kubecorr/internal/summary"
	"github.com/dackota/kubecorr/internal/timeline"
)

const refreshEvery = 3 * time.Second

// RefreshFunc rebuilds the pod summaries. It gets the current events so it
// can count probe failures without another API call.
type RefreshFunc func(events []timeline.Item) []summary.PodSummary

type refreshTickMsg struct{}

type summariesMsg []summary.PodSummary

// WithRefresh makes the header reload on a timer. Returns a new model.
func (m Model) WithRefresh(f RefreshFunc) Model {
	m.refresh = f
	return m
}

func scheduleRefresh(m Model) tea.Cmd {
	if m.refresh == nil {
		return nil
	}
	return tea.Tick(refreshEvery, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

// runRefresh calls the refresh function off the main loop.
func runRefresh(m Model) tea.Cmd {
	f, events := m.refresh, m.events
	return func() tea.Msg { return summariesMsg(f(events)) }
}
