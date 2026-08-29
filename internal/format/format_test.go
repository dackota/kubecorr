package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dackota/kubecorr/internal/timeline"
)

var ts = time.Date(2026, 8, 28, 14, 2, 11, 301000000, time.Local)

var items = []timeline.Item{
	{Time: ts, Kind: timeline.KindEvent, Source: "pod/api-7d9f", Type: "Warning", Reason: "BackOff", Text: "Back-off restarting"},
	{Time: ts.Add(500 * time.Millisecond), Kind: timeline.KindLog, Source: "api-7d9f/app", Text: "panic: nil"},
}

func TestText_OneLinePerItemWithKindAndTime(t *testing.T) {
	var buf bytes.Buffer

	if err := Text(&buf, items, false); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines got %d: %q", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "14:02:11.301 EVENT") || !strings.Contains(lines[0], "Warning") || !strings.Contains(lines[0], "BackOff") {
		t.Fatalf("event line wrong: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "14:02:11.801 LOG") || !strings.Contains(lines[1], "api-7d9f/app") || !strings.HasSuffix(lines[1], "panic: nil") {
		t.Fatalf("log line wrong: %q", lines[1])
	}
}

func TestText_NoColorMeansNoEscapeCodes(t *testing.T) {
	var buf bytes.Buffer
	_ = Text(&buf, items, false)
	if strings.Contains(buf.String(), "\x1b[") {
		t.Fatal("found escape codes with color off")
	}
}

func TestText_ColorAddsEscapeCodes(t *testing.T) {
	var buf bytes.Buffer
	_ = Text(&buf, items, true)
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatal("no escape codes with color on")
	}
}

func TestJSON_EmitsArrayRoundTrip(t *testing.T) {
	var buf bytes.Buffer

	if err := JSON(&buf, items); err != nil {
		t.Fatal(err)
	}

	var back []timeline.Item
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != 2 || back[0].Reason != "BackOff" || back[1].Text != "panic: nil" {
		t.Fatalf("round trip lost data: %+v", back)
	}
}

func TestJSON_EmptyIsEmptyArrayNotNull(t *testing.T) {
	var buf bytes.Buffer
	_ = JSON(&buf, nil)
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Fatalf("got %q", buf.String())
	}
}
