package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

// FilesChangedMetric counts changed paths. ShowExcluded adds an "excluded"
// key with the unique paths exclude_paths left out.
type FilesChangedMetric struct {
	ShowExcluded bool
}

func (m *FilesChangedMetric) Name() string {
	return "files_changed"
}

func (m *FilesChangedMetric) Compute(commits []model.ClassifiedCommit) Result {
	unique := make(map[string]bool)
	excluded := make(map[string]bool)
	total := 0

	for _, c := range commits {
		for _, f := range c.FilesChanged {
			total++
			unique[f] = true
		}
		for _, f := range c.ExcludedFiles {
			excluded[f] = true
		}
	}

	value := map[string]int{
		"unique": len(unique),
		"total":  total,
	}
	if m.ShowExcluded {
		value["excluded"] = len(excluded)
	}

	return Result{
		Name:  m.Name(),
		Value: value,
	}
}
