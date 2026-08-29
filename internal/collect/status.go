package collect

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/dackota/kubecorr/internal/timeline"
)

// TypeStatus marks items built from pod or node status, not from Events.
// They stay in the cluster after Events have expired.
const TypeStatus = "Status"

// StatusItems turns pod conditions and container terminations into timeline
// items. Items older than since are dropped.
func StatusItems(pod *corev1.Pod, since time.Time) []timeline.Item {
	src := "pod/" + pod.Name
	var out []timeline.Item
	for _, c := range pod.Status.Conditions {
		at := c.LastTransitionTime.Time
		if at.IsZero() || at.Before(since) {
			continue
		}
		text := c.Message
		if c.Reason != "" {
			text = c.Reason + ": " + text
		}
		out = append(out, timeline.Item{Time: at, Kind: timeline.KindEvent, Type: TypeStatus, Source: src,
			Reason: fmt.Sprintf("%s=%s", c.Type, c.Status), Text: text})
	}
	all := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
	for _, cs := range all {
		out = append(out, terminationItems(pod.Name, cs, since)...)
	}
	return out
}

func terminationItems(podName string, cs corev1.ContainerStatus, since time.Time) []timeline.Item {
	src := podName + "/" + cs.Name
	var out []timeline.Item
	for _, term := range []*corev1.ContainerStateTerminated{cs.LastTerminationState.Terminated, cs.State.Terminated} {
		if term == nil || term.FinishedAt.IsZero() || term.FinishedAt.Time.Before(since) {
			continue
		}
		out = append(out, timeline.Item{Time: term.FinishedAt.Time, Kind: timeline.KindEvent, Type: TypeStatus, Source: src,
			Reason: term.Reason, Text: fmt.Sprintf("container exited with code %d%s", term.ExitCode, optMsg(term.Message))})
	}
	return out
}

func optMsg(s string) string {
	if s == "" {
		return ""
	}
	return ": " + s
}
