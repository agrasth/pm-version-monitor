package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/classifier"
	"github.com/jfrog/pm-version-monitor/internal/config"
	"github.com/jfrog/pm-version-monitor/internal/sources"
	"github.com/jfrog/pm-version-monitor/internal/state"
)

// RunDiscover fetches all historical releases within each PM's discover_range
// and adds them to state as pending. Does NOT send Slack notifications.
// Run once to seed full historical version history before running compatibility tests.
func RunDiscover(configPath, statePath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	s, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	added := 0
	for i := range cfg.PackageManagers {
		pm := &cfg.PackageManagers[i]
		if !pm.Enabled || pm.DiscoverRange == "" {
			continue
		}

		src, err := sources.For(pm.SourceType)
		if err != nil {
			log.Printf("[pm-monitor] WARN: no source for %s: %v", pm.Name, err)
			continue
		}

		log.Printf("[pm-monitor] INFO: discovering %s releases since %s", pm.Name, pm.DiscoverRange)

		releases, err := src.FetchAll(pm.SourceID, pm.DiscoverRange)
		if err != nil {
			log.Printf("[pm-monitor] WARN: fetching all releases for %s: %v", pm.Name, err)
			continue
		}

		limit := cfg.Settings.DiscoverMaxVersions
		pmAdded := 0
		for _, rel := range releases {
			if limit > 0 && pmAdded >= limit {
				log.Printf("[pm-monitor] INFO: %s reached discover_max_versions limit (%d), stopping", pm.Name, limit)
				break
			}
			if state.HasVersion(s, pm.Name, rel.Version) {
				continue
			}
			rt := classifier.Classify(rel.Version, rel.IsPrerelease)
			detectedAt := time.Now().UTC().Format(time.RFC3339)
			if rel.PublishedAt != "" {
				detectedAt = rel.PublishedAt
			}
			state.AddVersion(s, pm.Name, rel.Version, rt, detectedAt)
			added++
			pmAdded++
			log.Printf("[pm-monitor] INFO: queued %s %s (%s)", pm.Name, rel.Version, rt)
		}
	}

	if err := state.Save(statePath, s); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Fprintf(os.Stdout, "[pm-monitor] discover complete — %d new versions queued\n", added)
	return nil
}
