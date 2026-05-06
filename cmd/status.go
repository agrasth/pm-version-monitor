package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/jfrog/pm-version-monitor/internal/state"
)

// RunStatus prints the current state in a human-readable table.
func RunStatus(statePath string) error {
	s, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PM\tLATEST KNOWN\tVERSION\tTYPE\tSTATUS\tDETECTED")
	fmt.Fprintln(w, "──\t────────────\t───────\t────\t──────\t────────")

	// Sort PMs for deterministic output
	pms := make([]string, 0, len(s.PMs))
	for pm := range s.PMs {
		pms = append(pms, pm)
	}
	sort.Strings(pms)

	for _, pm := range pms {
		pmState := s.PMs[pm]
		if len(pmState.Versions) == 0 {
			fmt.Fprintf(w, "%s\t%s\t-\t-\t-\t-\n", pm, pmState.LatestKnown)
			continue
		}
		// Sort versions for deterministic output
		vers := make([]string, 0, len(pmState.Versions))
		for v := range pmState.Versions {
			vers = append(vers, v)
		}
		sort.Strings(vers)
		for _, ver := range vers {
			entry := pmState.Versions[ver]
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				pm, pmState.LatestKnown, ver, entry.ReleaseType, entry.TestStatus, entry.DetectedAt)
		}
	}
	return w.Flush()
}
