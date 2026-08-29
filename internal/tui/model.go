package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dackota/kubecorr/internal/summary"
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
	summaries   []summary.PodSummary
	follow      bool
	stream      <-chan timeline.Item
	streamCtx   context.Context
	refresh     RefreshFunc

	query     string // active filter
	editing   bool   // true while the user types after /
	visLogs   []timeline.Item
	visEvents []timeline.Item
}

// New builds a model from already collected items.
func New(logs, events []timeline.Item, namespace, context string) Model {
	return applyFilter(Model{logs: logs, events: events, namespace: namespace, context: context})
}

// WithSummary sets the pod summaries shown under the header. Returns a new model.
func (m Model) WithSummary(s []summary.PodSummary) Model {
	m.summaries = s
	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(waitForItem(m.streamCtx, m.stream), scheduleRefresh(m))
}

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
	case ItemMsg:
		return addItem(m, timeline.Item(msg)), waitForItem(m.streamCtx, m.stream)
	case streamDoneMsg:
		m.stream = nil
		return m, nil
	case refreshTickMsg:
		return m, runRefresh(m)
	case summariesMsg:
		m.summaries = []summary.PodSummary(msg)
		return m, scheduleRefresh(m)
	}
	return m, nil
}

func handleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.editing {
		return editQuery(m, msg), nil
	}
	switch msg.String() {
	case "/":
		m.editing = true
	case "esc":
		m.query = ""
		m = applyFilter(m)
	case "q", "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.focus = 1 - m.focus
	case "w":
		m.wrap = !m.wrap
	case "f":
		m.follow = !m.follow
		if m.follow {
			m = jumpToEnd(m)
		}
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
// Any manual move turns follow off.
func move(m Model, delta int) Model {
	m.follow = false
	if m.focus == focusLogs {
		m.logCursor = clamp(m.logCursor+delta, len(m.visLogs))
		if len(m.visLogs) > 0 {
			m.eventCursor = nearestAtOrBefore(m.visEvents, m.visLogs[m.logCursor].Time)
		}
		return m
	}
	m.eventCursor = clamp(m.eventCursor+delta, len(m.visEvents))
	if len(m.visEvents) > 0 {
		m.logCursor = nearestAtOrBefore(m.visLogs, m.visEvents[m.eventCursor].Time)
	}
	return m
}

// editQuery handles keys while the user types a filter after "/".
// Enter keeps the filter. Esc clears it. Both leave edit mode.
func editQuery(m Model, msg tea.KeyMsg) Model {
	switch msg.Type {
	case tea.KeyEnter:
		m.editing = false
	case tea.KeyEsc, tea.KeyCtrlC:
		m.editing = false
		m.query = ""
	case tea.KeyBackspace:
		if len(m.query) > 0 {
			r := []rune(m.query)
			m.query = string(r[:len(r)-1])
		}
	case tea.KeyRunes, tea.KeySpace:
		m.query += string(msg.Runes)
		if msg.Type == tea.KeySpace {
			m.query += " "
		}
	default:
		return m
	}
	return applyFilter(m)
}
