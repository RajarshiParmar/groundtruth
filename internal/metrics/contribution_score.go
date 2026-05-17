package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

// ContributionScoreMetric computes a weighted composite score from classified commits.
// When Weights is nil, sensible defaults are used.
type ContributionScoreMetric struct {
	Weights map[string]any
}

func (m *ContributionScoreMetric) Name() string {
	return "contribution_score"
}

func (m *ContributionScoreMetric) Compute(commits []model.ClassifiedCommit) Result {
	typeWeights := resolveTypeWeights(m.Weights)
	countWeight := resolveCountWeight(m.Weights)

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

		score := w * countWeight
		breakdown[t] += score
		total += score
	}

	return Result{
		Name: m.Name(),
		Value: map[string]any{
			"total":        total,
			"breakdown":    breakdown,
			"commit_count": len(commits),
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

// resolveTypeWeights extracts per-type weights from the scoring config,
// falling back to defaults for any missing types.
func resolveTypeWeights(weights map[string]any) map[string]float64 {
	defaults := DefaultTypeWeights()

	if weights == nil {
		return defaults
	}

	raw, ok := weights["commit_by_type"]
	if !ok {
		return defaults
	}

	userMap, ok := raw.(map[string]any)
	if !ok {
		return defaults
	}

	// Merge user weights on top of defaults
	for k, v := range userMap {
		switch val := v.(type) {
		case int:
			defaults[k] = float64(val)
		case float64:
			defaults[k] = val
		}
	}

	return defaults
}

// resolveCountWeight extracts the flat per-commit multiplier from scoring config.
// Defaults to 1.0 if not specified.
func resolveCountWeight(weights map[string]any) float64 {
	if weights == nil {
		return 1.0
	}

	raw, ok := weights["commit_count"]
	if !ok {
		return 1.0
	}

	switch val := raw.(type) {
	case int:
		return float64(val)
	case float64:
		return val
	default:
		return 1.0
	}
}
