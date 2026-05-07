package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jfrog/pm-version-monitor/cmd"
)

const (
	defaultConfigPath = ".github/pm-versions-config.yml"
	defaultStatePath  = ".github/pm-versions-state.json"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "check":
		fs := flag.NewFlagSet("check", flag.ExitOnError)
		dryRun := fs.Bool("dry-run", false, "print notifications to stdout instead of posting to Slack (no token needed)")
		fs.Parse(os.Args[2:])
		configPath := envOr("PM_CONFIG_PATH", defaultConfigPath)
		statePath := envOr("PM_STATE_PATH", defaultStatePath)
		if err := cmd.RunCheck(configPath, statePath, *dryRun); err != nil {
			fmt.Fprintf(os.Stderr, "[pm-monitor] ERROR: %v\n", err)
			os.Exit(1)
		}

	case "discover":
		configPath := envOr("PM_CONFIG_PATH", defaultConfigPath)
		statePath := envOr("PM_STATE_PATH", defaultStatePath)
		if err := cmd.RunDiscover(configPath, statePath); err != nil {
			fmt.Fprintf(os.Stderr, "[pm-monitor] ERROR: %v\n", err)
			os.Exit(1)
		}

	case "matrix":
		fs := flag.NewFlagSet("matrix", flag.ExitOnError)
		outputPath := fs.String("output", "docs/index.html", "output path for the HTML matrix")
		fs.Parse(os.Args[2:])
		statePath := envOr("PM_STATE_PATH", defaultStatePath)
		if err := cmd.RunMatrix(statePath, *outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "[pm-monitor] ERROR: %v\n", err)
			os.Exit(1)
		}

	case "status":
		statePath := envOr("PM_STATE_PATH", defaultStatePath)
		if err := cmd.RunStatus(statePath); err != nil {
			fmt.Fprintf(os.Stderr, "[pm-monitor] ERROR: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "[pm-monitor] unknown command %q\n\n", os.Args[1])
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pm-version-monitor <check|discover|matrix|status>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  check    poll all enabled PMs, notify new releases, update state")
	fmt.Fprintln(os.Stderr, "  discover fetch all historical PM releases into state (bootstrap, run once)")
	fmt.Fprintln(os.Stderr, "  matrix   generate HTML compatibility matrix from state")
	fmt.Fprintln(os.Stderr, "  status   print current state table")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "env vars:")
	fmt.Fprintln(os.Stderr, "  PM_CONFIG_PATH   path to pm-versions-config.yml (default: .github/pm-versions-config.yml)")
	fmt.Fprintln(os.Stderr, "  PM_STATE_PATH    path to pm-versions-state.json (default: .github/pm-versions-state.json)")
	fmt.Fprintln(os.Stderr, "  SLACK_BOT_TOKEN  Slack bot token (required for check, unless --dry-run)")
	fmt.Fprintln(os.Stderr, "  GITHUB_TOKEN     GitHub API token (optional, increases rate limit)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "flags (check only):")
	fmt.Fprintln(os.Stderr, "  --dry-run        print notifications to stdout, no Slack token needed")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "flags (matrix only):")
	fmt.Fprintln(os.Stderr, "  --output         output path (default: docs/index.html)")
	os.Exit(1)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
