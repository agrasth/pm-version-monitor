package monitor_test

import (
	"errors"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/config"
	"github.com/jfrog/pm-version-monitor/internal/monitor"
	"github.com/jfrog/pm-version-monitor/internal/notify"
	"github.com/jfrog/pm-version-monitor/internal/sources"
	"github.com/jfrog/pm-version-monitor/internal/state"
)

// stubSource returns a fixed list of releases.
type stubSource struct {
	releases []sources.Release
	err      error
}

func (s *stubSource) FetchReleases(sourceID, sinceVersion string) ([]sources.Release, error) {
	return s.releases, s.err
}

func (s *stubSource) FetchAll(sourceID, sinceDate string) ([]sources.Release, error) {
	return s.releases, s.err
}

// stubNotifier records which notifications were sent.
type stubNotifier struct {
	sent []notify.Notification
	err  error
}

func (n *stubNotifier) Send(notif notify.Notification) (string, error) {
	n.sent = append(n.sent, notif)
	return "ts-" + notif.Release.Version, n.err
}

func makeCfg(pms []config.PackageManager) *config.Config {
	return &config.Config{
		Settings:        config.Settings{AutoTestAfterHours: 4, SlackChannel: "#pm-releases"},
		PackageManagers: pms,
	}
}

func TestRunDetectsNewVersions(t *testing.T) {
	cfg := makeCfg([]config.PackageManager{
		{Name: "maven", SourceType: "github_releases", SourceID: "apache/maven",
			NotifyReleaseTypes: []string{"rc", "ga"}, Enabled: true},
	})
	initialState := &state.State{PMs: map[string]*state.PMState{
		"maven": {LatestKnown: "maven-3.9.9", Versions: map[string]*state.VersionEntry{}},
	}}

	stub := &stubSource{releases: []sources.Release{
		{Version: "maven-4.0.0-rc-1", IsPrerelease: true, PublishedAt: "2026-05-01T00:00:00Z",
			ReleaseNotesURL: "https://example.com"},
	}}
	notifier := &stubNotifier{}

	newState, err := monitor.Run(cfg, initialState, notifier, monitor.WithSource("github_releases", stub))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if len(notifier.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.sent))
	}
	if notifier.sent[0].PM != "maven" {
		t.Errorf("notification PM = %q, want maven", notifier.sent[0].PM)
	}
	if state.GetLastKnown(newState, "maven") != "maven-4.0.0-rc-1" {
		t.Errorf("LatestKnown = %q, want maven-4.0.0-rc-1", state.GetLastKnown(newState, "maven"))
	}
	entry := newState.PMs["maven"].Versions["maven-4.0.0-rc-1"]
	if entry == nil {
		t.Fatal("version entry not in state")
	}
	if entry.TestStatus != state.StatusPending {
		t.Errorf("TestStatus = %q, want %q", entry.TestStatus, state.StatusPending)
	}

	// Deep-clone guarantee: input state must not be mutated.
	if state.GetLastKnown(initialState, "maven") != "maven-3.9.9" {
		t.Errorf("Run() mutated input LatestKnown: got %q, want maven-3.9.9",
			state.GetLastKnown(initialState, "maven"))
	}
	if initialState.PMs["maven"].Versions["maven-4.0.0-rc-1"] != nil {
		t.Error("Run() mutated input state: new version entry written back to original")
	}
}

func TestRunSkipsDisabledPMs(t *testing.T) {
	cfg := makeCfg([]config.PackageManager{
		{Name: "maven", SourceType: "github_releases", Enabled: false},
	})
	stub := &stubSource{releases: []sources.Release{{Version: "4.0.0"}}}
	notifier := &stubNotifier{}

	_, err := monitor.Run(cfg, state.Empty(), notifier, monitor.WithSource("github_releases", stub))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(notifier.sent) != 0 {
		t.Errorf("expected 0 notifications for disabled PM, got %d", len(notifier.sent))
	}
}

