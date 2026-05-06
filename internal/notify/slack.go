package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/sources"
)

// Notification carries everything needed to build a Slack message.
type Notification struct {
	PM          string
	DisplayName string
	Release     sources.Release
	ReleaseType string
	Emoji       string
}

// Notifier can send a release notification and returns the Slack thread timestamp.
type Notifier interface {
	Send(n Notification) (threadTS string, err error)
}

// StdoutNotifier prints notifications to stdout instead of posting to Slack.
// Use for dry runs and local testing without a Slack token.
type StdoutNotifier struct{}

func (s *StdoutNotifier) Send(n Notification) (string, error) {
	displayDate := n.Release.PublishedAt
	if t, err := time.Parse(time.RFC3339, n.Release.PublishedAt); err == nil {
		displayDate = t.Format("2006-01-02")
	}
	fmt.Printf("\n🔔 [DRY RUN] New release detected\n")
	fmt.Printf("   Package Manager : %s\n", n.DisplayName)
	fmt.Printf("   Version         : %s\n", n.Release.Version)
	fmt.Printf("   Release Type    : %s %s\n", n.Emoji, strings.ToUpper(n.ReleaseType))
	fmt.Printf("   Released        : %s\n", displayDate)
	fmt.Printf("   Release Notes   : %s\n", n.Release.ReleaseNotesURL)
	fmt.Printf("   [Would post to Slack and show Run Tests button]\n")
	return "dry-run-ts", nil
}

// SlackNotifier posts Block Kit messages to Slack.
type SlackNotifier struct {
	token   string
	channel string
	baseURL string
	client  *http.Client
}

// NewSlackNotifier creates a SlackNotifier.
// baseURL is normally "https://slack.com/api" but can be overridden in tests.
func NewSlackNotifier(token, channel, baseURL string) *SlackNotifier {
	return &SlackNotifier{
		token:   token,
		channel: channel,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Send posts a Block Kit notification and returns the Slack thread timestamp.
func (s *SlackNotifier) Send(n Notification) (string, error) {
	displayDate := n.Release.PublishedAt
	if t, err := time.Parse(time.RFC3339, n.Release.PublishedAt); err == nil {
		displayDate = t.Format("2006-01-02")
	}

	payload := map[string]interface{}{
		"channel": s.channel,
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": "🔔 New package manager release detected",
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Package Manager*\n%s", n.DisplayName)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Version*\n`%s`", n.Release.Version)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Release Type*\n%s %s", n.Emoji, strings.ToUpper(n.ReleaseType))},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Released*\n%s", displayDate)},
				},
			},
			{
				"type": "actions",
				"elements": []map[string]interface{}{
					{
						"type": "button",
						"text": map[string]string{"type": "plain_text", "text": "📋 View Release Notes"},
						"url":  n.Release.ReleaseNotesURL,
					},
					{
						"type":  "button",
						"style": "primary",
						"text":  map[string]string{"type": "plain_text", "text": "▶️ Run Compatibility Tests"},
						"url":   "https://github.com/jfrog/jfrog-cli/actions/workflows/pm-compatibility-test.yml",
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling Slack payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building Slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("posting to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("Slack API returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		OK      bool   `json:"ok"`
		TS      string `json:"ts"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding Slack response: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("Slack API error: %s", result.Error)
	}
	return result.TS, nil
}
