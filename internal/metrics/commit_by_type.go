package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

type CommitByTypeMetric struct{}

func (m *CommitByTypeMetric) Name() string {
	return "commit_by_type"
}

func (m *CommitByTypeMetric) Compute(commits []model.ClassifiedCommit) Result {
	counts := make(map[string]int)

	for _, c := range commits {
		t := c.Type
		if t == "" {
			t = "unknown"
		}
		counts[t]++
	}

	return Result{
		Name:  m.Name(),
		Value: counts,
	}
}
