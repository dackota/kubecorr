// Package logparse turns kubelet log output (with --timestamps) into timeline items.
package logparse

import (
	"errors"
	"strings"
	"time"

	"github.com/dackota/kubecorr/internal/timeline"
)

// ErrNoTimestamp means the line did not start with an RFC3339 timestamp.
var ErrNoTimestamp = errors.New("line has no leading RFC3339 timestamp")

// ParseLine splits one kubelet log line into a timeline Item.
// The kubelet prefixes each line with an RFC3339Nano timestamp and a space.
func ParseLine(line string) (timeline.Item, error) {
	stamp, text, _ := strings.Cut(line, " ")
	ts, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		return timeline.Item{}, ErrNoTimestamp
	}
	return timeline.Item{Time: ts, Kind: timeline.KindLog, Text: text}, nil
}

// ParseAll parses every line in raw. Lines without a timestamp are skipped.
// source is the "pod/container" label put on each item.
func ParseAll(raw, source string) []timeline.Item {
	lines := strings.Split(raw, "\n")
	out := make([]timeline.Item, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			continue
		}
		it, err := ParseLine(l)
		if err != nil {
			continue
		}
		it.Source = source
		out = append(out, it)
	}
	return out
}
