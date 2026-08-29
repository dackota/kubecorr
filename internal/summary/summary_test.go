package summary

import (
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podWithRestarts() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-1"},
		Spec:       corev1.PodSpec{NodeName: "node-1"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", RestartCount: 3, Ready: false,
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
						ExitCode: 137, Reason: "OOMKilled", FinishedAt: metav1.NewTime(time.Unix(100, 0))}}},
				{Name: "sidecar", RestartCount: 0, Ready: true,
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			},
		},
	}
}

func TestFromPod_CollectsRestartAndLastExit(t *testing.T) {
	s := FromPod(podWithRestarts())

	if s.Pod != "api-1" || s.Node != "node-1" || s.Phase != "Running" {
		t.Fatalf("bad header %+v", s)
	}
	if len(s.Containers) != 2 {
		t.Fatalf("want 2 containers got %d", len(s.Containers))
	}
	app := s.Containers[0]
	if app.Restarts != 3 || app.State != "Waiting: CrashLoopBackOff" || app.LastExitCode != 137 || app.LastExitReason != "OOMKilled" {
		t.Fatalf("bad app %+v", app)
	}
	if s.Containers[1].State != "Running" || s.Containers[1].LastExitReason != "" {
		t.Fatalf("bad sidecar %+v", s.Containers[1])
	}
}

func TestFromPod_EmptyStatusDoesNotPanic(t *testing.T) {
	s := FromPod(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p"}})
	if s.Pod != "p" || len(s.Containers) != 0 {
		t.Fatalf("got %+v", s)
	}
}

func TestText_ShowsOneLinePerContainer(t *testing.T) {
	out := Text([]PodSummary{FromPod(podWithRestarts())})

	if !strings.Contains(out, "api-1") || !strings.Contains(out, "node-1") {
		t.Fatalf("missing pod or node: %q", out)
	}
	if !strings.Contains(out, "app") || !strings.Contains(out, "restarts=3") || !strings.Contains(out, "OOMKilled") || !strings.Contains(out, "137") {
		t.Fatalf("missing restart info: %q", out)
	}
}

func TestText_EmptyIsEmpty(t *testing.T) {
	if Text(nil) != "" {
		t.Fatal("want empty")
	}
}
