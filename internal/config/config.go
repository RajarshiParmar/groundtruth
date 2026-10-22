package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RajarshiParmar/groundtruth/internal/paths"
)

type Config struct {
	Version      int          `yaml:"version"`
	Repository   Repository   `yaml:"repository"`
	Identity     Identity     `yaml:"identity"`
	TimeRange    TimeRange    `yaml:"time_range"`
	MergeCommits MergeCommits `yaml:"merge_commits"`
	Metrics      Metrics      `yaml:"metrics"`
	Report       *Report      `yaml:"report"` // deprecated: decoded only to warn about it
	Output       Output       `yaml:"output"`
	Scoring      Scoring      `yaml:"scoring"`
	Highlights   Highlights   `yaml:"highlights"`
	Tickets      Tickets      `yaml:"tickets"`
	ExcludePaths []string     `yaml:"exclude_paths"`

	// Warnings collects problems that do not stop the config from loading
	Warnings []string `yaml:"-"`
}

type Repository struct {
	Path    string `yaml:"path"`
	BaseDir bool   `yaml:"base_dir"`
}

type Identity struct {
	Authors    []Author `yaml:"authors"`
	Mailmap    string   `yaml:"mailmap"`     // personal .mailmap applied to every repo
	CoAuthored bool     `yaml:"co_authored"` // credit commits that name you in Co-authored-by
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
	CommitByScope     bool `yaml:"commit_by_scope"`
	Highlights        bool `yaml:"highlights"`
	Tickets           bool `yaml:"tickets"`
}

// Report is the retired report.sections block; metrics.* decides what is shown
type Report struct {
	Sections ReportSections `yaml:"sections"`
}

type ReportSections struct {
	Summary  bool `yaml:"summary"`
	Metrics  bool `yaml:"metrics"`
	Timeline bool `yaml:"timeline"`
	Merges   bool `yaml:"merges"`
}

// Highlights configures the highlights metric. Zero values fall back to the
// defaults defined in the metrics package.
type Highlights struct {
	Types    []string `yaml:"types"`
	PerScope int      `yaml:"per_scope"`
}

// Tickets lists issue-tracker keys (e.g. PROJ for PROJ-123); "#123" is always matched
type Tickets struct {
	Prefixes []string `yaml:"prefixes"`
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

// Load reads and validates a config file. Overrides (e.g. from CLI flags)
// are applied after decoding and before validation.
func Load(path string, overrides ...func(*Config)) (*Config, error) {
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

	for _, o := range overrides {
		o(&cfg)
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

	if hasExplicit && (cfg.TimeRange.From == "" || cfg.TimeRange.To == "") {
		return fmt.Errorf("time_range: both from and to must be set")
	}

	// Parsing and supported values live in TimeRangeToBounds
	from, to, err := TimeRangeToBounds(cfg.TimeRange)
	if err != nil {
		return err
	}

	// to is exclusive (the day after time_range.to), so from == to is a one-day range
	if !from.Before(to) {
		return fmt.Errorf("time_range: from must be before or equal to to")
	}

	// identity
	if cfg.Identity.Mailmap != "" {
		path, err := ExpandHome(cfg.Identity.Mailmap)
		if err != nil {
			return fmt.Errorf("identity.mailmap: %w", err)
		}
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("identity.mailmap: %w", err)
		}
		f.Close()
	}
	if cfg.Identity.CoAuthored && len(cfg.Identity.Authors) == 0 {
		cfg.Warnings = append(cfg.Warnings, "identity.co_authored has no effect without identity.authors: every author already counts.")
	}

	// exclude paths
	if _, err := paths.NewExcluder(cfg.ExcludePaths); err != nil {
		return err
	}
	for _, g := range cfg.ExcludePaths {
		// Repo paths never start or end with "/", and "\" escapes the next character
		if strings.HasPrefix(g, "/") || strings.HasSuffix(g, "/") || strings.Contains(g, `\`) {
			cfg.Warnings = append(cfg.Warnings, fmt.Sprintf(`exclude_paths: %q never matches a file: paths are relative to the repo root, use "/" between directories and no "/" at either end (for example "**/vendor/**")`, g))
		}
	}

	// merge commits
	switch cfg.MergeCommits.Mode {
	case "ignore", "include", "analyze":
		// ok
	default:
		return fmt.Errorf("merge_commits.mode must be one of ignore, include, analyze")
	}

	if cfg.Metrics.MergeSummaries && cfg.MergeCommits.Mode != "analyze" {
		return fmt.Errorf("metrics.merge_summaries requires merge_commits.mode: analyze")
	}

	// metrics
	if !(cfg.Metrics.CommitCount ||
		cfg.Metrics.CommitByType ||
		cfg.Metrics.LinesChanged ||
		cfg.Metrics.FilesChanged ||
		cfg.Metrics.ActivityTimeline ||
		cfg.Metrics.MergeSummaries ||
		cfg.Metrics.ContributionScore ||
		cfg.Metrics.CommitByScope ||
		cfg.Metrics.Highlights ||
		cfg.Metrics.Tickets) {
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
		"md":   true,
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

	if cfg.Report != nil {
		cfg.Warnings = append(cfg.Warnings, "report.sections is no longer used: metrics.* decides what appears in every report. Remove the report: block.")
	}

	// highlights and tickets
	if cfg.Highlights.PerScope < 0 {
		return fmt.Errorf("highlights.per_scope must be non-negative")
	}
	for _, t := range cfg.Highlights.Types {
		if !commitTypePattern.MatchString(t) {
			return fmt.Errorf("highlights.types: %q must be a lowercase word", t)
		}
	}
	for _, p := range cfg.Tickets.Prefixes {
		if !ticketPrefixPattern.MatchString(p) {
			return fmt.Errorf("tickets.prefixes: %q must match ^[A-Z][A-Z0-9]*$ (e.g. PROJ)", p)
		}
	}

	return nil
}

var (
	commitTypePattern   = regexp.MustCompile(`^[a-z]+$`)
	ticketPrefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*$`)
)

// ExpandHome replaces a leading "~/" with the home directory; other paths
// are returned as written (relative ones resolve against the working directory)
func ExpandHome(path string) (string, error) {
	rest, ok := strings.CutPrefix(path, "~/")
	if !ok {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, rest), nil
}
