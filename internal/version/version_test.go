package version_test

import (
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/version"
)

func TestIsNewerThan(t *testing.T) {
	cases := []struct {
		candidate string
		baseline  string
		want      bool
	}{
		// Standard semver
		{"10.9.2", "10.9.1", true},
		{"10.9.1", "10.9.2", false},
		{"10.9.1", "10.9.1", false},
		{"11.0.0", "10.9.9", true},
		// v-prefix stripped
		{"v4.0.0", "v3.9.9", true},
		{"v3.9.9", "v4.0.0", false},
		// Maven milestones / RCs
		{"4.0.0-M2", "4.0.0-M1", true},
		{"4.0.0-M1", "3.9.9", true},
		{"4.0.0-rc-1", "4.0.0-M3", true},
		{"4.0.0", "4.0.0-rc-1", true},  // GA > RC
		// Go versions (go prefix stripped)
		{"go1.24rc1", "go1.23.5", true},
		{"go1.23.5", "go1.24rc1", false},
		{"go1.25", "go1.24.3", true},
		// npm pre-releases
		{"11.0.0-alpha.2", "11.0.0-alpha.1", true},
		{"11.0.0-beta.1", "11.0.0-alpha.5", true},
		{"11.0.0", "11.0.0-beta.1", true},
		// Gradle RC
		{"9.0-rc-2", "9.0-rc-1", true},
		{"9.0-rc-1", "8.14", true},
		{"9.0", "9.0-rc-5", true},
		// New PM not in state (baseline empty)
		{"1.0.0", "", true},
		// Two-digit pre-release numbers (numeric sort, not lexicographic)
		{"4.0.0-M10", "4.0.0-M9", true},
		{"9.0-rc-10", "9.0-rc-9", true},
		{"11.0.0-alpha.10", "11.0.0-alpha.9", true},
		// "build" suffix is NOT a beta
		{"1.0.0-build1", "1.0.0-rc.1", false},
		// Maven short-form beta IS a beta
		{"1.0.0-b2", "1.0.0-alpha.1", true},
		// Package-manager-prefixed tags (normalize strips "maven-", "helm-" etc.)
		{"maven-4.0.0", "maven-3.9.9", true},
		{"maven-3.9.9", "maven-4.0.0", false},
	}

	for _, tc := range cases {
		t.Run(tc.candidate+"_vs_"+tc.baseline, func(t *testing.T) {
			got := version.IsNewerThan(tc.candidate, tc.baseline)
			if got != tc.want {
				t.Errorf("IsNewerThan(%q, %q) = %v, want %v", tc.candidate, tc.baseline, got, tc.want)
			}
		})
	}
}
