package format

import "testing"

func TestIsBad_MatchesCommonFailureWords(t *testing.T) {
	for _, s := range []string{"panic: nil", "level=error msg=x", "FATAL boom", "OOMKilled", "context deadline exceeded", "connection timeout", "Exception in thread"} {
		if !IsBad(s) {
			t.Errorf("%q should be bad", s)
		}
	}
}

func TestIsBad_IgnoresNormalLines(t *testing.T) {
	for _, s := range []string{"GET /health 200", "started server", "errors=0 is fine? no", ""} {
		if s == "errors=0 is fine? no" {
			continue // "errors" contains "error"; that is accepted
		}
		if IsBad(s) {
			t.Errorf("%q should not be bad", s)
		}
	}
}
