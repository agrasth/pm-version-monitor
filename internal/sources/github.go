package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/version"
)

// GitHubSource fetches releases from the GitHub Releases API.
type GitHubSource struct {
	baseURL string
	client  *http.Client
}

// NewGitHubSource creates a GitHubSource. baseURL is normally "https://api.github.com"
// but can be overridden in tests.
func NewGitHubSource(baseURL string) *GitHubSource {
	return &GitHubSource{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Prerelease  bool   `json:"prerelease"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// FetchReleases returns all GitHub releases for owner/repo that are newer than sinceVersion.
func (g *GitHubSource) FetchReleases(sourceID, sinceVersion string) ([]Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases?per_page=30", g.baseURL, sourceID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, url)
	}

	var raw []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding GitHub response: %w", err)
	}

	var releases []Release
	for _, r := range raw {
		if !version.IsNewerThan(r.TagName, sinceVersion) {
			continue
		}
		releases = append(releases, Release{
			Version:         r.TagName,
			IsPrerelease:    r.Prerelease,
			PublishedAt:     r.PublishedAt,
			ReleaseNotesURL: r.HTMLURL,
		})
	}
	return releases, nil
}

// FetchAll returns all GitHub releases published on or after sinceDate.
// Paginates through all pages (per_page=100), stopping when a release older than sinceDate is found.
func (g *GitHubSource) FetchAll(sourceID, sinceDate string) ([]Release, error) {
	var cutoff time.Time
	if sinceDate != "" {
		var err error
		cutoff, err = time.Parse("2006-01-02", sinceDate)
		if err != nil {
			return nil, fmt.Errorf("parsing sinceDate %q: %w", sinceDate, err)
		}
	}

	var allReleases []Release
	page := 1

	for {
		url := fmt.Sprintf("%s/repos/%s/releases?per_page=100&page=%d", g.baseURL, sourceID, page)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", url, err)
		}

		if resp.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, url)
		}

		var pageReleases []githubRelease
		if err := json.NewDecoder(resp.Body).Decode(&pageReleases); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding GitHub response page %d: %w", page, err)
		}
		resp.Body.Close()

		done := false
		for _, r := range pageReleases {
			if !cutoff.IsZero() && r.PublishedAt != "" {
				publishedAt, err := time.Parse(time.RFC3339, r.PublishedAt)
				if err == nil && publishedAt.Before(cutoff) {
					done = true
					break
				}
			}
			allReleases = append(allReleases, Release{
				Version:         r.TagName,
				IsPrerelease:    r.Prerelease,
				PublishedAt:     r.PublishedAt,
				ReleaseNotesURL: r.HTMLURL,
			})
		}

		if done || len(pageReleases) < 100 {
			break
		}
		page++
	}

	return allReleases, nil
}
