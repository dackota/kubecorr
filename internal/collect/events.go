// Package collect reads pods, events, and logs from a cluster.
package collect

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dackota/kubecorr/internal/timeline"
)

// Target is one object whose events we want.
type Target struct {
	Kind string
	Name string
}

// Targets returns the pod, each owner up the chain (ReplicaSet, Deployment,
// Job, StatefulSet, ...), and the node the pod runs on.
func Targets(ctx context.Context, cs kubernetes.Interface, pod *corev1.Pod) ([]Target, error) {
	out := []Target{{Kind: "Pod", Name: pod.Name}}
	if pod.Spec.NodeName != "" {
		out = append(out, Target{Kind: "Node", Name: pod.Spec.NodeName})
	}
	owners, err := ownerChain(ctx, cs, pod.Namespace, pod.OwnerReferences)
	if err != nil {
		return nil, err
	}
	return append(out, owners...), nil
}

// ownerChain walks OwnerReferences upward. It only knows how to read the
// owners of a ReplicaSet (to reach a Deployment); other kinds stop the walk.
func ownerChain(ctx context.Context, cs kubernetes.Interface, ns string, refs []metav1.OwnerReference) ([]Target, error) {
	var out []Target
	for _, ref := range refs {
		out = append(out, Target{Kind: ref.Kind, Name: ref.Name})
		if ref.Kind != "ReplicaSet" {
			continue
		}
		rs, err := cs.AppsV1().ReplicaSets(ns).Get(ctx, ref.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("get replicaset %s: %w", ref.Name, err)
		}
		parents, err := ownerChain(ctx, cs, ns, rs.OwnerReferences)
		if err != nil {
			return nil, err
		}
		out = append(out, parents...)
	}
	return out, nil
}

// Events lists events in ns (plus cluster-scoped node events) that involve one
// of the targets and happened at or after since.
func Events(ctx context.Context, cs kubernetes.Interface, ns string, targets []Target, since time.Time, opts ...EventOption) ([]timeline.Item, error) {
	f := newEventFilter(ns, targets, opts)
	list, err := cs.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	var out []timeline.Item
	for _, ev := range list.Items {
		obj := ev.InvolvedObject
		if !f.keep(obj, ev.Type) {
			continue
		}
		at := eventTime(ev)
		if at.Before(since) {
			continue
		}
		out = append(out, timeline.Item{
			Time:   at,
			Kind:   timeline.KindEvent,
			Source: f.source(obj),
			Type:   ev.Type,
			Reason: ev.Reason,
			Text:   ev.Message,
			Count:  int(ev.Count),
		})
	}
	return out, nil
}

// eventTime picks the best timestamp an Event offers.
func eventTime(ev corev1.Event) time.Time {
	switch {
	case !ev.LastTimestamp.IsZero():
		return ev.LastTimestamp.Time
	case !ev.EventTime.IsZero():
		return ev.EventTime.Time
	default:
		return ev.CreationTimestamp.Time
	}
}
