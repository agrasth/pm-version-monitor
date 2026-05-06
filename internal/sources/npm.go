package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/version"
)

// NpmSource fetches releases from the npm registry.
type NpmSource struct {
	baseURL string
	client  *http.Client
}

// NewNpmSource creates an NpmSource using the real npm registry.
func NewNpmSource() *NpmSource {
	return NewNpmSourceWithBase("https://registry.npmjs.org")
}

// NewNpmSourceWithBase creates an NpmSource with an overridden base URL (for testing).
func NewNpmSourceWithBase(baseURL string) *NpmSource {
	return &NpmSource{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type npmResponse struct {
	Versions map[string]interface{} `json:"versions"`
	Time     map[string]string      `json:"time"`
}

// FetchReleases returns all npm registry releases newer than sinceVersion.
func (n *NpmSource) FetchReleases(sourceID, sinceVersion string) ([]Release, error) {
	url := fmt.Sprintf("%s/%s", n.baseURL, sourceID)

	resp, err := n.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching npm %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("npm registry returned %d for %s", resp.StatusCode, url)
	}

	var raw npmResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding npm response: %w", err)
	}

	var releases []Release
	for ver := range raw.Versions {
		if !version.IsNewerThan(ver, sinceVersion) {
			continue
		}
		releases = append(releases, Release{
			Version:         ver,
			IsPrerelease:    strings.Contains(ver, "-"),
			PublishedAt:     raw.Time[ver],
			ReleaseNotesURL: fmt.Sprintf("https://www.npmjs.com/package/%s/v/%s", sourceID, ver),
		})
	}
	return releases, nil
}
