// Package cmd holds the kubecorr command line.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/dackota/kubecorr/internal/collect"
	"github.com/dackota/kubecorr/internal/format"
	"github.com/dackota/kubecorr/internal/summary"
	"github.com/dackota/kubecorr/internal/timeline"
	"github.com/dackota/kubecorr/internal/tui"
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
	tui       bool
	grep      string
	grepRE    *regexp.Regexp
	follow    bool
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

Give a pod name, use -l to pick pods by label, or give nothing to see
every pod in the namespace.`,
		Example: `  kubecorr -n api
  kubecorr --context prod -n api api-7d9f-abc
  kubecorr --context prod -n api -l app=api --since 30m
  kubecorr --context prod -n api api-7d9f-abc --previous -o json | jq .
  kubecorr -n api --grep 'panic|timeout'
  kubecorr -n api --tui
  kubecorr -n api -f --tui`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(cmd.Context(), cmd.OutOrStdout(), args)
		},
	}
	o.config.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.selector, "selector", "l", "", "label selector to pick pods (default: all pods in the namespace)")
	cmd.Flags().StringVarP(&o.container, "container", "c", "", "only this container (default: all)")
	cmd.Flags().BoolVarP(&o.previous, "previous", "p", false, "read logs of the last terminated container")
	cmd.Flags().DurationVar(&o.since, "since", defaultSince, "how far back to look")
	cmd.Flags().StringVarP(&o.output, "output", "o", outputText, "output format: text or json")
	cmd.Flags().BoolVarP(&o.tui, "tui", "t", false, "open a side by side view of logs and events")
	cmd.Flags().StringVar(&o.grep, "grep", "", "only keep log lines that match this regular expression")
	cmd.Flags().BoolVarP(&o.follow, "follow", "f", false, "keep running and show new logs and events as they happen")
	return cmd
}

func (o *options) run(ctx context.Context, w io.Writer, args []string) error {
	if err := o.validate(args); err != nil {
		return err
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
	var summaries []summary.PodSummary
	var streamTargets []collect.StreamTarget
	for i := range pods {
		got, targets, err := collectPod(ctx, cs, &pods[i], since, o)
		if err != nil {
			return err
		}
		items = append(items, got...)
		summaries = append(summaries, summary.FromPod(&pods[i]))
		streamTargets = append(streamTargets, collect.StreamTarget{Pod: &pods[i], Targets: targets})
	}
	items = filterLogs(items, o.grepRE)
	var stream <-chan timeline.Item
	if o.follow {
		stream = collect.Stream(ctx, cs, streamTargets, collect.LogOptions{Container: o.container})
	}
	if o.tui {
		return o.runTUI(ctx, items, summaries, ns, stream)
	}
	if err := o.write(w, timeline.Merge(items, nil), summaries); err != nil || stream == nil {
		return err
	}
	return o.writeStream(w, stream)
}

func (o *options) validate(args []string) error {
	if len(args) == 1 && o.selector != "" {
		return fmt.Errorf("use POD or -l SELECTOR, not both")
	}
	if o.output != outputText && o.output != outputJSON {
		return fmt.Errorf("--output must be %s or %s", outputText, outputJSON)
	}
	if o.grep != "" {
		re, err := regexp.Compile(o.grep)
		if err != nil {
			return fmt.Errorf("bad --grep: %w", err)
		}
		o.grepRE = re
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
		if o.selector == "" {
			return nil, fmt.Errorf("no pods in namespace %s", ns)
		}
		return nil, fmt.Errorf("no pods in %s match -l %q", ns, o.selector)
	}
	return list.Items, nil
}

// collectPod gathers logs, Events, pod status items and node status items.
func collectPod(ctx context.Context, cs kubernetes.Interface, pod *corev1.Pod, since time.Time, o *options) ([]timeline.Item, []collect.Target, error) {
	targets, err := collect.Targets(ctx, cs, pod)
	if err != nil {
		return nil, nil, err
	}
	events, err := collect.Events(ctx, cs, pod.Namespace, targets, since)
	if err != nil {
		return nil, nil, err
	}
	events = append(events, collect.StatusItems(pod, since)...)
	nodeItems, err := collect.NodeItems(ctx, cs, pod.Spec.NodeName, since)
	if err != nil {
		return nil, nil, err
	}
	events = append(events, nodeItems...)
	logs, err := collect.Logs(ctx, cs, pod, collect.LogOptions{Container: o.container, Previous: o.previous, Since: since})
	if err != nil {
		return nil, nil, err
	}
	return timeline.Merge(logs, events), targets, nil
}

// filterLogs drops log lines that do not match re. Events always stay.
// A nil re keeps everything.
func filterLogs(items []timeline.Item, re *regexp.Regexp) []timeline.Item {
	if re == nil {
		return items
	}
	out := make([]timeline.Item, 0, len(items))
	for _, it := range items {
		if it.Kind == timeline.KindEvent || re.MatchString(it.Text) {
			out = append(out, it)
		}
	}
	return out
}

func (o *options) runTUI(ctx context.Context, items []timeline.Item, summaries []summary.PodSummary, ns string, stream <-chan timeline.Item) error {
	var logs, events []timeline.Item
	for _, it := range items {
		if it.Kind == timeline.KindEvent {
			events = append(events, it)
		} else {
			logs = append(logs, it)
		}
	}
	ctxName := ""
	if o.config.Context != nil {
		ctxName = *o.config.Context
	}
	m := tui.New(timeline.Merge(logs, nil), timeline.Merge(events, nil), ns, ctxName).
		WithSummary(summary.Text(summaries)).
		WithFollow(stream != nil).
		WithStream(ctx, stream)
	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx)).Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func (o *options) write(w io.Writer, items []timeline.Item, summaries []summary.PodSummary) error {
	if o.output == outputJSON {
		return format.JSON(w, items)
	}
	useColor := false
	if f, ok := w.(*os.File); ok {
		useColor = term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == ""
	}
	if s := summary.Text(summaries); s != "" {
		if _, err := fmt.Fprint(w, s, "\n"); err != nil {
			return fmt.Errorf("write summary: %w", err)
		}
	}
	return format.Text(w, items, useColor)
}

// writeStream prints items as they arrive until the stream closes.
// Lines print in arrival order, which is close to, but not always, time order.
func (o *options) writeStream(w io.Writer, stream <-chan timeline.Item) error {
	useColor := false
	if f, ok := w.(*os.File); ok {
		useColor = term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == ""
	}
	for it := range stream {
		if it.Kind == timeline.KindLog && o.grepRE != nil && !o.grepRE.MatchString(it.Text) {
			continue
		}
		if o.output == outputJSON {
			if err := format.JSONLine(w, it); err != nil {
				return err
			}
			continue
		}
		if err := format.Text(w, []timeline.Item{it}, useColor); err != nil {
			return err
		}
	}
	return nil
}
