package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

type CommitCountMetric struct{}

func (m *CommitCountMetric) Name() string {
	return "commit_count"
}

func (m *CommitCountMetric) Compute(commits []model.ClassifiedCommit) Result {
	return Result{
		Name:  m.Name(),
		Value: len(commits),
	}
}
