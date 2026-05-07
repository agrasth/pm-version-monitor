package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the parsed representation of pm-versions-config.yml.
type Config struct {
	Settings        Settings         `yaml:"settings"`
	PackageManagers []PackageManager `yaml:"package_managers"`
}

type Settings struct {
	AutoTestAfterHours  int    `yaml:"auto_test_after_hours"`
	SlackChannel        string `yaml:"slack_channel"`
	DiscoverMaxVersions int    `yaml:"discover_max_versions"` // max new versions to queue per PM per discover run; 0 = unlimited
}

type PackageManager struct {
	Name               string   `yaml:"name"`
	DisplayName        string   `yaml:"display_name"`
	SourceType         string   `yaml:"source_type"`
	SourceID           string   `yaml:"source_id"`
	NotifyReleaseTypes []string `yaml:"notify_release_types"`
	DiscoverRange      string   `yaml:"discover_range"` // date string "YYYY-MM-DD"; if empty, PM is skipped by discover
	Enabled            bool     `yaml:"enabled"`
}

// Load reads and parses a pm-versions-config.yml file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

// DisplayNameOrName returns DisplayName if set, otherwise Name.
func (pm *PackageManager) DisplayNameOrName() string {
	if pm.DisplayName != "" {
		return pm.DisplayName
	}
	return pm.Name
}
