package sources_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/sources"
)

func TestNpmSourceFetchReleases(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/npm_response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pnpm" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewNpmSourceWithBase(srv.URL)
	releases, err := src.FetchReleases("pnpm", "10.9.0")
	if err != nil {
		t.Fatalf("FetchReleases error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases newer than 10.9.0, got %d", len(releases))
	}
}

func TestNpmSourceFetchAll(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/npm_response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pnpm" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewNpmSourceWithBase(srv.URL)

	// sinceDate "2026-04-10" should exclude 10.9.0 (2026-04-01), include the other 2
	releases, err := src.FetchAll("pnpm", "2026-04-10")
	if err != nil {
		t.Fatalf("FetchAll error: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases since 2026-04-10, got %d", len(releases))
	}
}
