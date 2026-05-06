package sources_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/sources"
)

func TestPyPISourceFetchReleases(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/pypi_response.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pypi/pip/json" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	src := sources.NewPyPISourceWithBase(srv.URL)
	releases, err := src.FetchReleases("pip", "25.1.1")
	if err != nil {
		t.Fatalf("FetchReleases error: %v", err)
	}
	// 25.2.0b1 and 25.2.0rc1 are newer than 25.1.1
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases newer than 25.1.1, got %d: %+v", len(releases), releases)
	}
}

func TestPyPISourceEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"info":{"name":"pip"},"releases":{}}`))
	}))
	defer srv.Close()

	src := sources.NewPyPISourceWithBase(srv.URL)
	releases, err := src.FetchReleases("pip", "25.1.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(releases) != 0 {
		t.Errorf("expected 0 releases, got %d", len(releases))
	}
}
