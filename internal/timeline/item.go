// Package timeline holds the shared Item type and the merge step.
package timeline

import "time"

// Kind says where an Item came from.
type Kind string

const (
	KindLog   Kind = "LOG"
	KindEvent Kind = "EVENT"
)

// Item is one line on the timeline: a log line or a cluster event.
type Item struct {
	Time   time.Time `json:"time"`
	Kind   Kind      `json:"kind"`
	Source string    `json:"source"`           // pod/container for logs; involved object for events
	Type   string    `json:"type,omitempty"`   // event only: Normal or Warning
	Reason string    `json:"reason,omitempty"` // event only
	Text   string    `json:"text"`
}
