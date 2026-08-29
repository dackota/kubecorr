package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dackota/kubecorr/internal/timeline"
)

func fixtureModel() Model {
	logs := []timeline.Item{at(1), at(4), at(6), at(10)}
	events := []timeline.Item{at(2), at(5), at(9)}
	m := New(logs, events, "api", "prod")
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 20})
	return m
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestScrollLogsMovesEventCursorByTime(t *testing.T) {
	m := fixtureModel()

	m, _ = update(m, key("j")) // log 4
	m, _ = update(m, key("j")) // log 6

	if m.logCursor != 2 {
		t.Fatalf("logCursor want 2 got %d", m.logCursor)
	}
	if m.eventCursor != 1 { // event at 5 is nearest <= 6
		t.Fatalf("eventCursor want 1 got %d", m.eventCursor)
	}
}

func TestTabThenScrollEventsMovesLogCursor(t *testing.T) {
	m := fixtureModel()

	m, _ = update(m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = update(m, key("G")) // event 9

	if m.focus != focusEvents || m.eventCursor != 2 {
		t.Fatalf("focus=%v eventCursor=%d", m.focus, m.eventCursor)
	}
	if m.logCursor != 2 { // log at 6 is nearest <= 9
		t.Fatalf("logCursor want 2 got %d", m.logCursor)
	}
}

func TestCursorDoesNotGoOutOfRange(t *testing.T) {
	m := fixtureModel()
	m, _ = update(m, key("k"))
	if m.logCursor != 0 {
		t.Fatal("went above 0")
	}
	for i := 0; i < 20; i++ {
		m, _ = update(m, key("j"))
	}
	if m.logCursor != 3 {
		t.Fatalf("went past end: %d", m.logCursor)
	}
}

func TestQuitReturnsQuitCmd(t *testing.T) {
	m := fixtureModel()
	_, cmd := update(m, key("q"))
	if cmd == nil {
		t.Fatal("want quit cmd")
	}
}

func TestWrapToggles(t *testing.T) {
	m := fixtureModel()
	m, _ = update(m, key("w"))
	if !m.wrap {
		t.Fatal("wrap should be on")
	}
}

func TestViewRendersWithoutPanicOnEmptyData(t *testing.T) {
	m := New(nil, nil, "ns", "ctx")
	m, _ = update(m, tea.WindowSizeMsg{Width: 80, Height: 10})
	if out := m.View(); out == "" {
		t.Fatal("empty view")
	}
}

func TestViewContainsBothPaneTitles(t *testing.T) {
	m := fixtureModel()
	out := m.View()
	for _, want := range []string{"Logs", "Events", "api"} {
		if !contains(out, want) {
			t.Fatalf("view missing %q", want)
		}
	}
	_ = time.Now
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestViewShowsSummaryUnderHeader(t *testing.T) {
	m := fixtureModel().WithSummary(sampleSummaries())
	if out := m.View(); !contains(out, "restarts 7") {
		t.Fatal("summary missing from view")
	}
}

func TestItemMsgAppendsInTimeOrder(t *testing.T) {
	m := fixtureModel()

	m, _ = update(m, ItemMsg(timeline.Item{Time: time.Unix(5, 0), Kind: timeline.KindLog, Text: "late"}))
	m, _ = update(m, ItemMsg(timeline.Item{Time: time.Unix(7, 0), Kind: timeline.KindEvent, Text: "ev"}))

	if len(m.logs) != 5 || m.logs[2].Text != "late" {
		t.Fatalf("log insert wrong: %+v", m.logs)
	}
	if len(m.events) != 4 || m.events[2].Text != "ev" {
		t.Fatalf("event insert wrong: %+v", m.events)
	}
}

func TestFollowKeepsCursorAtEndAndScrollTurnsItOff(t *testing.T) {
	m := fixtureModel().WithFollow(true)

	m, _ = update(m, ItemMsg(timeline.Item{Time: time.Unix(20, 0), Kind: timeline.KindLog, Text: "new"}))
	if m.logCursor != len(m.logs)-1 {
		t.Fatalf("follow should move to end, cursor=%d", m.logCursor)
	}

	m, _ = update(m, key("k"))
	if m.follow {
		t.Fatal("scrolling up should turn follow off")
	}

	m, _ = update(m, key("f"))
	if !m.follow || m.logCursor != len(m.logs)-1 {
		t.Fatalf("f should turn follow on and jump to end, follow=%v cursor=%d", m.follow, m.logCursor)
	}
}

func longModel(wrap bool) Model {
	long := strings.Repeat("very long log text ", 20)
	var logs, events []timeline.Item
	for i := 0; i < 30; i++ {
		logs = append(logs, timeline.Item{Time: time.Unix(int64(i), 0), Kind: timeline.KindLog, Source: "pod-a/app", Text: long})
		events = append(events, timeline.Item{Time: time.Unix(int64(i), 0), Kind: timeline.KindEvent, Type: "Warning", Reason: "BackOff", Source: "pod/pod-a", Text: long})
	}
	m := New(logs, events, "ns", "ctx").WithSummary(sampleSummaries())
	m.wrap = wrap
	m, _ = update(m, tea.WindowSizeMsg{Width: 100, Height: 24})
	return m
}

func TestView_AlwaysFillsExactlyTheTerminalHeight(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		m := longModel(wrap)
		for _, cursor := range []int{0, 15, 29} {
			m.logCursor = cursor
			got := strings.Count(m.View(), "\n") + 1
			if got != m.height {
				t.Errorf("wrap=%v cursor=%d: view has %d lines, terminal has %d", wrap, cursor, got, m.height)
			}
		}
	}
}

func TestView_WrappedCursorLineIsVisible(t *testing.T) {
	m := longModel(true)
	m.logCursor = 15
	out := stripANSI(m.View())
	// The cursor row is rendered from the item's text, so the text must appear.
	if !strings.Contains(out, "very long log text") {
		t.Fatal("no log text visible with wrap on")
	}
}
