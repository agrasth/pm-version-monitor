package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/version"
)

// pypiPreReleaseRe matches PEP 440 pre-release markers with boundary awareness.
// Matches: a1, b2, rc1 (pre-release), .dev0/.dev1 (dev releases).
var pypiPreReleaseRe = regexp.MustCompile(`(?i)(a|b|rc)\d|\.dev\d?`)

// PyPISource fetches releases from the PyPI JSON API.
type PyPISource struct {
	baseURL string
	client  *http.Client
}

// NewPyPISource creates a PyPISource using the real PyPI API.
func NewPyPISource() *PyPISource {
	return NewPyPISourceWithBase("https://pypi.org")
}

// NewPyPISourceWithBase creates a PyPISource with an overridden base URL (for testing).
func NewPyPISourceWithBase(baseURL string) *PyPISource {
	return &PyPISource{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type pypiResponse struct {
	Releases map[string][]struct {
		UploadTime string `json:"upload_time"`
	} `json:"releases"`
}

// FetchReleases returns all PyPI releases newer than sinceVersion.
func (p *PyPISource) FetchReleases(sourceID, sinceVersion string) ([]Release, error) {
	url := fmt.Sprintf("%s/pypi/%s/json", p.baseURL, sourceID)

	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching PyPI %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("PyPI returned %d for %s", resp.StatusCode, url)
	}

	var raw pypiResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding PyPI response: %w", err)
	}

	var releases []Release
	for ver, files := range raw.Releases {
		if len(files) == 0 {
			continue
		}
		if !version.IsNewerThan(ver, sinceVersion) {
			continue
		}
		releases = append(releases, Release{
			Version:         ver,
			IsPrerelease:    isPyPIPrerelease(ver),
			PublishedAt:     files[0].UploadTime,
			ReleaseNotesURL: fmt.Sprintf("https://pypi.org/project/%s/%s/", sourceID, ver),
		})
	}
	return releases, nil
}

func isPyPIPrerelease(v string) bool {
	return pypiPreReleaseRe.MatchString(v)
}

// FetchAll returns all PyPI releases published on or after sinceDate ("YYYY-MM-DD").
// If sinceDate is empty, returns all releases regardless of date.
func (p *PyPISource) FetchAll(sourceID, sinceDate string) ([]Release, error) {
	var cutoff time.Time
	if sinceDate != "" {
		var err error
		cutoff, err = time.Parse("2006-01-02", sinceDate)
		if err != nil {
			return nil, fmt.Errorf("parsing sinceDate %q: %w", sinceDate, err)
		}
	}

	url := fmt.Sprintf("%s/pypi/%s/json", p.baseURL, sourceID)
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching PyPI %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("PyPI returned %d for %s", resp.StatusCode, url)
	}

	var raw pypiResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding PyPI response: %w", err)
	}

	var releases []Release
	for ver, files := range raw.Releases {
		if len(files) == 0 {
			continue
		}
		if !cutoff.IsZero() {
			uploadTime, err := time.Parse("2006-01-02T15:04:05", files[0].UploadTime)
			if err == nil && uploadTime.Before(cutoff) {
				continue
			}
		}
		releases = append(releases, Release{
			Version:         ver,
			IsPrerelease:    isPyPIPrerelease(ver),
			PublishedAt:     files[0].UploadTime,
			ReleaseNotesURL: fmt.Sprintf("https://pypi.org/project/%s/%s/", sourceID, ver),
		})
	}
	return releases, nil
}
