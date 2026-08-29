package summary

import (
	"strings"
	"testing"
	"time"

	"github.com/dackota/kubecorr/internal/timeline"
)

func unhealthy(pod, msg string, count int, sec int) timeline.Item {
	return timeline.Item{Time: time.Unix(int64(sec), 0), Kind: timeline.KindEvent, Type: "Warning",
		Reason: "Unhealthy", Source: "pod/" + pod, Text: msg, Count: count}
}

func TestProbeFailures_CountsByProbeTypeForOnePod(t *testing.T) {
	items := []timeline.Item{
		unhealthy("api-1", "Readiness probe failed: HTTP probe failed with statuscode: 503", 10, 100),
		unhealthy("api-1", "Readiness probe failed: connection refused", 2, 200),
		unhealthy("api-1", "Liveness probe failed: timeout", 1, 150),
		unhealthy("api-1", "Startup probe failed: x", 0, 50), // count 0 counts as 1
		unhealthy("other", "Readiness probe failed: y", 99, 300),
		{Time: time.Unix(1, 0), Kind: timeline.KindEvent, Reason: "BackOff", Source: "pod/api-1"},
	}

	got := ProbeFailures(items, "api-1")

	want := map[string]int{"readiness": 12, "liveness": 1, "startup": 1}
	if len(got) != 3 {
		t.Fatalf("want 3 probe types got %+v", got)
	}
	for _, p := range got {
		if want[p.Probe] != p.Failures {
			t.Errorf("%s: want %d got %d", p.Probe, want[p.Probe], p.Failures)
		}
		if p.Probe == "readiness" && p.Last.Unix() != 200 {
			t.Errorf("readiness last want 200 got %d", p.Last.Unix())
		}
	}
}

func TestProbeFailures_NoneGivesNil(t *testing.T) {
	if got := ProbeFailures(nil, "p"); got != nil {
		t.Fatalf("want nil got %v", got)
	}
}

func TestText_ShowsProbeLine(t *testing.T) {
	s := FromPod(podWithRestarts())
	s.Probes = []ProbeSummary{{Probe: "readiness", Failures: 12, Last: time.Unix(200, 0)}}

	out := Text([]PodSummary{s})

	if !strings.Contains(out, "probes:") || !strings.Contains(out, "readiness failed 12x") {
		t.Fatalf("missing probe line: %q", out)
	}
}