func TestRunSkipsFilteredReleaseTypes(t *testing.T) {
	cfg := makeCfg([]config.PackageManager{
		{Name: "gradle", SourceType: "github_releases", NotifyReleaseTypes: []string{"rc", "ga"}, Enabled: true},
	})
	initialState := &state.State{PMs: map[string]*state.PMState{
		"gradle": {LatestKnown: "8.14", Versions: map[string]*state.VersionEntry{}},
	}}
	stub := &stubSource{releases: []sources.Release{
		{Version: "9.0-alpha-1", IsPrerelease: true},
	}}
	notifier := &stubNotifier{}

	_, err := monitor.Run(cfg, initialState, notifier, monitor.WithSource("github_releases", stub))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(notifier.sent) != 0 {
		t.Errorf("expected 0 notifications for filtered type, got %d", len(notifier.sent))
	}
}

func TestRunSeedsNewPMWithoutNotifying(t *testing.T) {
	cfg := makeCfg([]config.PackageManager{
		{Name: "helm", SourceType: "github_releases", NotifyReleaseTypes: []string{"rc", "ga"}, Enabled: true},
	})
	// State has no entry for helm yet
	stub := &stubSource{releases: []sources.Release{
		{Version: "v4.0.0", IsPrerelease: false},
		{Version: "v4.0.0-rc.1", IsPrerelease: true},
	}}
	notifier := &stubNotifier{}

	newState, err := monitor.Run(cfg, state.Empty(), notifier, monitor.WithSource("github_releases", stub))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	// No notification on first run — seed only
	if len(notifier.sent) != 0 {
		t.Errorf("expected 0 notifications for new PM seed, got %d", len(notifier.sent))
	}
	// But state is seeded with the semantically newest version
	seeded := state.GetLastKnown(newState, "helm")
	if seeded != "v4.0.0" {
		t.Errorf("expected newest version v4.0.0 seeded, got %q", seeded)
	}
}

func TestRunNotifyFailureDoesNotUpdateState(t *testing.T) {
	cfg := makeCfg([]config.PackageManager{
		{Name: "pip", SourceType: "pypi", NotifyReleaseTypes: []string{"rc", "ga"}, Enabled: true},
	})
	initialState := &state.State{PMs: map[string]*state.PMState{
		"pip": {LatestKnown: "25.1.0", Versions: map[string]*state.VersionEntry{}},
	}}
	stub := &stubSource{releases: []sources.Release{{Version: "25.2.0rc1", IsPrerelease: true}}}
	notifier := &stubNotifier{err: errors.New("slack unavailable")}

	newState, err := monitor.Run(cfg, initialState, notifier, monitor.WithSource("pypi", stub))
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	// State must NOT be updated when notification fails
	if state.GetLastKnown(newState, "pip") != "25.1.0" {
		t.Errorf("LatestKnown changed after notify failure, got %q", state.GetLastKnown(newState, "pip"))
	}
}

func TestRunSourceFailureContinuesOtherPMs(t *testing.T) {
	cfg := makeCfg([]config.PackageManager{
		{Name: "broken", SourceType: "github_releases", NotifyReleaseTypes: []string{"ga"}, Enabled: true},
		{Name: "working", SourceType: "pypi", NotifyReleaseTypes: []string{"ga"}, Enabled: true},
	})
	initialState := &state.State{PMs: map[string]*state.PMState{
		"broken":  {LatestKnown: "1.0.0", Versions: map[string]*state.VersionEntry{}},
		"working": {LatestKnown: "1.0.0", Versions: map[string]*state.VersionEntry{}},
	}}
	brokenSrc := &stubSource{err: errors.New("network error")}
	workingSrc := &stubSource{releases: []sources.Release{{Version: "2.0.0", IsPrerelease: false}}}
	notifier := &stubNotifier{}

	_, err := monitor.Run(cfg, initialState, notifier,
		monitor.WithSource("github_releases", brokenSrc),
		monitor.WithSource("pypi", workingSrc),
	)
	// Run() should NOT return an error for partial source failure
	if err != nil {
		t.Fatalf("Run() returned error on partial failure: %v", err)
	}
	// working PM should still get notified
	if len(notifier.sent) != 1 || notifier.sent[0].PM != "working" {
		t.Errorf("expected 1 notification for working PM, got %d", len(notifier.sent))
	}
}
