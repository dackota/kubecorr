package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/dackota/kubecorr/internal/summary"
)

func init() { lipgloss.SetColorProfile(termenv.ANSI) }

func sampleSummaries() []summary.PodSummary {
	return []summary.PodSummary{{
		Pod: "crasher-1", Node: "n1", Phase: "Running",
		Containers: []summary.ContainerSummary{{Name: "app", Restarts: 7, State: "Waiting: CrashLoopBackOff", LastExitCode: 1, LastExitReason: "Error"}},
		Probes:     []summary.ProbeSummary{{Probe: "readiness", Failures: 44, Last: time.Unix(0, 0)}},
	}, {
		Pod: "ok-1", Node: "n2", Phase: "Running",
		Containers: []summary.ContainerSummary{{Name: "app", Ready: true, State: "Running"}},
	}}
}

func TestRenderSummary_OneLinePerPodPlusContainersAndProbes(t *testing.T) {
	out := stripANSI(renderSummary(sampleSummaries(), 120))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) != 5 {
		t.Fatalf("want 5 lines got %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "crasher-1") || !strings.Contains(lines[0], "n1") {
		t.Fatalf("pod line wrong: %q", lines[0])
	}
	if !strings.Contains(lines[1], "CrashLoopBackOff") || !strings.Contains(lines[1], "restarts 7") || !strings.Contains(lines[1], "Error (1)") {
		t.Fatalf("container line wrong: %q", lines[1])
	}
	if !strings.Contains(lines[2], "readiness 44x") {
		t.Fatalf("probe line wrong: %q", lines[2])
	}
}

func TestRenderSummary_BadStateIsRedHealthyIsNot(t *testing.T) {
	out := renderSummary(sampleSummaries(), 120)
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[1], "\x1b[31m") && !strings.Contains(lines[1], "\x1b[91m") && !strings.Contains(lines[1], ";31m") {
		t.Fatalf("crash line has no red: %q", lines[1])
	}
	if strings.Contains(lines[4], "\x1b[31m") || strings.Contains(lines[4], ";31m") || strings.Contains(lines[4], "\x1b[91m") {
		t.Fatalf("healthy line is red: %q", lines[4])
	}
}

func TestRenderSummary_EmptyGivesEmpty(t *testing.T) {
	if renderSummary(nil, 80) != "" {
		t.Fatal("want empty")
	}
}
