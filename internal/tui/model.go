package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dackota/kubecorr/internal/timeline"
)

type pane int

const (
	focusLogs pane = iota
	focusEvents
)

const (
	logPaneShare = 0.6 // logs take 60% of the width
	chromeRows   = 4   // header, top border, bottom border, footer
	pageStep     = 10
)

// Model is the bubbletea state. It is a value type: every update returns a
// new copy and never changes the old one.
type Model struct {
	logs, events []timeline.Item
	namespace    string
	context      string

	focus       pane
	logCursor   int
	eventCursor int
	wrap        bool
	width       int
	height      int
}

// New builds a model from already collected items.
func New(logs, events []timeline.Item, namespace, context string) Model {
	return Model{logs: logs, events: events, namespace: namespace, context: context}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return update(m, msg) }

// update is the typed form of Update so tests do not need type assertions.
func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return handleKey(m, msg)
	}
	return m, nil
}

func handleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "tab":
		m.focus = 1 - m.focus
	case "w":
		m.wrap = !m.wrap
	case "j", "down":
		m = move(m, 1)
	case "k", "up":
		m = move(m, -1)
	case "pgdown", "ctrl+d":
		m = move(m, pageStep)
	case "pgup", "ctrl+u":
		m = move(m, -pageStep)
	case "g", "home":
		m = move(m, -1<<30)
	case "G", "end":
		m = move(m, 1<<30)
	}
	return m, nil
}

// move shifts the focused cursor by delta and re-links the other pane by time.
func move(m Model, delta int) Model {
	if m.focus == focusLogs {
		m.logCursor = clamp(m.logCursor+delta, len(m.logs))
		if len(m.logs) > 0 {
			m.eventCursor = nearestAtOrBefore(m.events, m.logs[m.logCursor].Time)
		}
		return m
	}
	m.eventCursor = clamp(m.eventCursor+delta, len(m.events))
	if len(m.events) > 0 {
		m.logCursor = nearestAtOrBefore(m.logs, m.events[m.eventCursor].Time)
	}
	return m
}
