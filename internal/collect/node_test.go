package collect

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNodeItems_ReportsPressureAndNotReadyOnly(t *testing.T) {
	t0 := time.Unix(5000, 0)
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{
			{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue, Reason: "KubeletHasInsufficientMemory", LastTransitionTime: metav1.NewTime(t0)},
			{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse, LastTransitionTime: metav1.NewTime(t0)},
			{Type: corev1.NodeReady, Status: corev1.ConditionFalse, Reason: "KubeletNotReady", LastTransitionTime: metav1.NewTime(t0)},
		}},
	}
	cs := fake.NewSimpleClientset(node)

	got, err := NodeItems(context.Background(), cs, "node-1", time.Unix(0, 0))

	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 items got %+v", got)
	}
	for _, it := range got {
		if it.Source != "node/node-1" || it.Type != TypeStatus {
			t.Fatalf("bad item %+v", it)
		}
	}
}

func TestNodeItems_MissingNodeIsNotFatal(t *testing.T) {
	got, err := NodeItems(context.Background(), fake.NewSimpleClientset(), "gone", time.Unix(0, 0))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v %v", got, err)
	}
}

func TestNodeItems_EmptyNameReturnsNothing(t *testing.T) {
	got, err := NodeItems(context.Background(), fake.NewSimpleClientset(), "", time.Unix(0, 0))
	if err != nil || got != nil {
		t.Fatalf("got %v %v", got, err)
	}
}
