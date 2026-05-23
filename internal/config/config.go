package config

import (
	"bytes"
	"fmt"
	"os"
	"time"

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
	Weights ScoringWeights `yaml:"weights"`
}

// ScoringWeights is the typed schema for scoring.weights.
// Unknown fields are rejected at decode time via yaml.v3 KnownFields(true).
type ScoringWeights struct {
	// CommitCount is a flat per-commit multiplier applied on top of the
	// per-type weight. A zero value is treated as 1.0 by the metric.
	CommitCount float64 `yaml:"commit_count"`
	// CommitByType overrides the default per-type weights. Missing types
	// fall back to defaults defined in the metrics package.
	CommitByType map[string]float64 `yaml:"commit_by_type"`
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
	// version
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version: %d", cfg.Version)
	}

	// repository
	if cfg.Repository.Path == "" {
		return fmt.Errorf("repository.path is required")
	}

	// time range
	hasExplicit := cfg.TimeRange.From != "" || cfg.TimeRange.To != ""
	hasRelative := cfg.TimeRange.Last != ""

	if hasExplicit && hasRelative {
		return fmt.Errorf("time_range: use either from/to or last, not both")
	}

	if !hasExplicit && !hasRelative {
		return fmt.Errorf("time_range: one of from/to or last must be specified")
	}

	if hasExplicit {
		if cfg.TimeRange.From == "" || cfg.TimeRange.To == "" {
			return fmt.Errorf("time_range: both from and to must be set")
		}

		from, err := time.Parse("2006-01-02", cfg.TimeRange.From)
		if err != nil {
			return fmt.Errorf("time_range.from: invalid date %q", cfg.TimeRange.From)
		}

		to, err := time.Parse("2006-01-02", cfg.TimeRange.To)
		if err != nil {
			return fmt.Errorf("time_range.to: invalid date %q", cfg.TimeRange.To)
		}

		if from.After(to) {
			return fmt.Errorf("time_range: from must be before or equal to to")
		}
	}

	if hasRelative {
		switch cfg.TimeRange.Last {
		case "30d", "6m", "1y":
			// ok
		default:
			return fmt.Errorf("time_range.last: unsupported value %q", cfg.TimeRange.Last)
		}
	}

	// merge commits
	switch cfg.MergeCommits.Mode {
	case "ignore", "include", "analyze":
		// ok
	default:
		return fmt.Errorf("merge_commits.mode must be one of ignore, include, analyze")
	}

	// metrics
	if !(cfg.Metrics.CommitCount ||
		cfg.Metrics.CommitByType ||
		cfg.Metrics.LinesChanged ||
		cfg.Metrics.FilesChanged ||
		cfg.Metrics.ActivityTimeline ||
		cfg.Metrics.MergeSummaries ||
		cfg.Metrics.ContributionScore) {
		return fmt.Errorf("metrics: at least one metric must be enabled")
	}

	// output
	if len(cfg.Output.Formats) == 0 {
		return fmt.Errorf("output.formats must contain at least one format")
	}

	allowedFormats := map[string]bool{
		"csv":  true,
		"xlsx": true,
		"pdf":  true,
	}

	for _, f := range cfg.Output.Formats {
		if !allowedFormats[f] {
			return fmt.Errorf("output.formats: unsupported format %q", f)
		}
	}

	if cfg.Output.Directory == "" {
		return fmt.Errorf("output.directory is required")
	}

	// scoring
	if cfg.Scoring.Enabled {
		w := cfg.Scoring.Weights
		if w.CommitCount == 0 && len(w.CommitByType) == 0 {
			return fmt.Errorf("scoring.enabled is true but no weights are defined")
		}
		if w.CommitCount < 0 {
			return fmt.Errorf("scoring.weights.commit_count must be non-negative")
		}
		for k, v := range w.CommitByType {
			if v < 0 {
				return fmt.Errorf("scoring.weights.commit_by_type[%s] must be non-negative", k)
			}
		}
		if !cfg.Metrics.ContributionScore {
			return fmt.Errorf("scoring.enabled requires metrics.contribution_score to be true")
		}
	}

	return nil
}
