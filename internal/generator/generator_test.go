package generator_test

import (
	"strings"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/generator"
	"github.com/jfrog/pm-version-monitor/internal/state"
)

func makeTestState() *state.State {
	return &state.State{PMs: map[string]*state.PMState{
		"maven": {
			LatestKnown: "maven-4.0.0-rc-1",
			Versions: map[string]*state.VersionEntry{
				"maven-3.8.8": {
					ReleaseType:  "ga",
					TestStatus:   state.StatusPassed,
					JFCLIVersion: "2.103.0",
					DetectedAt:   "2026-01-01T00:00:00Z",
					TestedAt:     "2026-01-02T00:00:00Z",
					RunURL:       "https://github.com/jfrog/pm-version-monitor/actions/runs/1",
				},
				"maven-4.0.0-rc-1": {
					ReleaseType: "rc",
					TestStatus:  state.StatusFailed,
					DetectedAt:  "2026-05-01T00:00:00Z",
					RunURL:      "https://github.com/jfrog/pm-version-monitor/actions/runs/2",
				},
				"maven-4.0.0-M1": {
					ReleaseType: "milestone",
					TestStatus:  state.StatusPending,
					DetectedAt:  "2026-03-01T00:00:00Z",
				},
			},
		},
		"gradle": {
			LatestKnown: "8.3",
			Versions: map[string]*state.VersionEntry{
				"8.3": {
					ReleaseType:  "ga",
					TestStatus:   state.StatusPassed,
					JFCLIVersion: "2.103.0",
					DetectedAt:   "2026-01-01T00:00:00Z",
				},
			},
		},
	}}
}

func TestGenerateContainsPMs(t *testing.T) {
	s := makeTestState()
	var buf strings.Builder
	if err := generator.Generate(s, &buf); err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	html := buf.String()

	for _, want := range []string{"maven", "gradle", "maven-3.8.8", "maven-4.0.0-rc-1", "2.103.0"} {
		if !strings.Contains(html, want) {
			t.Errorf("generated HTML missing %q", want)
		}
	}
}

func TestGenerateStatusClasses(t *testing.T) {
	s := makeTestState()
	var buf strings.Builder
	generator.Generate(s, &buf)
	html := buf.String()

	if !strings.Contains(html, "status-passing") {
		t.Error("HTML missing status-passing class")
	}
	if !strings.Contains(html, "status-failing") {
		t.Error("HTML missing status-failing class")
	}
	if !strings.Contains(html, "status-pending") {
		t.Error("HTML missing status-pending class")
	}
}

func TestGenerateIsValidHTML(t *testing.T) {
	s := makeTestState()
	var buf strings.Builder
	generator.Generate(s, &buf)
	html := buf.String()

	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("output does not start with DOCTYPE")
	}
	if !strings.Contains(html, "</html>") {
		t.Error("output missing closing </html>")
	}
}

func TestGenerateSummaryStats(t *testing.T) {
	s := makeTestState()
	var buf strings.Builder
	generator.Generate(s, &buf)
	html := buf.String()

	// 2 passing (maven-3.8.8 + gradle-8.3), 1 failing, 1 pending
	if !strings.Contains(html, "2 passing") {
		t.Errorf("HTML missing correct passing count, got:\n%s", html[:500])
	}
	if !strings.Contains(html, "1 failing") {
		t.Error("HTML missing failing count")
	}
	if !strings.Contains(html, "1 pending") {
		t.Error("HTML missing pending count")
	}
}

func TestGenerateEmpty(t *testing.T) {
	s := state.Empty()
	var buf strings.Builder
	if err := generator.Generate(s, &buf); err != nil {
		t.Fatalf("Generate on empty state error: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, "JFrog CLI") {
		t.Error("empty state HTML missing title")
	}
}
