package sources_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/sources"
)

func TestGoDLSourceFetchReleases(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/go_dl_response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewGoDLSourceWithBase(srv.URL)
	releases, err := src.FetchReleases("go", "go1.24.3")
	if err != nil {
		t.Fatalf("FetchReleases error: %v", err)
	}
	// go1.25 and go1.25rc1 are newer than go1.24.3
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases newer than go1.24.3, got %d: %+v", len(releases), releases)
	}
}

func TestGoDLSourceInvalidSourceID(t *testing.T) {
	src := sources.NewGoDLSourceWithBase("http://unused")
	_, err := src.FetchReleases("notgo", "go1.24.3")
	if err == nil {
		t.Error("expected error for invalid sourceID, got nil")
	}
}

func TestGoDLSourceFetchAll(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/go_dl_response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewGoDLSourceWithBase(srv.URL)

	// sinceDate is ignored for go.dev — all 3 releases should be returned
	releases, err := src.FetchAll("go", "2020-01-01")
	if err != nil {
		t.Fatalf("FetchAll error: %v", err)
	}
	if len(releases) != 3 {
		t.Fatalf("expected 3 releases (sinceDate ignored for go.dev), got %d", len(releases))
	}
}
