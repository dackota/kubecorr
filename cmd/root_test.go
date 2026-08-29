package cmd

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
