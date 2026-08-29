package collect

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dackota/kubecorr/internal/logparse"
	"github.com/dackota/kubecorr/internal/timeline"
)

// LogOptions controls which container logs are read.
type LogOptions struct {
	Container string // empty means every container in the pod
	Previous  bool   // read the last terminated container's logs
	Since     time.Time
}

// Logs reads logs with kubelet timestamps for the chosen containers and
// returns them as timeline items tagged "pod/container".
func Logs(ctx context.Context, cs kubernetes.Interface, pod *corev1.Pod, opts LogOptions) ([]timeline.Item, error) {
	var out []timeline.Item
	for _, name := range containerNames(pod, opts.Container) {
		raw, err := readContainerLog(ctx, cs, pod, name, opts)
		if err != nil {
			return nil, fmt.Errorf("logs for %s/%s: %w", pod.Name, name, err)
		}
		out = append(out, logparse.ParseAll(raw, pod.Name+"/"+name)...)
	}
	return out, nil
}

func containerNames(pod *corev1.Pod, only string) []string {
	if only != "" {
		return []string{only}
	}
	var names []string
	for _, c := range pod.Spec.InitContainers {
		names = append(names, c.Name)
	}
	for _, c := range pod.Spec.Containers {
		names = append(names, c.Name)
	}
	return names
}

func readContainerLog(ctx context.Context, cs kubernetes.Interface, pod *corev1.Pod, container string, opts LogOptions) (string, error) {
	seconds := int64(time.Since(opts.Since).Seconds())
	if seconds < 1 {
		seconds = 1
	}
	req := cs.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container:    container,
		Previous:     opts.Previous,
		Timestamps:   true,
		SinceSeconds: &seconds,
	})
	rc, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
