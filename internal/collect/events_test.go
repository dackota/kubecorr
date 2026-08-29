package collect

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const ns = "default"

func owner(kind, name string) metav1.OwnerReference {
	return metav1.OwnerReference{Kind: kind, Name: name, APIVersion: "apps/v1"}
}

func event(name, kind, obj, typ, reason string, at time.Time) *corev1.Event {
	return &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: name, Namespace: ns},
		InvolvedObject: corev1.ObjectReference{Kind: kind, Name: obj, Namespace: ns},
		Type:           typ, Reason: reason, Message: reason + " msg",
		LastTimestamp:  metav1.NewTime(at),
	}
}

func fixture() (*fake.Clientset, *corev1.Pod) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-7d9f-abc", Namespace: ns, OwnerReferences: []metav1.OwnerReference{owner("ReplicaSet", "api-7d9f")}},
		Spec:       corev1.PodSpec{NodeName: "node-1"},
	}
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "api-7d9f", Namespace: ns, OwnerReferences: []metav1.OwnerReference{owner("Deployment", "api")}}}
	now := time.Now()
	cs := fake.NewSimpleClientset(pod, rs,
		event("e1", "Pod", "api-7d9f-abc", "Warning", "BackOff", now),
		event("e2", "ReplicaSet", "api-7d9f", "Normal", "SuccessfulCreate", now),
		event("e3", "Deployment", "api", "Normal", "ScalingReplicaSet", now),
		event("e4", "Pod", "other-pod", "Warning", "Unrelated", now),
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "e5", Namespace: ns},
			InvolvedObject: corev1.ObjectReference{Kind: "Node", Name: "node-1"},
			Type:           "Warning", Reason: "NodeNotReady", LastTimestamp: metav1.NewTime(now),
		},
	)
	return cs, pod
}

func TestTargets_FollowsOwnerChainAndNode(t *testing.T) {
	cs, pod := fixture()

	got, err := Targets(context.Background(), cs, pod)

	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"Pod/api-7d9f-abc": true, "ReplicaSet/api-7d9f": true, "Deployment/api": true, "Node/node-1": true}
	if len(got) != len(want) {
		t.Fatalf("want %d targets got %v", len(want), got)
	}
	for _, tg := range got {
		if !want[tg.Kind+"/"+tg.Name] {
			t.Fatalf("unexpected target %v", tg)
		}
	}
}

func TestEvents_KeepsOnlyTargetEventsAndSkipsOld(t *testing.T) {
	cs, pod := fixture()
	old := event("e-old", "Pod", "api-7d9f-abc", "Normal", "Old", time.Now().Add(-3*time.Hour))
	_, _ = cs.CoreV1().Events(ns).Create(context.Background(), old, metav1.CreateOptions{})
	targets, _ := Targets(context.Background(), cs, pod)

	got, err := Events(context.Background(), cs, ns, targets, time.Now().Add(-time.Hour))

	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("want 4 events got %d: %+v", len(got), got)
	}
	for _, it := range got {
		if it.Reason == "Unrelated" || it.Reason == "Old" {
			t.Fatalf("kept filtered event %+v", it)
		}
		if it.Kind != "EVENT" || it.Source == "" {
			t.Fatalf("bad item %+v", it)
		}
	}
}
