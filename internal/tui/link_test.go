package tui

import (
	"math/rand"
	"testing"
	"time"

	"github.com/dackota/kubecorr/internal/timeline"
)

func at(sec int) timeline.Item { return timeline.Item{Time: time.Unix(int64(sec), 0)} }

func TestNearestAtOrBefore_PicksLastItemNotAfterTime(t *testing.T) {
	items := []timeline.Item{at(1), at(5), at(9)}

	if got := nearestAtOrBefore(items, time.Unix(6, 0)); got != 1 {
		t.Fatalf("want 1 got %d", got)
	}
	if got := nearestAtOrBefore(items, time.Unix(5, 0)); got != 1 {
		t.Fatalf("exact match: want 1 got %d", got)
	}
}

func TestNearestAtOrBefore_BeforeFirstReturnsZero(t *testing.T) {
	if got := nearestAtOrBefore([]timeline.Item{at(5)}, time.Unix(1, 0)); got != 0 {
		t.Fatalf("want 0 got %d", got)
	}
}

func TestNearestAtOrBefore_EmptyReturnsZero(t *testing.T) {
	if got := nearestAtOrBefore(nil, time.Unix(1, 0)); got != 0 {
		t.Fatal("want 0")
	}
}

func TestNearestAtOrBefore_PropertyResultIsValidIndex(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for round := 0; round < 500; round++ {
		n := r.Intn(30)
		items := make([]timeline.Item, n)
		sec := 0
		for i := range items {
			sec += r.Intn(3)
			items[i] = at(sec)
		}
		got := nearestAtOrBefore(items, time.Unix(int64(r.Intn(60)), 0))
		if n == 0 && got != 0 {
			t.Fatal("empty must give 0")
		}
		if n > 0 && (got < 0 || got >= n) {
			t.Fatalf("index %d out of range %d", got, n)
		}
	}
}

func TestWindowStart_KeepsCursorVisible(t *testing.T) {
	cases := []struct{ cursor, total, height, want int }{
		{0, 100, 10, 0},
		{5, 100, 10, 0},
		{10, 100, 10, 5},
		{99, 100, 10, 90},
		{3, 5, 10, 0},
		{0, 0, 10, 0},
		{4, 10, 0, 4},
	}
	for _, c := range cases {
		if got := windowStart(c.cursor, c.total, c.height); got != c.want {
			t.Errorf("windowStart(%d,%d,%d) = %d want %d", c.cursor, c.total, c.height, got, c.want)
		}
	}
}

func TestClamp(t *testing.T) {
	if clamp(-1, 5) != 0 || clamp(7, 5) != 4 || clamp(2, 5) != 2 || clamp(0, 0) != 0 {
		t.Fatal("clamp wrong")
	}
}
