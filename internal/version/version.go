package version

import (
	"regexp"
	"strconv"
	"strings"
)

// IsNewerThan reports whether candidate is semantically newer than baseline.
// An empty baseline means the PM is new — candidate is always newer.
func IsNewerThan(candidate, baseline string) bool {
	if baseline == "" {
		return true
	}
	return compare(normalize(candidate), normalize(baseline)) > 0
}

// normalize strips common version prefixes (v, go) and any remaining
// leading non-digit characters (e.g., "maven-", "helm-").
func normalize(v string) string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "go")
	// Strip any remaining leading non-digit prefix.
	for len(v) > 0 && (v[0] < '0' || v[0] > '9') {
		v = v[1:]
	}
	return v
}

type parsed struct {
	numeric    []int
	prerelease string // empty = GA
}

var (
	numericRe     = regexp.MustCompile(`^(\d+)`)
	trailingNumRe = regexp.MustCompile(`(\d+)$`)
	betaShortRe   = regexp.MustCompile(`^-?b\d`)
)

func parseVersion(v string) parsed {
	parts := strings.FieldsFunc(v, func(r rune) bool { return r == '.' })
	var numeric []int
	var prerelease string
	for i, p := range parts {
		m := numericRe.FindString(p)
		if m == "" {
			prerelease = strings.Join(parts[i:], ".")
			break
		}
		n, _ := strconv.Atoi(m)
		numeric = append(numeric, n)
		suffix := strings.TrimPrefix(p, m)
		if suffix != "" {
			prerelease = suffix + "." + strings.Join(parts[i+1:], ".")
			prerelease = strings.Trim(prerelease, ".")
			break
		}
	}
	return parsed{numeric: numeric, prerelease: prerelease}
}

// compare returns -1, 0, or 1.
func compare(a, b string) int {
	if a == b {
		return 0
	}
	ap := parseVersion(a)
	bp := parseVersion(b)

	// Compare numeric parts
	maxLen := len(ap.numeric)
	if len(bp.numeric) > maxLen {
		maxLen = len(bp.numeric)
	}
	for i := 0; i < maxLen; i++ {
		av, bv := 0, 0
		if i < len(ap.numeric) {
			av = ap.numeric[i]
		}
		if i < len(bp.numeric) {
			bv = bp.numeric[i]
		}
		if av != bv {
			if av > bv {
				return 1
			}
			return -1
		}
	}

	// Numeric parts equal — GA (no pre-release) beats any pre-release.
	if ap.prerelease == "" && bp.prerelease != "" {
		return 1
	}
	if ap.prerelease != "" && bp.prerelease == "" {
		return -1
	}
	if ap.prerelease == bp.prerelease {
		return 0
	}

	// Both have pre-release — rank by type, then numeric suffix.
	ar := prereleaseRank(ap.prerelease)
	br := prereleaseRank(bp.prerelease)
	if ar != br {
		if ar > br {
			return 1
		}
		return -1
	}
	// Same rank — compare trailing number
	an := prereleaseNum(ap.prerelease)
	bn := prereleaseNum(bp.prerelease)
	if an > bn {
		return 1
	}
	if an < bn {
		return -1
	}
	return strings.Compare(ap.prerelease, bp.prerelease)
}

// prereleaseRank assigns an ordering: RC > beta > alpha > milestone > dev > unknown.
func prereleaseRank(pre string) int {
	lo := strings.ToLower(pre)
	switch {
	case strings.Contains(lo, "rc"):
		return 4
	case strings.Contains(lo, "beta") || betaShortRe.MatchString(lo):
		return 3
	case strings.Contains(lo, "alpha"):
		return 2
	case strings.Contains(lo, "-m") || strings.HasSuffix(lo, "milestone"):
		return 1
	case strings.Contains(lo, "dev"):
		return 0
	default:
		return 0
	}
}

// prereleaseNum extracts the trailing integer from a pre-release string.
func prereleaseNum(pre string) int {
	m := trailingNumRe.FindString(pre)
	if m == "" {
		return 0
	}
	n, _ := strconv.Atoi(m)
	return n
}
