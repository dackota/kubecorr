package collect

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// EventOption changes how Events and Stream pick events.
type EventOption func(*eventFilter)

// WithExtraNamespaces also keeps Warning events from these namespaces, no
// matter which object they name. Useful for kube-system.
func WithExtraNamespaces(namespaces ...string) EventOption {
	return func(f *eventFilter) {
		for _, n := range namespaces {
			if n != "" {
				f.extra[n] = true
			}
		}
	}
}

// eventFilter decides which events belong on the timeline.
type eventFilter struct {
	ns    string          // the pod's namespace
	want  map[string]bool // "Kind/name" of the pod, its owners, its node
	extra map[string]bool // namespaces whose Warnings are always kept
}

func newEventFilter(ns string, targets []Target, opts []EventOption) eventFilter {
	f := eventFilter{ns: ns, want: make(map[string]bool, len(targets)), extra: map[string]bool{}}
	for _, t := range targets {
		f.want[t.Kind+"/"+t.Name] = true
	}
	for _, o := range opts {
		o(&f)
	}
	return f
}

// keep reports whether an event about obj with the given type belongs.
func (f eventFilter) keep(obj corev1.ObjectReference, eventType string) bool {
	if f.extra[obj.Namespace] && eventType == corev1.EventTypeWarning {
		return true
	}
	if obj.Kind != "Node" && obj.Namespace != f.ns {
		return false
	}
	return f.want[obj.Kind+"/"+obj.Name]
}

// source names an involved object. Objects outside the pod's namespace get
// a namespace prefix so the reader can tell them apart.
func (f eventFilter) source(obj corev1.ObjectReference) string {
	s := strings.ToLower(obj.Kind) + "/" + obj.Name
	if obj.Namespace != "" && obj.Namespace != f.ns && obj.Kind != "Node" {
		return obj.Namespace + "/" + s
	}
	return s
}
