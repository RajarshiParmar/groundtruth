package metrics

import (
	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// ContributionScoreMetric computes a weighted composite score from classified
// commits. When Weights is the zero value, the built-in defaults are used.
type ContributionScoreMetric struct {
	Weights config.ScoringWeights
}

func (m *ContributionScoreMetric) Name() string {
	return "contribution_score"
}

func (m *ContributionScoreMetric) Compute(commits []model.ClassifiedCommit) Result {
	typeWeights := mergedTypeWeights(m.Weights.CommitByType)

	base := m.Weights.CommitCount
	if base == 0 {
		base = 1.0
	}

	breakdown := make(map[string]float64)
	var total float64

	for _, c := range commits {
		t := c.Type
		if t == "" {
			t = "other"
		}

		w, ok := typeWeights[t]
		if !ok {
			w = typeWeights["other"]
		}

		score := w * base
		breakdown[t] += score
		total += score
	}

	return Result{
		Name: m.Name(),
		Value: model.ContributionScore{
			Total:       total,
			Breakdown:   breakdown,
			CommitCount: len(commits),
		},
	}
}

// DefaultTypeWeights returns the built-in weights per conventional commit type.
func DefaultTypeWeights() map[string]float64 {
	return map[string]float64{
		"feat":     3,
		"fix":      2,
		"refactor": 2,
		"docs":     1,
		"chore":    1,
		"test":     1,
		"style":    1,
		"perf":     2,
		"ci":       1,
		"build":    1,
		"other":    1,
	}
}

// mergedTypeWeights overlays user-supplied weights on top of defaults so that
// unspecified types retain sensible defaults.
func mergedTypeWeights(user map[string]float64) map[string]float64 {
	out := DefaultTypeWeights()
	for k, v := range user {
		out[k] = v
	}
	return out
}
