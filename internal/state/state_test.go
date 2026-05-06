package state_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/state"
)

func TestLoadEmpty(t *testing.T) {
	f, err := os.CreateTemp("", "pm-state-*.json")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString("{}"); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	s, err := state.Load(f.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if s == nil {
		t.Fatal("Load() returned nil state")
	}
}

func TestLoadMissing(t *testing.T) {
	// Missing file should return empty state, not an error.
	s, err := state.Load("/nonexistent/state.json")
	if err != nil {
		t.Fatalf("Load() of missing file error: %v", err)
	}
	if s == nil {
		t.Fatal("Load() returned nil for missing file")
	}
}

func TestGetLastKnown(t *testing.T) {
	s := &state.State{
		PMs: map[string]*state.PMState{
			"maven": {LatestKnown: "3.9.9"},
		},
	}
	if got := state.GetLastKnown(s, "maven"); got != "3.9.9" {
		t.Errorf("GetLastKnown(maven) = %q, want 3.9.9", got)
	}
	if got := state.GetLastKnown(s, "unknown-pm"); got != "" {
		t.Errorf("GetLastKnown(unknown) = %q, want empty", got)
	}
}

func TestAddVersion(t *testing.T) {
	s := &state.State{PMs: map[string]*state.PMState{}}

	state.AddVersion(s, "maven", "4.0.0-M1", "milestone", "2026-05-01T00:00:00Z")

	pm := s.PMs["maven"]
	if pm == nil {
		t.Fatal("pm state is nil after AddVersion")
	}
	if pm.LatestKnown != "4.0.0-M1" {
		t.Errorf("LatestKnown = %q, want 4.0.0-M1", pm.LatestKnown)
	}
	entry := pm.Versions["4.0.0-M1"]
	if entry.ReleaseType != "milestone" {
		t.Errorf("ReleaseType = %q, want milestone", entry.ReleaseType)
	}
	if entry.TestStatus != state.StatusPending {
		t.Errorf("TestStatus = %q, want %q", entry.TestStatus, state.StatusPending)
	}
}

func TestSeedVersion(t *testing.T) {
	s := &state.State{PMs: map[string]*state.PMState{}}

	// Seeding should set LatestKnown but NOT create a pending version entry.
	state.SeedVersion(s, "maven", "3.9.9")

	pm := s.PMs["maven"]
	if pm.LatestKnown != "3.9.9" {
		t.Errorf("LatestKnown = %q, want 3.9.9", pm.LatestKnown)
	}
	if len(pm.Versions) != 0 {
		t.Errorf("expected no version entries after seed, got %d", len(pm.Versions))
	}
}

func TestUpdateTestStatus(t *testing.T) {
	s := &state.State{PMs: map[string]*state.PMState{
		"gradle": {
			LatestKnown: "9.0-rc-1",
			Versions: map[string]*state.VersionEntry{
				"9.0-rc-1": {TestStatus: state.StatusPending},
			},
		},
	}}

	err := state.UpdateTestStatus(s, "gradle", "9.0-rc-1", state.StatusPassed, "https://github.com/run/1")
	if err != nil {
		t.Fatalf("UpdateTestStatus error: %v", err)
	}
	if s.PMs["gradle"].Versions["9.0-rc-1"].TestStatus != state.StatusPassed {
		t.Error("TestStatus not updated to passed")
	}
	if s.PMs["gradle"].Versions["9.0-rc-1"].RunURL != "https://github.com/run/1" {
		t.Error("RunURL not set")
	}
}

func TestSaveAndReload(t *testing.T) {
	s := &state.State{PMs: map[string]*state.PMState{}}
	state.AddVersion(s, "npm", "11.0.0-alpha.1", "alpha", "2026-05-01T00:00:00Z")

	f, err := os.CreateTemp("", "pm-state-*.json")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := state.Save(f.Name(), s); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	s2, err := state.Load(f.Name())
	if err != nil {
		t.Fatalf("Load() after save error: %v", err)
	}
	if state.GetLastKnown(s2, "npm") != "11.0.0-alpha.1" {
		t.Error("reloaded state lost npm version")
	}
	// Verify JSON is valid
	data, _ := os.ReadFile(f.Name())
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Errorf("saved JSON is invalid: %v", err)
	}
}

func TestUpdateTestStatusErrors(t *testing.T) {
	s := state.Empty()

	// PM not in state
	err := state.UpdateTestStatus(s, "nonexistent", "1.0.0", state.StatusPassed, "")
	if err == nil {
		t.Error("expected error for unknown PM, got nil")
	}

	// Version not in state
	state.SeedVersion(s, "maven", "3.9.9")
	err = state.UpdateTestStatus(s, "maven", "4.0.0-M1", state.StatusPassed, "")
	if err == nil {
		t.Error("expected error for unknown version, got nil")
	}

	// Invalid status
	state.AddVersion(s, "gradle", "9.0-rc-1", "rc", "2026-05-01T00:00:00Z")
	err = state.UpdateTestStatus(s, "gradle", "9.0-rc-1", "INVALID", "")
	if err == nil {
		t.Error("expected error for invalid status, got nil")
	}
}

func TestPendingVersionsOlderThan(t *testing.T) {
	now := time.Now()
	old := now.Add(-5 * time.Hour).Format(time.RFC3339)
	recent := now.Add(-1 * time.Hour).Format(time.RFC3339)

	s := &state.State{PMs: map[string]*state.PMState{
		"maven": {Versions: map[string]*state.VersionEntry{
			"4.0.0-M1": {TestStatus: state.StatusPending, DetectedAt: old},
			"4.0.0-M2": {TestStatus: state.StatusPending, DetectedAt: recent},
			"3.9.9":     {TestStatus: state.StatusPassed,  DetectedAt: old},
		}},
	}}

	pending := state.PendingVersionsOlderThan(s, 4)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending old version, got %d", len(pending))
	}
	if pending[0].PM != "maven" || pending[0].Version != "4.0.0-M1" {
		t.Errorf("unexpected pending entry: %+v", pending[0])
	}
}
