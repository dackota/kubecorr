package collect

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestContainerNames_InitContainersFirstThenAll(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{
		InitContainers: []corev1.Container{{Name: "init"}},
		Containers:     []corev1.Container{{Name: "app"}, {Name: "sidecar"}},
	}}

	got := containerNames(pod, "")

	if len(got) != 3 || got[0] != "init" || got[1] != "app" || got[2] != "sidecar" {
		t.Fatalf("got %v", got)
	}
}

func TestContainerNames_OnlyOneWhenAsked(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}, {Name: "sidecar"}}}}
	if got := containerNames(pod, "sidecar"); len(got) != 1 || got[0] != "sidecar" {
		t.Fatalf("got %v", got)
	}
}

func TestLogs_DoesNotErrorWithFakeClient(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: ns},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}},
	}
	cs := fake.NewSimpleClientset(pod)

	_, err := Logs(context.Background(), cs, pod, LogOptions{Since: time.Now().Add(-time.Hour)})

	if err != nil {
		t.Fatal(err)
	}
}

func TestTailLines_ZeroMeansNoLimit(t *testing.T) {
	if tailLines(0) != nil {
		t.Fatal("0 should mean unlimited")
	}
	if v := tailLines(500); v == nil || *v != 500 {
		t.Fatalf("got %v", v)
	}
}
