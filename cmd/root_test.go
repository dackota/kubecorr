package cmd

import (
	"context"
	"regexp"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/dackota/kubecorr/internal/timeline"
)

func TestValidate_AllowsNoPodAndNoSelector(t *testing.T) {
	o := &options{output: outputText}
	if err := o.validate(nil); err != nil {
		t.Fatal(err)
	}
}

func TestValidate_RejectsPodAndSelectorTogether(t *testing.T) {
	o := &options{output: outputText, selector: "a=b"}
	if err := o.validate([]string{"pod"}); err == nil {
		t.Fatal("want error")
	}
}

func TestValidate_RejectsBadOutput(t *testing.T) {
	o := &options{output: "yaml"}
	if err := o.validate([]string{"pod"}); err == nil {
		t.Fatal("want error")
	}
}

func TestValidate_AcceptsPodWithJSON(t *testing.T) {
	o := &options{output: outputJSON}
	if err := o.validate([]string{"pod"}); err != nil {
		t.Fatal(err)
	}
}

func TestPods_NoArgsAndNoSelectorListsWholeNamespace(t *testing.T) {
	cs := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "ns1"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "ns1"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "c", Namespace: "other"}},
	)
	o := &options{output: outputText}

	got, err := o.pods(context.Background(), cs, "ns1", nil)

	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 pods got %d", len(got))
	}
}

func TestFilterLogs_KeepsEventsAndMatchingLogs(t *testing.T) {
	items := []timeline.Item{
		{Kind: timeline.KindLog, Text: "panic: boom"},
		{Kind: timeline.KindLog, Text: "GET /health"},
		{Kind: timeline.KindEvent, Text: "whatever"},
	}
	got := filterLogs(items, regexp.MustCompile("panic"))
	if len(got) != 2 || got[0].Text != "panic: boom" || got[1].Kind != timeline.KindEvent {
		t.Fatalf("got %+v", got)
	}
	if n := len(filterLogs(items, nil)); n != 3 {
		t.Fatalf("nil re should keep all, got %d", n)
	}
}

func TestValidate_RejectsBadGrep(t *testing.T) {
	o := &options{output: outputText, grep: "("}
	if err := o.validate(nil); err == nil {
		t.Fatal("want error")
	}
}

func TestUsePlain_FalseOnATerminalWithDefaults(t *testing.T) {
	o := &options{output: outputText}
	if o.usePlain(true) {
		t.Fatal("want the TUI by default on a terminal")
	}
}

func TestUsePlain_TrueWhenFlagSet(t *testing.T) {
	o := &options{output: outputText, plain: true}
	if !o.usePlain(true) {
		t.Fatal("want plain when --plain is set")
	}
}

func TestUsePlain_TrueWhenOutputIsJSON(t *testing.T) {
	o := &options{output: outputJSON}
	if !o.usePlain(true) {
		t.Fatal("want plain when -o json")
	}
}

func TestUsePlain_TrueWhenNotATerminal(t *testing.T) {
	o := &options{output: outputText}
	if !o.usePlain(false) {
		t.Fatal("want plain when stdout is not a terminal")
	}
}
