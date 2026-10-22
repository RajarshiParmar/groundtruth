package config

import (
	"strings"
	"testing"
)

func TestLoad_RepoBreakdown(t *testing.T) {
	cfg, err := Load(writeConfig(t, strings.Replace(validBase, "metrics:\n", "metrics:\n  repo_breakdown: true\n", 1)))
	if err != nil || !cfg.Metrics.RepoBreakdown {
		t.Errorf("on: err %v, RepoBreakdown %v", err, cfg.Metrics.RepoBreakdown)
	}
	cfg, err = Load(writeConfig(t, validBase))
	if err != nil || cfg.Metrics.RepoBreakdown {
		t.Errorf("absent: err %v, RepoBreakdown %v", err, cfg.Metrics.RepoBreakdown)
	}
}
