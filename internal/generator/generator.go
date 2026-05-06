package generator

import (
	"html/template"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/jfrog/pm-version-monitor/internal/state"
)

// MatrixData is passed to the HTML template.
type MatrixData struct {
	GeneratedAt string
	Passing     int
	Failing     int
	Pending     int
	Total       int
	Groups      []PMGroup
}

// PMGroup holds all version rows for one package manager.
type PMGroup struct {
	Name     string
	Versions []VersionRow
}

// VersionRow is one row in the compatibility table.
type VersionRow struct {
	Version      string
	Type         string
	Status       string
	StatusClass  string
	JFCLIVersion string
	Date         string
	RunURL       string
}

// Generate writes a self-contained HTML compatibility matrix to w.
func Generate(s *state.State, w io.Writer) error {
	data := buildData(s)
	tmpl, err := template.New("matrix").Parse(matrixTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

func buildData(s *state.State) MatrixData {
	data := MatrixData{
		GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04 UTC"),
	}

	pmNames := make([]string, 0, len(s.PMs))
	for name := range s.PMs {
		pmNames = append(pmNames, name)
	}
	sort.Strings(pmNames)

	for _, pmName := range pmNames {
		pmState := s.PMs[pmName]
		if len(pmState.Versions) == 0 {
			continue
		}

		verKeys := make([]string, 0, len(pmState.Versions))
		for v := range pmState.Versions {
			verKeys = append(verKeys, v)
		}
		// Sort newest first by DetectedAt
		sort.Slice(verKeys, func(i, j int) bool {
			ei := pmState.Versions[verKeys[i]]
			ej := pmState.Versions[verKeys[j]]
			ti, _ := time.Parse(time.RFC3339, ei.DetectedAt)
			tj, _ := time.Parse(time.RFC3339, ej.DetectedAt)
			if ti.Equal(tj) {
				return verKeys[i] > verKeys[j]
			}
			return ti.After(tj)
		})

		group := PMGroup{Name: pmName}

		for _, ver := range verKeys {
			entry := pmState.Versions[ver]
			row := VersionRow{
				Version:      ver,
				Type:         strings.ToUpper(entry.ReleaseType),
				JFCLIVersion: entry.JFCLIVersion,
				RunURL:       entry.RunURL,
			}

			// Format date: prefer TestedAt, fall back to DetectedAt
			dateStr := entry.TestedAt
			if dateStr == "" {
				dateStr = entry.DetectedAt
			}
			if dateStr != "" {
				if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
					row.Date = t.Format("2006-01-02")
				}
			}

			switch entry.TestStatus {
			case state.StatusPassed:
				row.Status = "Passing"
				row.StatusClass = "status-passing"
				data.Passing++
			case state.StatusFailed:
				row.Status = "Failing"
				row.StatusClass = "status-failing"
				data.Failing++
			case state.StatusPending:
				row.Status = "Pending"
				row.StatusClass = "status-pending"
				data.Pending++
			default:
				row.Status = "Unknown"
				row.StatusClass = "status-unknown"
			}
			data.Total++
			group.Versions = append(group.Versions, row)
		}
		data.Groups = append(data.Groups, group)
	}

	return data
}
