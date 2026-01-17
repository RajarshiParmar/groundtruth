package metrics

import (
	"fmt"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

type ActivityTimelineMetric struct{}

func (m *ActivityTimelineMetric) Name() string {
	return "activity_timeline"
}

func (m *ActivityTimelineMetric) Compute(commits []model.ClassifiedCommit) Result {
	weekly := make(map[string]int)
	monthly := make(map[string]int)

	for _, c := range commits {
		t := c.Timestamp

		year, week := t.ISOWeek()
		weekKey := fmt.Sprintf("%04d-W%02d", year, week)
		weekly[weekKey]++

		monthKey := t.Format("2006-01")
		monthly[monthKey]++
	}

	return Result{
		Name: m.Name(),
		Value: map[string]map[string]int{
			"weekly":  weekly,
			"monthly": monthly,
		},
	}
}
