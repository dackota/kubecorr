package cmd

import "testing"

func TestValidate_RejectsNoPodAndNoSelector(t *testing.T) {
	o := &options{output: outputText}
	if err := o.validate(nil); err == nil {
		t.Fatal("want error")
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
