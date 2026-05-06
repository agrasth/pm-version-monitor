package sources_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/sources"
)

func TestGitHubSourceFetchReleases(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/github_releases.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/apache/maven/releases" {
			t.Errorf("unexpected request path: got %q, want /repos/apache/maven/releases", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewGitHubSource(srv.URL)
	releases, err := src.FetchReleases("apache/maven", "maven-3.9.9")
	if err != nil {
		t.Fatalf("FetchReleases error: %v", err)
	}

	// Only maven-4.0.0-rc-1 is newer than 3.9.9
	if len(releases) != 1 {
		t.Fatalf("expected 1 new release, got %d", len(releases))
	}
	r := releases[0]
	if r.Version != "maven-4.0.0-rc-1" {
		t.Errorf("Version = %q, want maven-4.0.0-rc-1", r.Version)
	}
	if !r.IsPrerelease {
		t.Error("IsPrerelease = false, want true")
	}
	if r.ReleaseNotesURL == "" {
		t.Error("ReleaseNotesURL is empty")
	}
}

func TestGitHubSourceAllNew(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/github_releases.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewGitHubSource(srv.URL)
	// Empty baseline = new PM, all releases are new
	releases, err := src.FetchReleases("apache/maven", "")
	if err != nil {
		t.Fatalf("FetchReleases error: %v", err)
	}
	if len(releases) != 2 {
		t.Errorf("expected 2 releases for empty baseline, got %d", len(releases))
	}
}

func TestGitHubSourceHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src := sources.NewGitHubSource(srv.URL)
	_, err := src.FetchReleases("apache/maven", "")
	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
}
