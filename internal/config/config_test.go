package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validBase = `
version: 1
repository:
  path: /tmp/repo
identity:
  authors:
    - name: x
      email: x@y.z
time_range:
  last: 30d
merge_commits:
  mode: ignore
metrics:
  commit_count: true
  contribution_score: true
output:
  formats: [csv]
  directory: ./out
`

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte(contents), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestLoad_ValidScoringWeights(t *testing.T) {
	cfg, err := Load(writeConfig(t, validBase+`
scoring:
  enabled: true
  weights:
    commit_count: 2
    commit_by_type:
      feat: 5
      fix: 3
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Scoring.Weights.CommitCount != 2 {
		t.Errorf("CommitCount = %v, want 2", cfg.Scoring.Weights.CommitCount)
	}
	if cfg.Scoring.Weights.CommitByType["feat"] != 5 {
		t.Errorf("feat = %v, want 5", cfg.Scoring.Weights.CommitByType["feat"])
	}
}

func TestLoad_ScoringEnabledButNoWeights(t *testing.T) {
	_, err := Load(writeConfig(t, validBase+`
scoring:
  enabled: true
`))
	if err == nil || !strings.Contains(err.Error(), "no weights are defined") {
		t.Fatalf("expected weight error, got: %v", err)
	}
}

func TestLoad_NegativeWeightRejected(t *testing.T) {
	_, err := Load(writeConfig(t, validBase+`
scoring:
  enabled: true
  weights:
    commit_count: -1
`))
	if err == nil || !strings.Contains(err.Error(), "commit_count must be non-negative") {
		t.Fatalf("expected negative weight error, got: %v", err)
	}
}

func TestLoad_NegativeTypeWeightRejected(t *testing.T) {
	_, err := Load(writeConfig(t, validBase+`
scoring:
  enabled: true
  weights:
    commit_by_type:
      feat: -2
`))
	if err == nil || !strings.Contains(err.Error(), "commit_by_type[feat]") {
		t.Fatalf("expected negative type weight error, got: %v", err)
	}
}

func TestLoad_UnknownWeightKeyRejected(t *testing.T) {
	_, err := Load(writeConfig(t, validBase+`
scoring:
  enabled: true
  weights:
    commit_count: 1
    bogus_key: 5
`))
	if err == nil || !strings.Contains(err.Error(), "bogus_key") {
		t.Fatalf("expected unknown-field error, got: %v", err)
	}
}

func TestLoad_ScoringEnabledRequiresContributionScoreMetric(t *testing.T) {
	cfgYAML := `
version: 1
repository:
  path: /tmp/repo
identity:
  authors:
    - name: x
      email: x@y.z
time_range:
  last: 30d
merge_commits:
  mode: ignore
metrics:
  commit_count: true
output:
  formats: [csv]
  directory: ./out
scoring:
  enabled: true
  weights:
    commit_count: 1
`
	_, err := Load(writeConfig(t, cfgYAML))
	if err == nil || !strings.Contains(err.Error(), "metrics.contribution_score") {
		t.Fatalf("expected contribution_score requirement error, got: %v", err)
	}
}

func TestLoad_ScoringDisabledByDefault(t *testing.T) {
	cfg, err := Load(writeConfig(t, validBase))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Scoring.Enabled {
		t.Errorf("Scoring.Enabled = true, want false")
	}
}
