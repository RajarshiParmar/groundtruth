package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

type FilesChangedMetric struct{}

func (m *FilesChangedMetric) Name() string {
	return "files_changed"
}

func (m *FilesChangedMetric) Compute(commits []model.ClassifiedCommit) Result {
	unique := make(map[string]bool)
	total := 0

	for _, c := range commits {
		for _, f := range c.FilesChanged {
			total++
			unique[f] = true
		}
	}

	return Result{
		Name: m.Name(),
		Value: map[string]int{
			"unique": len(unique),
			"total":  total,
		},
	}
}
