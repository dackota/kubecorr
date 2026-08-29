package collect

import (
	"bufio"
	"context"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/dackota/kubecorr/internal/logparse"
	"github.com/dackota/kubecorr/internal/timeline"
)

// StreamTarget is one pod to follow plus the objects whose events we want.
type StreamTarget struct {
	Pod     *corev1.Pod
	Targets []Target
}

// Stream follows container logs and watches events for every target. Items
// arrive on the returned channel as they happen. The channel closes when ctx
// ends. Errors from a single stream end that stream only; the rest go on.
func Stream(ctx context.Context, cs kubernetes.Interface, targets []StreamTarget, opts LogOptions) <-chan timeline.Item {
	out := make(chan timeline.Item, 256)
	var wg sync.WaitGroup
	start := time.Now()

	want := make(map[string]bool)
	for _, st := range targets {
		for _, t := range st.Targets {
			want[t.Kind+"/"+t.Name] = true
		}
		for _, name := range containerNames(st.Pod, opts.Container) {
			wg.Add(1)
			go func(pod *corev1.Pod, container string) {
				defer wg.Done()
				followLog(ctx, cs, pod, container, start, out)
			}(st.Pod, name)
		}
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		watchEvents(ctx, cs, want, start, out)
	}()
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func followLog(ctx context.Context, cs kubernetes.Interface, pod *corev1.Pod, container string, since time.Time, out chan<- timeline.Item) {
	sinceMeta := metav1.NewTime(since)
	req := cs.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: container, Follow: true, Timestamps: true, SinceTime: &sinceMeta,
	})
	rc, err := req.Stream(ctx)
	if err != nil {
		return
	}
	defer rc.Close()
	sc := bufio.NewScanner(rc)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		it, err := logparse.ParseLine(sc.Text())
		if err != nil {
			continue
		}
		it.Source = pod.Name + "/" + container
		if !send(ctx, out, it) {
			return
		}
	}
}

func watchEvents(ctx context.Context, cs kubernetes.Interface, want map[string]bool, since time.Time, out chan<- timeline.Item) {
	w, err := cs.CoreV1().Events("").Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-w.ResultChan():
			if !ok {
				return
			}
			if e.Type != watch.Added && e.Type != watch.Modified {
				continue
			}
			ev, ok := e.Object.(*corev1.Event)
			if !ok || !want[ev.InvolvedObject.Kind+"/"+ev.InvolvedObject.Name] {
				continue
			}
			at := eventTime(*ev)
			if at.Before(since) {
				continue
			}
			it := timeline.Item{Time: at, Kind: timeline.KindEvent, Type: ev.Type, Reason: ev.Reason,
				Source: sourceOf(ev.InvolvedObject), Text: ev.Message}
			if !send(ctx, out, it) {
				return
			}
		}
	}
}

func send(ctx context.Context, out chan<- timeline.Item, it timeline.Item) bool {
	select {
	case out <- it:
		return true
	case <-ctx.Done():
		return false
	}
}
