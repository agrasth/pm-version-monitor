package classifier_test

import (
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/classifier"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		version      string
		isPrerelease bool
		want         string
	}{
		{"4.0.0",           false, "ga"},
		{"4.0.0-rc-1",      true,  "rc"},
		{"4.0.0-RC1",       true,  "rc"},
		{"4.0.0-beta.1",    true,  "beta"},
		{"2.0.0b3",         true,  "beta"},
		{"4.0.0-alpha.1",   true,  "alpha"},
		{"4.0.0a1",         true,  "alpha"},
		{"4.0.0-M1",        true,  "milestone"},
		{"4.0.0-milestone", true,  "milestone"},
		{"1.0.0.dev1",      true,  "dev"},
		{"1.0.0-dev",       true,  "dev"},
		{"go1.24rc1",       false, "rc"},
		{"go1.25",          false, "ga"},
		{"9.0-rc-2",        true,  "rc"},
		{"11.0.0",          false, "ga"},
	}

	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			got := classifier.Classify(tc.version, tc.isPrerelease)
			if got != tc.want {
				t.Errorf("Classify(%q, %v) = %q, want %q", tc.version, tc.isPrerelease, got, tc.want)
			}
		})
	}
}

func TestShouldNotify(t *testing.T) {
	allowed := []string{"rc", "ga"}

	cases := []struct {
		releaseType string
		want        bool
	}{
		{"rc", true},
		{"ga", true},
		{"beta", false},
		{"alpha", false},
		{"milestone", false},
	}

	for _, tc := range cases {
		t.Run(tc.releaseType, func(t *testing.T) {
			got := classifier.ShouldNotify(tc.releaseType, allowed)
			if got != tc.want {
				t.Errorf("ShouldNotify(%q, %v) = %v, want %v", tc.releaseType, allowed, got, tc.want)
			}
		})
	}
}

func TestEmoji(t *testing.T) {
	cases := map[string]string{
		"ga":          "✅",
		"rc":          "🔖",
		"beta":        "🔧",
		"alpha":       "🧪",
		"milestone":   "🏁",
		"dev":         "🚧",
		"pre-release": "📦",
		"unknown":     "📦",
	}
	for rt, want := range cases {
		t.Run(rt, func(t *testing.T) {
			got := classifier.Emoji(rt)
			if got != want {
				t.Errorf("Emoji(%q) = %q, want %q", rt, got, want)
			}
		})
	}
}
