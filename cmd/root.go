// Package cmd holds the kubecorr command line.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/dackota/kubecorr/internal/collect"
	"github.com/dackota/kubecorr/internal/format"
	"github.com/dackota/kubecorr/internal/timeline"
)

const (
	defaultSince = time.Hour
	outputText   = "text"
	outputJSON   = "json"
)

type options struct {
	config    *genericclioptions.ConfigFlags
	selector  string
	container string
	previous  bool
	since     time.Duration
	output    string
}

// New builds the root command.
func New() *cobra.Command {
	o := &options{config: genericclioptions.NewConfigFlags(true)}
	cmd := &cobra.Command{
		Use:   "kubecorr [POD]",
		Short: "Merge pod logs and cluster events into one timeline",
		Long: `kubecorr reads a pod's container logs and the events for the pod, its
owners (ReplicaSet, Deployment, Job, ...) and its node. It merges them by
time into one list so you can see what the cluster did next to what the
app printed.

Give a pod name, or use -l to pick pods by label.`,
		Example: `  kubecorr --context prod -n api api-7d9f-abc
  kubecorr --context prod -n api -l app=api --since 30m
  kubecorr --context prod -n api api-7d9f-abc --previous -o json | jq .`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd.Context(), cmd.OutOrStdout(), args)
		},
	}
	o.config.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.selector, "selector", "l", "", "label selector to pick pods (instead of POD)")
	cmd.Flags().StringVarP(&o.container, "container", "c", "", "only this container (default: all)")
	cmd.Flags().BoolVarP(&o.previous, "previous", "p", false, "read logs of the last terminated container")
	cmd.Flags().DurationVar(&o.since, "since", defaultSince, "how far back to look")
	cmd.Flags().StringVarP(&o.output, "output", "o", outputText, "output format: text or json")
	return cmd
}

func (o *options) run(ctx context.Context, w io.Writer, args []string) error {
	if err := o.validate(args); err != nil {
		return err
	}
	if o.config.Context == nil || *o.config.Context == "" {
		return fmt.Errorf("--context is required")
	}
	cs, ns, err := o.client()
	if err != nil {
		return err
	}
	pods, err := o.pods(ctx, cs, ns, args)
	if err != nil {
		return err
	}
	since := time.Now().Add(-o.since)
	var items []timeline.Item
	for i := range pods {
		got, err := collectPod(ctx, cs, &pods[i], since, o)
		if err != nil {
			return err
		}
		items = append(items, got...)
	}
	return o.write(w, timeline.Merge(items, nil))
}

func (o *options) validate(args []string) error {
	if len(args) == 0 && o.selector == "" {
		return fmt.Errorf("give a POD name or -l SELECTOR")
	}
	if len(args) == 1 && o.selector != "" {
		return fmt.Errorf("use POD or -l SELECTOR, not both")
	}
	if o.output != outputText && o.output != outputJSON {
		return fmt.Errorf("--output must be %s or %s", outputText, outputJSON)
	}
	return nil
}

func (o *options) client() (kubernetes.Interface, string, error) {
	rest, err := o.config.ToRESTConfig()
	if err != nil {
		return nil, "", fmt.Errorf("load kubeconfig: %w", err)
	}
	cs, err := kubernetes.NewForConfig(rest)
	if err != nil {
		return nil, "", fmt.Errorf("build client: %w", err)
	}
	ns, _, err := o.config.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, "", fmt.Errorf("resolve namespace: %w", err)
	}
	return cs, ns, nil
}

func (o *options) pods(ctx context.Context, cs kubernetes.Interface, ns string, args []string) ([]corev1.Pod, error) {
	if len(args) == 1 {
		p, err := cs.CoreV1().Pods(ns).Get(ctx, args[0], metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("get pod %s/%s: %w", ns, args[0], err)
		}
		return []corev1.Pod{*p}, nil
	}
	list, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: o.selector})
	if err != nil {
		return nil, fmt.Errorf("list pods with -l %q: %w", o.selector, err)
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("no pods in %s match -l %q", ns, o.selector)
	}
	return list.Items, nil
}

func collectPod(ctx context.Context, cs kubernetes.Interface, pod *corev1.Pod, since time.Time, o *options) ([]timeline.Item, error) {
	targets, err := collect.Targets(ctx, cs, pod)
	if err != nil {
		return nil, err
	}
	events, err := collect.Events(ctx, cs, pod.Namespace, targets, since)
	if err != nil {
		return nil, err
	}
	logs, err := collect.Logs(ctx, cs, pod, collect.LogOptions{Container: o.container, Previous: o.previous, Since: since})
	if err != nil {
		return nil, err
	}
	return timeline.Merge(logs, events), nil
}

func (o *options) write(w io.Writer, items []timeline.Item) error {
	if o.output == outputJSON {
		return format.JSON(w, items)
	}
	useColor := false
	if f, ok := w.(*os.File); ok {
		useColor = term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == ""
	}
	return format.Text(w, items, useColor)
}
