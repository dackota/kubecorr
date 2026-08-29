package logparse

import (
	"math/rand"
	"testing"
	"time"
)

func TestParseLine_SplitsKubeletTimestampFromText(t *testing.T) {
	line := "2026-08-28T14:02:11.804123456Z panic: nil pointer"

	got, err := ParseLine(line)

	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 28, 14, 2, 11, 804123456, time.UTC)
	if !got.Time.Equal(want) {
		t.Fatalf("time: want %v got %v", want, got.Time)
	}
	if got.Text != "panic: nil pointer" {
		t.Fatalf("text: got %q", got.Text)
	}
}

func TestParseLine_ErrorsWhenNoTimestamp(t *testing.T) {
	if _, err := ParseLine("just text"); err == nil {
		t.Fatal("want error")
	}
}

func TestParseLine_EmptyTextAfterTimestampIsOK(t *testing.T) {
	got, err := ParseLine("2026-08-28T14:02:11Z")
	if err != nil || got.Text != "" {
		t.Fatalf("got %v %v", got, err)
	}
}

// Property: never panics on garbage.
func TestParseLine_PropertyNeverPanics(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	alphabet := []byte("0123456789TZ:-. \x00\xff abcXYZ")
	for i := 0; i < 5000; i++ {
		n := r.Intn(60)
		b := make([]byte, n)
		for j := range b {
			b[j] = alphabet[r.Intn(len(alphabet))]
		}
		_, _ = ParseLine(string(b))
	}
}

func TestParseAll_SkipsBadLinesAndTagsSource(t *testing.T) {
	in := "2026-08-28T14:02:11Z one\nbad line\n2026-08-28T14:02:12Z two\n"

	got := ParseAll(in, "pod-a/app")

	if len(got) != 2 {
		t.Fatalf("want 2 got %d", len(got))
	}
	if got[0].Source != "pod-a/app" || got[1].Text != "two" {
		t.Fatalf("bad items: %+v", got)
	}
}
