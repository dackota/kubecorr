package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dackota/kubecorr/internal/timeline"
)

func bigModel(n int) Model {
	logs := make([]timeline.Item, 0, n)
	events := make([]timeline.Item, 0, n/10)
	for i := 0; i < n; i++ {
		logs = append(logs, timeline.Item{Time: time.Unix(int64(i), 0), Kind: timeline.KindLog, Source: "pod-a/app", Text: "some log line with error text in it"})
		if i%10 == 0 {
			events = append(events, timeline.Item{Time: time.Unix(int64(i), 0), Kind: timeline.KindEvent, Type: "Warning", Reason: "BackOff", Source: "pod/pod-a", Text: "back off"})
		}
	}
	m := New(logs, events, "ns", "ctx")
	m, _ = update(m, tea.WindowSizeMsg{Width: 160, Height: 50})
	m.logCursor = n / 2
	return m
}

func BenchmarkView50k(b *testing.B) {
	m := bigModel(50000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkScroll50k(b *testing.B) {
	m := bigModel(50000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m, _ = update(m, key("j"))
	}
}

func BenchmarkAddItem50k(b *testing.B) {
	m := bigModel(50000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m = addItem(m, timeline.Item{Time: time.Unix(int64(60000+i), 0), Kind: timeline.KindLog, Text: "x"})
	}
}
