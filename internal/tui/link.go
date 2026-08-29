// Package tui shows logs and events side by side, linked by time.
package tui

import (
	"sort"
	"time"

	"github.com/dackota/kubecorr/internal/timeline"
)

// nearestAtOrBefore returns the index of the last item whose time is not
// after t. Items must be sorted by time. Empty input or all items after t
// give 0.
func nearestAtOrBefore(items []timeline.Item, t time.Time) int {
	if len(items) == 0 {
		return 0
	}
	n := sort.Search(len(items), func(i int) bool { return items[i].Time.After(t) })
	if n == 0 {
		return 0
	}
	return n - 1
}

// windowStart returns the first visible index so that cursor is on screen.
// The cursor is kept near the middle when there is room.
func windowStart(cursor, total, height int) int {
	if height <= 0 || total <= height {
		if height <= 0 {
			return cursor
		}
		return 0
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	if start > total-height {
		start = total - height
	}
	return start
}

// clamp keeps i inside [0, n-1]. n <= 0 gives 0.
func clamp(i, n int) int {
	if n <= 0 || i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}
