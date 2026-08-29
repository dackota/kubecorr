package timeline

import (
	"math/rand"
	"sort"
	"testing"
	"time"
)

func mk(sec int, k Kind) Item {
	return Item{Time: time.Unix(int64(sec), 0), Kind: k, Text: "x"}
}

func TestMerge_ReturnsSortedUnionOfBothLists(t *testing.T) {
	logs := []Item{mk(1, KindLog), mk(5, KindLog)}
	events := []Item{mk(3, KindEvent)}

	got := Merge(logs, events)

	if len(got) != 3 {
		t.Fatalf("want 3 items, got %d", len(got))
	}
	if got[0].Time.Unix() != 1 || got[1].Time.Unix() != 3 || got[2].Time.Unix() != 5 {
		t.Fatalf("not sorted: %v", got)
	}
}

func TestMerge_EmptyInputsReturnEmpty(t *testing.T) {
	if got := Merge(nil, nil); len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}

func TestMerge_DoesNotMutateInputs(t *testing.T) {
	logs := []Item{mk(9, KindLog), mk(1, KindLog)}
	copyLogs := append([]Item(nil), logs...)

	Merge(logs, nil)

	for i := range logs {
		if logs[i] != copyLogs[i] {
			t.Fatal("input slice was changed")
		}
	}
}

// Property: for random inputs, output is sorted and keeps every item.
func TestMerge_PropertySortedAndComplete(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for round := 0; round < 200; round++ {
		var logs, events []Item
		for i := 0; i < r.Intn(20); i++ {
			logs = append(logs, mk(r.Intn(50), KindLog))
		}
		for i := 0; i < r.Intn(20); i++ {
			events = append(events, mk(r.Intn(50), KindEvent))
		}

		got := Merge(logs, events)

		if len(got) != len(logs)+len(events) {
			t.Fatalf("lost items: %d != %d", len(got), len(logs)+len(events))
		}
		if !sort.SliceIsSorted(got, func(i, j int) bool { return got[i].Time.Before(got[j].Time) }) {
			t.Fatalf("not sorted: %v", got)
		}
	}
}
