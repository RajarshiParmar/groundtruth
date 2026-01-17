package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version      int          `yaml:"version"`
	Repository   Repository   `yaml:"repository"`
	Identity     Identity     `yaml:"identity"`
	TimeRange    TimeRange    `yaml:"time_range"`
	MergeCommits MergeCommits `yaml:"merge_commits"`
	Metrics      Metrics      `yaml:"metrics"`
	Report       Report       `yaml:"report"`
	Output       Output       `yaml:"output"`
	Scoring      Scoring      `yaml:"scoring"`
}

type Repository struct {
	Path    string `yaml:"path"`
	BaseDir bool   `yaml:"base_dir"`
}

type Identity struct {
	Authors []Author `yaml:"authors"`
}

type Author struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type TimeRange struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
	Last string `yaml:"last"`
}

type MergeCommits struct {
	Mode string `yaml:"mode"`
}

type Metrics struct {
	CommitCount       bool `yaml:"commit_count"`
	CommitByType      bool `yaml:"commit_by_type"`
	LinesChanged      bool `yaml:"lines_changed"`
	FilesChanged      bool `yaml:"files_changed"`
	ActivityTimeline  bool `yaml:"activity_timeline"`
	MergeSummaries    bool `yaml:"merge_summaries"`
	ContributionScore bool `yaml:"contribution_score"`
}

type Report struct {
	Sections ReportSections `yaml:"sections"`
}

type ReportSections struct {
	Summary  bool `yaml:"summary"`
	Metrics  bool `yaml:"metrics"`
	Timeline bool `yaml:"timeline"`
	Merges   bool `yaml:"merges"`
}

type Output struct {
	Formats   []string `yaml:"formats"`
	Directory string   `yaml:"directory"`
}

type Scoring struct {
	Enabled bool           `yaml:"enabled"`
	Weights map[string]any `yaml:"weights"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %w", err)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("Invalid config file: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", cfg.Version)
	}

	// Time range validation
	hasExplicit := cfg.TimeRange.From != "" || cfg.TimeRange.To != ""
	hasRelative := cfg.TimeRange.Last != ""

	if hasExplicit && hasRelative {
		return fmt.Errorf("time_range: use either from/to or last, not both")
	}

	if !hasExplicit && !hasRelative {
		return fmt.Errorf("time_range: one of from/to or last must be specified")
	}

	switch cfg.MergeCommits.Mode {
	case "ignore", "include", "analyze":
	// ok
	default:
		return fmt.Errorf("merge_commits.mode must be one of ignore, include, analyze")
	}

	if len(cfg.Output.Formats) == 0 {
		return fmt.Errorf("output.formats must contain at least one format")
	}

	return nil
}
