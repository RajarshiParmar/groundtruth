package metrics

import (
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

type MergeSummariesMetric struct{}

func (m *MergeSummariesMetric) Name() string {
	return "merge_summaries"
}

func (m *MergeSummariesMetric) Compute(commits []model.ClassifiedCommit) Result {
	byHash := make(map[string]model.ClassifiedCommit, len(commits))
	for _, c := range commits {
		byHash[c.Hash] = c
	}

	var summaries []model.MergeSummary

	for _, c := range commits {
		if !c.IsMerge {
			continue
		}

		typeCounts := make(map[string]int)
		childCount := 0

		// Only count merged commits that are in the report (author and time range)
		for _, h := range c.MergedHashes {
			child, ok := byHash[h]
			if !ok || child.IsMerge {
				continue
			}
			childCount++
			t := child.Type
			if t == "" {
				t = "unknown"
			}
			typeCounts[t]++
		}

		header, _, _ := strings.Cut(c.Message, "\n")

		summaries = append(summaries, model.MergeSummary{
			Hash:        c.Hash,
			Message:     header,
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
