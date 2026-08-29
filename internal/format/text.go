// Package format writes timeline items as text or JSON.
package format

import (
	"fmt"
	"io"

	"github.com/dackota/kubecorr/internal/timeline"
)

const (
	ansiReset  = "\x1b[0m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiDim    = "\x1b[2m"
	timeLayout = "15:04:05.000"
)

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
	line := fmt.Sprintf("%s %-5s %s  %s", stamp, it.Kind, it.Source, it.Text)
	if !useColor {
		return line
	}
	return ansiDim + stamp + " " + fmt.Sprintf("%-5s", it.Kind) + ansiReset + " " + it.Source + "  " + it.Text
}
