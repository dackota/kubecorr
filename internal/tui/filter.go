package tui

import (
	"strings"

	"github.com/dackota/kubecorr/internal/timeline"
)

// matches reports whether an item's source, reason or text contains q.
// The check ignores letter case. An empty q matches everything.
func matches(it timeline.Item, q string) bool {
	if q == "" {
		return true
	}
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(it.Source), q) ||
		strings.Contains(strings.ToLower(it.Reason), q) ||
		strings.Contains(strings.ToLower(it.Text), q)
}

// filter returns the items that match q. With an empty q it returns the
// input itself, so no copy is made in the common case.
func filter(items []timeline.Item, q string) []timeline.Item {
	if q == "" {
		return items
	}
	out := make([]timeline.Item, 0, len(items))
	for _, it := range items {
		if matches(it, q) {
			out = append(out, it)
		}
	}
	return out
}

// applyFilter recomputes the visible lists and clamps the cursors.
func applyFilter(m Model) Model {
	m.visLogs = filter(m.logs, m.query)
	m.visEvents = filter(m.events, m.query)
	m.logCursor = clamp(m.logCursor, len(m.visLogs))
	m.eventCursor = clamp(m.eventCursor, len(m.visEvents))
	return m
}
