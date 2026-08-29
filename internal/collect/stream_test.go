package collect

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStream_DeliversNewEventsForTargetsOnly(t *testing.T) {
	cs, pod := fixture()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out := Stream(ctx, cs, []StreamTarget{{Pod: pod, Targets: []Target{{Kind: "Pod", Name: pod.Name}}}}, LogOptions{})
	time.Sleep(100 * time.Millisecond) // let the watch start

	_, _ = cs.CoreV1().Events(ns).Create(ctx, event("new-other", "Pod", "other", "Normal", "Ignore", time.Now()), metav1.CreateOptions{})
	_, _ = cs.CoreV1().Events(ns).Create(ctx, event("new-1", "Pod", pod.Name, "Warning", "Killing", time.Now()), metav1.CreateOptions{})

	for {
		select {
		case it := <-out:
			if it.Reason == "Ignore" {
				t.Fatal("got event for a non-target")
			}
			if it.Reason == "Killing" && it.Kind == "EVENT" {
				return
			}
		case <-ctx.Done():
			t.Fatal("no event arrived")
		}
	}
}

func TestStream_ClosesWhenContextEnds(t *testing.T) {
	cs, pod := fixture()
	ctx, cancel := context.WithCancel(context.Background())

	out := Stream(ctx, cs, []StreamTarget{{Pod: pod}}, LogOptions{})
	cancel()

	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("channel did not close")
		}
	}
}
