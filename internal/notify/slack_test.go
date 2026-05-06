package notify_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jfrog/pm-version-monitor/internal/notify"
	"github.com/jfrog/pm-version-monitor/internal/sources"
)

func TestSlackNotifierSend(t *testing.T) {
	var captured map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify Authorization header contains the token
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", authHeader, "Bearer test-token")
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"ts":"1234567890.123456","channel":"C123"}`))
	}))
	defer srv.Close()

	n := notify.NewSlackNotifier("test-token", "#pm-releases", srv.URL)

	rel := sources.Release{
		Version:         "maven-4.0.0-rc-1",
		IsPrerelease:    true,
		PublishedAt:     "2026-05-04T10:00:00Z",
		ReleaseNotesURL: "https://github.com/apache/maven/releases/tag/maven-4.0.0-rc-1",
	}

	ts, err := n.Send(notify.Notification{
		PM:          "maven",
		DisplayName: "Maven",
		Release:     rel,
		ReleaseType: "rc",
		Emoji:       "🔖",
	})
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if ts != "1234567890.123456" {
		t.Errorf("ts = %q, want 1234567890.123456", ts)
	}
	if captured["channel"] != "#pm-releases" {
		t.Errorf("channel = %v, want #pm-releases", captured["channel"])
	}
	if captured["blocks"] == nil {
		t.Error("blocks field missing from Slack payload")
	}
}

func TestSlackNotifierSlackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer srv.Close()

	n := notify.NewSlackNotifier("test-token", "#pm-releases", srv.URL)
	_, err := n.Send(notify.Notification{
		PM:          "maven",
		DisplayName: "Maven",
		Release:     sources.Release{Version: "4.0.0"},
	})
	if err == nil {
		t.Error("expected error for Slack ok=false, got nil")
	}
}

func TestSlackNotifierHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := notify.NewSlackNotifier("test-token", "#pm-releases", srv.URL)
	_, err := n.Send(notify.Notification{
		PM: "maven", DisplayName: "Maven",
		Release: sources.Release{Version: "4.0.0"},
	})
	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
}
