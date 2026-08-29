// Package format writes timeline items as text or JSON.
package format

import (
	"fmt"
	"hash/fnv"
	"io"
	"strings"

	"github.com/dackota/kubecorr/internal/timeline"
)

const (
	ansiReset  = "\x1b[0m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiDim    = "\x1b[2m"
	timeLayout = "15:04:05.000"
)

// podPalette is the set of colors used for pod names. Cyan and yellow are
// left out so log lines do not look like events.
var podPalette = []string{
	"\x1b[32m", // green
	"\x1b[35m", // magenta
	"\x1b[34m", // blue
	"\x1b[31m", // red
	"\x1b[92m", // bright green
	"\x1b[95m", // bright magenta
	"\x1b[94m", // bright blue
	"\x1b[91m", // bright red
}

// Text writes one line per item. When useColor is true, ANSI colors are added.
func Text(w io.Writer, items []timeline.Item, useColor bool) error {
	for _, it := range items {
		if _, err := fmt.Fprintln(w, textLine(it, useColor)); err != nil {
			return fmt.Errorf("write text line: %w", err)
		}
	}
	return nil
}

func textLine(it timeline.Item, useColor bool) string {
	stamp := it.Time.Local().Format(timeLayout)
	if it.Kind == timeline.KindEvent {
		return eventLine(stamp, it, useColor)
	}
	return logLine(stamp, it, useColor)
}

func eventLine(stamp string, it timeline.Item, useColor bool) string {
	line := fmt.Sprintf("%s %-5s %-7s %-10s %s  %s", stamp, it.Kind, it.Type, it.Reason, it.Source, it.Text)
	if !useColor {
		return line
	}
	color := ansiCyan
	if it.Type == "Warning" {
		color = ansiYellow
	}
	return color + line + ansiReset
}

func logLine(stamp string, it timeline.Item, useColor bool) string {
	if !useColor {
		return fmt.Sprintf("%s %-5s %s  %s", stamp, it.Kind, it.Source, it.Text)
	}
	head := ansiDim + stamp + " " + fmt.Sprintf("%-5s", it.Kind) + ansiReset
	src := podColor(it.Source) + it.Source + ansiReset
	return head + " " + src + "  " + it.Text
}

// podColor returns a stable color for the pod part of "pod/container".
func podColor(source string) string {
	pod, _, _ := strings.Cut(source, "/")
	h := fnv.New32a()
	_, _ = h.Write([]byte(pod))
	return podPalette[h.Sum32()%uint32(len(podPalette))]
}
