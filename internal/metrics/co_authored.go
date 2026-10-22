package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

// CoAuthoredMetric counts the commits that name the user only as a co-author
type CoAuthoredMetric struct{}

func (m *CoAuthoredMetric) Name() string {
	return "co_authored"
}

func (m *CoAuthoredMetric) Compute(commits []model.ClassifiedCommit) Result {
	n := 0
	for _, c := range commits {
		if c.CoAuthored {
			n++
		}
	}
	return Result{Name: m.Name(), Value: n}
}
