package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

type MergeSummariesMetric struct{}

func (m *MergeSummariesMetric) Name() string {
	return "merge_summaries"
}

func (m *MergeSummariesMetric) Compute(commits []model.ClassifiedCommit) Result {
	var summaries []model.MergeSummary

	// Walk commits in order; merge commits act as boundaries
	for i, c := range commits {
		if !c.IsMerge {
			continue
		}

		typeCounts := make(map[string]int)
		childCount := 0

		// Look backwards until previous merge or beginning
		for j := i - 1; j >= 0; j-- {
			if commits[j].IsMerge {
				break
			}
			childCount++
			t := commits[j].Type
			if t == "" {
				t = "unknown"
			}
			typeCounts[t]++
		}

		summaries = append(summaries, model.MergeSummary{
			Hash:        c.Hash,
			Message:     firstLine(c.Message),
			Timestamp:   c.Timestamp,
			CommitCount: childCount,
			Types:       typeCounts,
		})
	}

	return Result{
		Name:  m.Name(),
		Value: summaries,
	}
}

func firstLine(msg string) string {
	for i, c := range msg {
		if c == '\n' {
			return msg[:i]
		}
	}
	return msg
}
