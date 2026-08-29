package summary

import (
	"sort"
	"strings"
	"time"

	"github.com/dackota/kubecorr/internal/timeline"
)

// ProbeSummary counts failures of one probe type for one pod.
type ProbeSummary struct {
	Probe    string    `json:"probe"` // readiness, liveness, startup
	Failures int       `json:"failures"`
	Last     time.Time `json:"last"`
}

var probeTypes = []string{"readiness", "liveness", "startup"}

// ProbeFailures sums Unhealthy events for pod, grouped by probe type. It uses
// the event Count so aggregated events are not undercounted. Returns nil
// when there are none.
func ProbeFailures(items []timeline.Item, pod string) []ProbeSummary {
	byProbe := map[string]*ProbeSummary{}
	for _, it := range items {
		if it.Kind != timeline.KindEvent || it.Reason != "Unhealthy" || it.Source != "pod/"+pod {
			continue
		}
		probe := probeType(it.Text)
		if probe == "" {
			continue
		}
		p := byProbe[probe]
		if p == nil {
			p = &ProbeSummary{Probe: probe}
			byProbe[probe] = p
		}
		p.Failures += max(it.Count, 1)
		if it.Time.After(p.Last) {
			p.Last = it.Time
		}
	}
	if len(byProbe) == 0 {
		return nil
	}
	out := make([]ProbeSummary, 0, len(byProbe))
	for _, p := range byProbe {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Probe < out[j].Probe })
	return out
}

func probeType(msg string) string {
	lower := strings.ToLower(msg)
	for _, p := range probeTypes {
		if strings.HasPrefix(lower, p+" probe") {
			return p
		}
	}
	return ""
}
