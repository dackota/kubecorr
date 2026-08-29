package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dackota/kubecorr/internal/timeline"
)

// ItemMsg carries one new item from a live stream into the model.
type ItemMsg timeline.Item

// streamDoneMsg says the stream channel closed.
type streamDoneMsg struct{}

// WithFollow turns follow mode on or off at start. Returns a new model.
func (m Model) WithFollow(on bool) Model {
	m.follow = on
	return m
}

// WithStream makes Init read items from ch until it closes or ctx ends.
func (m Model) WithStream(ctx context.Context, ch <-chan timeline.Item) Model {
	m.stream = ch
	m.streamCtx = ctx
	return m
}

func waitForItem(ctx context.Context, ch <-chan timeline.Item) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case it, ok := <-ch:
			if !ok {
				return streamDoneMsg{}
			}
			return ItemMsg(it)
		case <-ctx.Done():
			return streamDoneMsg{}
		}
	}
}

// insertSorted returns a new slice with it placed by time.
func insertSorted(items []timeline.Item, it timeline.Item) []timeline.Item {
	i := len(items)
	for i > 0 && items[i-1].Time.After(it.Time) {
		i--
	}
	out := make([]timeline.Item, 0, len(items)+1)
	out = append(out, items[:i]...)
	out = append(out, it)
	return append(out, items[i:]...)
}

func addItem(m Model, it timeline.Item) Model {
	if it.Kind == timeline.KindEvent {
		m.events = insertSorted(m.events, it)
	} else {
		m.logs = insertSorted(m.logs, it)
	}
	if m.follow {
		m = jumpToEnd(m)
	}
	return m
}

func jumpToEnd(m Model) Model {
	m.logCursor = clamp(len(m.logs)-1, len(m.logs))
	m.eventCursor = clamp(len(m.events)-1, len(m.events))
	return m
}
