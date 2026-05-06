package classifier

import "strings"

// Classify determines the release type from a version string and GitHub prerelease flag.
// Returns one of: "ga", "rc", "beta", "alpha", "milestone", "dev", "pre-release".
func Classify(version string, isPrerelease bool) string {
	v := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(version, "v"), "go"))

	switch {
	case contains(v, "rc", "-rc", ".rc"):
		return "rc"
	case contains(v, "beta", "-beta", "-b.") || hasBetaSuffix(v):
		return "beta"
	case contains(v, "alpha", "-alpha") || hasAlphaSuffix(v):
		return "alpha"
	case strings.Contains(v, ".dev") || strings.Contains(v, "-dev"):
		return "dev"
	case strings.Contains(v, "milestone") || milestoneSuffix(v):
		return "milestone"
	}

	if isPrerelease {
		return "pre-release"
	}
	return "ga"
}

// ShouldNotify reports whether releaseType is in the allowed list.
func ShouldNotify(releaseType string, allowed []string) bool {
	for _, a := range allowed {
		if a == releaseType {
			return true
		}
	}
	return false
}

// Emoji returns the display emoji for a release type.
func Emoji(releaseType string) string {
	switch releaseType {
	case "ga":
		return "✅"
	case "rc":
		return "🔖"
	case "beta":
		return "🔧"
	case "alpha":
		return "🧪"
	case "milestone":
		return "🏁"
	case "dev":
		return "🚧"
	default:
		return "📦"
	}
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func hasBetaSuffix(v string) bool {
	// Match trailing "b" followed by digits e.g. "2.0.0b3"
	if len(v) < 2 {
		return false
	}
	i := len(v) - 1
	for i >= 0 && v[i] >= '0' && v[i] <= '9' {
		i--
	}
	return i >= 0 && v[i] == 'b' && i < len(v)-1
}

func hasAlphaSuffix(v string) bool {
	// Match trailing "a" followed by digits e.g. "4.0.0a1"
	if len(v) < 2 {
		return false
	}
	i := len(v) - 1
	for i >= 0 && v[i] >= '0' && v[i] <= '9' {
		i--
	}
	return i >= 0 && v[i] == 'a' && i < len(v)-1
}

func milestoneSuffix(v string) bool {
	// Maven pattern: -M1, -M2, etc. (case-insensitive, already lowercased)
	idx := strings.LastIndex(v, "-m")
	if idx == -1 {
		return false
	}
	rest := v[idx+2:]
	if len(rest) == 0 {
		return false
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
