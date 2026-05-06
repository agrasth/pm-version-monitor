package config_test

import (
	"os"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/config"
)

const testYAML = `
settings:
  auto_test_after_hours: 4
  slack_channel: "#pm-releases"

package_managers:
  - name: maven
    source_type: github_releases
    source_id: apache/maven
    notify_release_types: [milestone, rc, ga]
    discover_range: "2022-01-01"
    enabled: true
  - name: pip
    source_type: pypi
    source_id: pip
    notify_release_types: [alpha, beta, rc, ga]
    enabled: false
`

func TestLoad(t *testing.T) {
	f, err := os.CreateTemp("", "pm-config-*.yml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(testYAML); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	cfg, err := config.Load(f.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Settings.AutoTestAfterHours != 4 {
		t.Errorf("AutoTestAfterHours = %d, want 4", cfg.Settings.AutoTestAfterHours)
	}
	if cfg.Settings.SlackChannel != "#pm-releases" {
		t.Errorf("SlackChannel = %q, want %q", cfg.Settings.SlackChannel, "#pm-releases")
	}
	if len(cfg.PackageManagers) != 2 {
		t.Fatalf("len(PackageManagers) = %d, want 2", len(cfg.PackageManagers))
	}

	maven := cfg.PackageManagers[0]
	if maven.Name != "maven" {
		t.Errorf("Name = %q, want maven", maven.Name)
	}
	if maven.SourceType != "github_releases" {
		t.Errorf("SourceType = %q, want github_releases", maven.SourceType)
	}
	if maven.SourceID != "apache/maven" {
		t.Errorf("SourceID = %q, want apache/maven", maven.SourceID)
	}
	if !maven.Enabled {
		t.Error("Enabled = false, want true")
	}
	if len(maven.NotifyReleaseTypes) != 3 {
		t.Errorf("NotifyReleaseTypes len = %d, want 3", len(maven.NotifyReleaseTypes))
	}
	if maven.DiscoverRange != "2022-01-01" {
		t.Errorf("DiscoverRange = %q, want 2022-01-01", maven.DiscoverRange)
	}

	pip := cfg.PackageManagers[1]
	if pip.Enabled {
		t.Error("pip Enabled = true, want false")
	}
	if pip.DiscoverRange != "" {
		t.Errorf("DiscoverRange for pip (unset) = %q, want empty", pip.DiscoverRange)
	}

	// DisplayName not set — should fall back to Name
	if got := maven.DisplayNameOrName(); got != "maven" {
		t.Errorf("DisplayNameOrName() with no display name = %q, want %q", got, "maven")
	}

	// Simulate a PM with display_name set
	pm := config.PackageManager{Name: "nuget", DisplayName: "NuGet / .NET SDK"}
	if got := pm.DisplayNameOrName(); got != "NuGet / .NET SDK" {
		t.Errorf("DisplayNameOrName() with display name = %q, want %q", got, "NuGet / .NET SDK")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := config.Load("/nonexistent/path.yml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
