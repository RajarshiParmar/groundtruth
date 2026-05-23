package metrics

import (
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func mkCommit(t string) model.ClassifiedCommit {
	return model.ClassifiedCommit{Type: t}
}

func resultScore(t *testing.T, r Result) model.ContributionScore {
	t.Helper()
	s, ok := r.Value.(model.ContributionScore)
	if !ok {
		t.Fatalf("expected model.ContributionScore, got %T", r.Value)
	}
	return s
}

func TestContributionScore_Empty(t *testing.T) {
	m := &ContributionScoreMetric{}
	got := resultScore(t, m.Compute(nil))

	if got.Total != 0 {
		t.Errorf("Total = %v, want 0", got.Total)
	}
	if got.CommitCount != 0 {
		t.Errorf("CommitCount = %d, want 0", got.CommitCount)
	}
	if len(got.Breakdown) != 0 {
		t.Errorf("Breakdown len = %d, want 0", len(got.Breakdown))
	}
}

func TestContributionScore_DefaultsApplied(t *testing.T) {
	m := &ContributionScoreMetric{}
	commits := []model.ClassifiedCommit{
		mkCommit("feat"), // 3
		mkCommit("fix"),  // 2
		mkCommit("docs"), // 1
	}
	got := resultScore(t, m.Compute(commits))

	if got.Total != 6 {
		t.Errorf("Total = %v, want 6", got.Total)
	}
	if got.CommitCount != 3 {
		t.Errorf("CommitCount = %d, want 3", got.CommitCount)
	}
	if got.Breakdown["feat"] != 3 || got.Breakdown["fix"] != 2 || got.Breakdown["docs"] != 1 {
		t.Errorf("unexpected breakdown: %v", got.Breakdown)
	}
}

func TestContributionScore_UnknownTypeFallsBackToOther(t *testing.T) {
	m := &ContributionScoreMetric{}
	commits := []model.ClassifiedCommit{
		mkCommit("madeup"),
		mkCommit(""), // empty -> "other"
	}
	got := resultScore(t, m.Compute(commits))

	if got.Total != 2 {
		t.Errorf("Total = %v, want 2 (2 commits @ 1.0)", got.Total)
	}
	if got.Breakdown["madeup"] != 1 {
		t.Errorf("madeup score = %v, want 1", got.Breakdown["madeup"])
	}
	if got.Breakdown["other"] != 1 {
		t.Errorf("other score = %v, want 1", got.Breakdown["other"])
	}
}

func TestContributionScore_UserWeightsOverrideAndPreserveDefaults(t *testing.T) {
	m := &ContributionScoreMetric{
		Weights: config.ScoringWeights{
			CommitByType: map[string]float64{
				"feat": 5, // override
			},
		},
	}
	commits := []model.ClassifiedCommit{
		mkCommit("feat"), // user-weighted 5
		mkCommit("fix"),  // default 2
	}
	got := resultScore(t, m.Compute(commits))

	if got.Breakdown["feat"] != 5 {
		t.Errorf("feat = %v, want 5", got.Breakdown["feat"])
	}
	if got.Breakdown["fix"] != 2 {
		t.Errorf("fix = %v, want 2", got.Breakdown["fix"])
	}
	if got.Total != 7 {
		t.Errorf("Total = %v, want 7", got.Total)
	}
}

func TestContributionScore_CommitCountWeightMultiplies(t *testing.T) {
	m := &ContributionScoreMetric{
		Weights: config.ScoringWeights{
			CommitCount: 2,
		},
	}
	commits := []model.ClassifiedCommit{
		mkCommit("feat"), // 3 * 2 = 6
		mkCommit("fix"),  // 2 * 2 = 4
	}
	got := resultScore(t, m.Compute(commits))

	if got.Total != 10 {
		t.Errorf("Total = %v, want 10", got.Total)
	}
	if got.Breakdown["feat"] != 6 || got.Breakdown["fix"] != 4 {
		t.Errorf("unexpected breakdown: %v", got.Breakdown)
	}
}

func TestContributionScore_BreakdownSumsToTotal(t *testing.T) {
	m := &ContributionScoreMetric{}
	commits := []model.ClassifiedCommit{
		mkCommit("feat"), mkCommit("feat"),
		mkCommit("fix"),
		mkCommit("docs"),
		mkCommit("chore"),
	}
	got := resultScore(t, m.Compute(commits))

	var sum float64
	for _, v := range got.Breakdown {
		sum += v
	}
	if sum != got.Total {
		t.Errorf("breakdown sum %v != Total %v", sum, got.Total)
	}
}

func TestContributionScore_ZeroCommitCountWeightDefaultsToOne(t *testing.T) {
	// Regression: a missing/zero CommitCount weight should not zero out scores.
	m := &ContributionScoreMetric{
		Weights: config.ScoringWeights{CommitCount: 0},
	}
	got := resultScore(t, m.Compute([]model.ClassifiedCommit{mkCommit("feat")}))

	if got.Total != 3 {
		t.Errorf("Total = %v, want 3 (default base 1.0)", got.Total)
	}
}
