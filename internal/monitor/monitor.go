package monitor

import (
	"log"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/classifier"
	"github.com/jfrog/pm-version-monitor/internal/config"
	"github.com/jfrog/pm-version-monitor/internal/notify"
	"github.com/jfrog/pm-version-monitor/internal/sources"
	"github.com/jfrog/pm-version-monitor/internal/state"
	"github.com/jfrog/pm-version-monitor/internal/version"
)

// Option allows overriding source implementations (used in tests).
type Option func(*runner)

// WithSource overrides the source used for a given source_type.
func WithSource(sourceType string, src sources.Source) Option {
	return func(r *runner) {
		r.overrides[sourceType] = src
	}
}

type runner struct {
	overrides map[string]sources.Source
}

// Run executes the monitor loop: for each enabled PM, fetches new releases,
// classifies them, notifies via Slack, and returns an updated state.
// Individual PM failures are logged but do not abort the run.
// State is only updated for versions that were successfully notified.
func Run(cfg *config.Config, s *state.State, notifier notify.Notifier, opts ...Option) (*state.State, error) {
	r := &runner{overrides: map[string]sources.Source{}}
	for _, o := range opts {
		o(r)
	}

	updated := cloneState(s)

	for i := range cfg.PackageManagers {
		pm := &cfg.PackageManagers[i]
		if !pm.Enabled {
			continue
		}

		src, err := r.sourceFor(pm.SourceType)
		if err != nil {
			log.Printf("[pm-monitor] WARN: no source for %s (%s): %v", pm.Name, pm.SourceType, err)
			continue
		}

		lastKnown := state.GetLastKnown(updated, pm.Name)
		isNewPM := lastKnown == ""

		releases, err := src.FetchReleases(pm.SourceID, lastKnown)
		if err != nil {
			log.Printf("[pm-monitor] WARN: fetching %s: %v", pm.Name, err)
			continue
		}

		if isNewPM {
			seedLatest(updated, pm.Name, releases)
			continue
		}

		for _, rel := range releases {
			rt := classifier.Classify(rel.Version, rel.IsPrerelease)
			if !classifier.ShouldNotify(rt, pm.NotifyReleaseTypes) {
				continue
			}

			_, err := notifier.Send(notify.Notification{
				PM:          pm.Name,
				DisplayName: pm.DisplayNameOrName(),
				Release:     rel,
				ReleaseType: rt,
				Emoji:       classifier.Emoji(rt),
			})
			if err != nil {
				log.Printf("[pm-monitor] WARN: notify failed for %s %s: %v", pm.Name, rel.Version, err)
				// Do not update state — version will be re-detected next run.
				continue
			}

			state.AddVersion(updated, pm.Name, rel.Version, rt, time.Now().UTC().Format(time.RFC3339))
		}
	}

	return updated, nil
}

func (r *runner) sourceFor(sourceType string) (sources.Source, error) {
	if src, ok := r.overrides[sourceType]; ok {
		return src, nil
	}
	return sources.For(sourceType)
}

// seedLatest finds the semantically newest release and seeds it into state.
// On first run this prevents notifying for all historical versions.
// Sources (especially map-based PyPI/npm) don't guarantee ordering, so we must compare all.
func seedLatest(s *state.State, pm string, releases []sources.Release) {
	if len(releases) == 0 {
		return
	}
	// Compare all releases to find the semantically newest.
	latest := releases[0].Version
	for _, r := range releases[1:] {
		if version.IsNewerThan(r.Version, latest) {
			latest = r.Version
		}
	}
	state.SeedVersion(s, pm, latest)
	log.Printf("[pm-monitor] INFO: seeded %s with %s (first run)", pm, latest)
}

// cloneState makes a deep copy of the state so the original is not mutated.
func cloneState(s *state.State) *state.State {
	ns := &state.State{PMs: make(map[string]*state.PMState, len(s.PMs))}
	for pm, pms := range s.PMs {
		versions := make(map[string]*state.VersionEntry, len(pms.Versions))
		for v, e := range pms.Versions {
			cp := *e
			versions[v] = &cp
		}
		ns.PMs[pm] = &state.PMState{
			LatestKnown: pms.LatestKnown,
			Versions:    versions,
		}
	}
	return ns
}
