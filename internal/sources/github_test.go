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

func TestGitHubSourceFetchAll(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/github_releases_paginated.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/apache/maven/releases" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewGitHubSource(srv.URL)

	// sinceDate "2023-01-01" should include maven-3.8.8 (2023-01-15) and newer but exclude maven-3.6.3 (2021)
	releases, err := src.FetchAll("apache/maven", "2023-01-01")
	if err != nil {
		t.Fatalf("FetchAll error: %v", err)
	}
	if len(releases) != 3 {
		t.Fatalf("expected 3 releases since 2023-01-01, got %d: %+v", len(releases), releases)
	}

	// Empty sinceDate should return all 4
	all, err := src.FetchAll("apache/maven", "")
	if err != nil {
		t.Fatalf("FetchAll (no cutoff) error: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 releases with no cutoff, got %d", len(all))
	}
}

func TestGitHubSourceFetchAllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	src := sources.NewGitHubSource(srv.URL)
	_, err := src.FetchAll("apache/maven", "2023-01-01")
	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
}
