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

// GoDLSource fetches Go releases from go.dev/dl.
type GoDLSource struct {
	baseURL string
	client  *http.Client
}

// NewGoDLSource creates a GoDLSource using the real go.dev API.
func NewGoDLSource() *GoDLSource {
	return NewGoDLSourceWithBase("https://go.dev")
}

// NewGoDLSourceWithBase creates a GoDLSource with an overridden base URL (for testing).
func NewGoDLSourceWithBase(baseURL string) *GoDLSource {
	return &GoDLSource{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type goDLRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// FetchReleases returns all Go releases newer than sinceVersion.
// go.dev/dl serves all Go versions regardless of sourceID.
// The only valid sourceID for this source type is "go".
func (g *GoDLSource) FetchReleases(sourceID, sinceVersion string) ([]Release, error) {
	if sourceID != "go" {
		return nil, fmt.Errorf("GoDLSource: unexpected sourceID %q — only \"go\" is supported", sourceID)
	}

	url := fmt.Sprintf("%s/dl/?mode=json&include=all", g.baseURL)

	resp, err := g.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching go.dev/dl: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("go.dev/dl returned %d", resp.StatusCode)
	}

	var raw []goDLRelease
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding go.dev/dl response: %w", err)
	}

	var releases []Release
	for _, r := range raw {
		if !version.IsNewerThan(r.Version, sinceVersion) {
			continue
		}
		releases = append(releases, Release{
			Version:         r.Version,
			IsPrerelease:    !r.Stable,
			PublishedAt:     "",
			ReleaseNotesURL: fmt.Sprintf("https://go.dev/doc/go%s", strings.TrimPrefix(r.Version, "go")),
		})
	}
	return releases, nil
}

func (g *GoDLSource) FetchAll(sourceID, sinceDate string) ([]Release, error) {
	return nil, fmt.Errorf("GoDLSource.FetchAll: not yet implemented")
}
