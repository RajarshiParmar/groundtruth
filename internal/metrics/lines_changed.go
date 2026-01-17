package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

type LinesChangedMetric struct{}

func (m *LinesChangedMetric) Name() string {
	return "lines_changed"
}

func (m *LinesChangedMetric) Compute(commits []model.ClassifiedCommit) Result {
	added := 0
	removed := 0

	for _, c := range commits {
		added += c.LinesAdded
		removed += c.LinesRemoved
	}

	return Result{
		Name: m.Name(),
		Value: map[string]int{
			"added":   added,
			"removed": removed,
			"net":     added - removed,
		},
	}
}
