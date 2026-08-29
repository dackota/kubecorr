// Package summary builds a short restart and exit report for pods.
package summary

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// ContainerSummary is the restart state of one container.
type ContainerSummary struct {
	Name           string `json:"name"`
	Ready          bool   `json:"ready"`
	Restarts       int32  `json:"restarts"`
	State          string `json:"state"`
	LastExitCode   int32  `json:"lastExitCode,omitempty"`
	LastExitReason string `json:"lastExitReason,omitempty"`
}

// PodSummary is the header shown before the timeline.
type PodSummary struct {
	Pod        string             `json:"pod"`
	Node       string             `json:"node"`
	Phase      string             `json:"phase"`
	Containers []ContainerSummary `json:"containers"`
}

// FromPod reads restart and exit data out of a pod's status.
func FromPod(pod *corev1.Pod) PodSummary {
	s := PodSummary{Pod: pod.Name, Node: pod.Spec.NodeName, Phase: string(pod.Status.Phase)}
	all := append(append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
	for _, cs := range all {
		c := ContainerSummary{Name: cs.Name, Ready: cs.Ready, Restarts: cs.RestartCount, State: stateText(cs.State)}
		if term := cs.LastTerminationState.Terminated; term != nil {
			c.LastExitCode = term.ExitCode
			c.LastExitReason = term.Reason
		}
		s.Containers = append(s.Containers, c)
	}
	return s
}

func stateText(st corev1.ContainerState) string {
	switch {
	case st.Waiting != nil:
		return "Waiting: " + st.Waiting.Reason
	case st.Terminated != nil:
		return fmt.Sprintf("Terminated: %s (exit %d)", st.Terminated.Reason, st.Terminated.ExitCode)
	case st.Running != nil:
		return "Running"
	default:
		return "Unknown"
	}
}

// Text renders summaries as plain lines. Empty input gives "".
func Text(pods []PodSummary) string {
	if len(pods) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range pods {
		fmt.Fprintf(&b, "%s  node=%s  phase=%s\n", p.Pod, p.Node, p.Phase)
		for _, c := range p.Containers {
			fmt.Fprintf(&b, "  %-20s ready=%-5t restarts=%-3d %s", c.Name, c.Ready, c.Restarts, c.State)
			if c.LastExitReason != "" || c.LastExitCode != 0 {
				fmt.Fprintf(&b, "  last exit: %s (%d)", c.LastExitReason, c.LastExitCode)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
