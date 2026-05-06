package sources

import "fmt"

// Release represents a single release of a package manager.
type Release struct {
	Version         string
	IsPrerelease    bool
	PublishedAt     string
	ReleaseNotesURL string
}

// Source can fetch releases for a package manager from a specific upstream.
type Source interface {
	// FetchReleases returns all releases newer than sinceVersion.
	// If sinceVersion is empty, returns all available releases.
	FetchReleases(sourceID, sinceVersion string) ([]Release, error)
}

// For returns the Source implementation for the given source_type.
func For(sourceType string) (Source, error) {
	switch sourceType {
	case "github_releases":
		return NewGitHubSource("https://api.github.com"), nil
	case "pypi":
		return NewPyPISource(), nil
	case "npm_registry":
		return NewNpmSource(), nil
	case "go_dl":
		return NewGoDLSource(), nil
	default:
		return nil, fmt.Errorf("unknown source_type %q", sourceType)
	}
}
