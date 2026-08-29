package collect

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/dackota/kubecorr/internal/timeline"
)

// NodeItems reports node conditions that mean trouble: any pressure
// condition that is True, or Ready that is not True. A node that no longer
// exists gives no items and no error.
func NodeItems(ctx context.Context, cs kubernetes.Interface, name string, since time.Time) ([]timeline.Item, error) {
	if name == "" {
		return nil, nil
	}
	node, err := cs.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get node %s: %w", name, err)
	}
	var out []timeline.Item
	for _, c := range node.Status.Conditions {
		if !isNodeTrouble(c) || c.LastTransitionTime.Time.Before(since) {
			continue
		}
		out = append(out, timeline.Item{
			Time: c.LastTransitionTime.Time, Kind: timeline.KindEvent, Type: TypeStatus,
			Source: "node/" + name, Reason: fmt.Sprintf("%s=%s", c.Type, c.Status),
			Text: c.Reason + optMsg(c.Message),
		})
	}
	return out, nil
}

func isNodeTrouble(c corev1.NodeCondition) bool {
	if c.Type == corev1.NodeReady {
		return c.Status != corev1.ConditionTrue
	}
	return c.Status == corev1.ConditionTrue
}
