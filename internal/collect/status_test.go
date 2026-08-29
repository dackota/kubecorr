package collect

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStatusItems_ConditionsAndTerminationsBecomeEvents(t *testing.T) {
	t0 := time.Unix(1000, 0)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p"},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse, Reason: "ContainersNotReady", Message: "containers with unready status: [app]", LastTransitionTime: metav1.NewTime(t0)},
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(t0.Add(-time.Minute))},
				{Type: corev1.PodInitialized, Status: corev1.ConditionTrue}, // zero time: skipped
			},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "app",
				LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					ExitCode: 137, Reason: "OOMKilled", FinishedAt: metav1.NewTime(t0.Add(-5 * time.Second))}},
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off 5m"}},
			}},
		},
	}

	got := StatusItems(pod, time.Unix(0, 0))

	reasons := map[string]bool{}
	for _, it := range got {
		reasons[it.Reason] = true
		if it.Kind != "EVENT" || it.Type != "Status" || it.Time.IsZero() {
			t.Fatalf("bad item %+v", it)
		}
	}
	for _, want := range []string{"Ready=False", "PodScheduled=True", "OOMKilled"} {
		if !reasons[want] {
			t.Fatalf("missing %q in %v", want, reasons)
		}
	}
	if reasons["Initialized=True"] {
		t.Fatal("zero-time condition should be skipped")
	}
}

func TestStatusItems_RespectsSince(t *testing.T) {
	pod := &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
		{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(time.Unix(10, 0))},
	}}}
	if got := StatusItems(pod, time.Unix(100, 0)); len(got) != 0 {
		t.Fatalf("want none got %v", got)
	}
}
