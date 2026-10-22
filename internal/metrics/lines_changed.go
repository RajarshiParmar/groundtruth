package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

// LinesChangedMetric sums added and removed lines. ShowExcluded adds an
// "excluded" key with the lines exclude_paths left out.
type LinesChangedMetric struct {
	ShowExcluded bool
}

func (m *LinesChangedMetric) Name() string {
	return "lines_changed"
}

func (m *LinesChangedMetric) Compute(commits []model.ClassifiedCommit) Result {
	added := 0
	removed := 0
	excluded := 0

	for _, c := range commits {
		added += c.LinesAdded
		removed += c.LinesRemoved
		excluded += c.ExcludedLines
	}

	value := map[string]int{
		"added":   added,
		"removed": removed,
		"net":     added - removed,
	}
	if m.ShowExcluded {
		value["excluded"] = excluded
	}

	return Result{
		Name:  m.Name(),
		Value: value,
	}
}
