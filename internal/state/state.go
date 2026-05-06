package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// TestStatus values for VersionEntry.
const (
	StatusPending = "pending"
	StatusPassed  = "passed"
	StatusFailed  = "failed"
)

// State is the full content of pm-versions-state.json.
type State struct {
	PMs map[string]*PMState `json:"pms"`
}

// PMState holds the tracking state for one package manager.
type PMState struct {
	LatestKnown string                   `json:"latest_known"`
	Versions    map[string]*VersionEntry `json:"versions"`
}

// VersionEntry records what we know about one specific release.
type VersionEntry struct {
	ReleaseType string `json:"release_type"`
	DetectedAt  string `json:"detected_at"`
	TestStatus  string `json:"test_status"` // pending | passed | failed
	TestedAt    string `json:"tested_at,omitempty"`
	RunURL      string `json:"run_url,omitempty"`
	AnalysisURL string `json:"analysis_url,omitempty"`
}

// PendingVersion is a (PM, version) pair returned by PendingVersionsOlderThan.
type PendingVersion struct {
	PM      string
	Version string
	Entry   *VersionEntry
}

// Load reads pm-versions-state.json. Returns an empty State if the file does not exist.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return empty(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state %s: %w", path, err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state %s: %w", path, err)
	}
	if s.PMs == nil {
		s.PMs = map[string]*PMState{}
	}
	return &s, nil
}

// Save writes state to path atomically (write to temp, rename).
func Save(path string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing state tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// Empty returns a new empty State.
func Empty() *State {
	return empty()
}

// GetLastKnown returns the latest_known version for a PM, or "" if unknown.
func GetLastKnown(s *State, pm string) string {
	if p, ok := s.PMs[pm]; ok {
		return p.LatestKnown
	}
	return ""
}

// AddVersion records a newly detected version as pending and updates latest_known.
func AddVersion(s *State, pm, version, releaseType, detectedAt string) {
	ensurePM(s, pm)
	s.PMs[pm].LatestKnown = version
	s.PMs[pm].Versions[version] = &VersionEntry{
		ReleaseType: releaseType,
		DetectedAt:  detectedAt,
		TestStatus:  StatusPending,
	}
}

// SeedVersion sets latest_known for a PM without creating a version entry.
// Used on first run for a new PM to avoid notifying for historical versions.
func SeedVersion(s *State, pm, version string) {
	ensurePM(s, pm)
	s.PMs[pm].LatestKnown = version
}

// UpdateTestStatus sets the test result for a specific version.
func UpdateTestStatus(s *State, pm, version, status, runURL string) error {
	if status != StatusPending && status != StatusPassed && status != StatusFailed {
		return fmt.Errorf("invalid test status %q: must be pending, passed, or failed", status)
	}
	p, ok := s.PMs[pm]
	if !ok {
		return fmt.Errorf("pm %q not in state", pm)
	}
	entry, ok := p.Versions[version]
	if !ok {
		return fmt.Errorf("version %q of %q not in state", version, pm)
	}
	entry.TestStatus = status
	entry.RunURL = runURL
	entry.TestedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}

// PendingVersionsOlderThan returns all pending versions detected more than olderThanHours ago.
func PendingVersionsOlderThan(s *State, olderThanHours int) []PendingVersion {
	cutoff := time.Now().UTC().Add(-time.Duration(olderThanHours) * time.Hour)
	var result []PendingVersion
	for pm, pmState := range s.PMs {
		for version, entry := range pmState.Versions {
			if entry.TestStatus != StatusPending {
				continue
			}
			t, err := time.Parse(time.RFC3339, entry.DetectedAt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[pm-monitor] WARN: skipping %s %s — malformed DetectedAt %q: %v\n",
					pm, version, entry.DetectedAt, err)
				continue
			}
			if t.Before(cutoff) {
				result = append(result, PendingVersion{PM: pm, Version: version, Entry: entry})
			}
		}
	}
	return result
}

func empty() *State {
	return &State{PMs: map[string]*PMState{}}
}

func ensurePM(s *State, pm string) {
	if _, ok := s.PMs[pm]; !ok {
		s.PMs[pm] = &PMState{Versions: map[string]*VersionEntry{}}
	}
}
