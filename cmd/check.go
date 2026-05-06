package cmd

import (
	"fmt"
	"os"

	"github.com/jfrog/pm-version-monitor/internal/config"
	"github.com/jfrog/pm-version-monitor/internal/monitor"
	"github.com/jfrog/pm-version-monitor/internal/notify"
	"github.com/jfrog/pm-version-monitor/internal/state"
)

// RunCheck polls all enabled PMs, sends notifications, and saves updated state.
// configPath and statePath are file paths to the YAML config and JSON state files.
// If dryRun is true, notifications are printed to stdout instead of posting to Slack.
func RunCheck(configPath, statePath string, dryRun bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	var notifier notify.Notifier
	if dryRun {
		fmt.Fprintln(os.Stdout, "[pm-monitor] dry-run mode — notifications printed to stdout, state will be saved")
		notifier = &notify.StdoutNotifier{}
	} else {
		token := os.Getenv("SLACK_BOT_TOKEN")
		if token == "" {
			return fmt.Errorf("SLACK_BOT_TOKEN env var is required (or use --dry-run for local testing)")
		}
		notifier = notify.NewSlackNotifier(token, cfg.Settings.SlackChannel, "https://slack.com/api")
	}

	updated, err := monitor.Run(cfg, s, notifier)
	if err != nil {
		return fmt.Errorf("monitor run: %w", err)
	}

	if err := state.Save(statePath, updated); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Fprintln(os.Stdout, "[pm-monitor] check complete")
	return nil
}
