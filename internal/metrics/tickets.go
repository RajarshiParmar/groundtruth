package metrics

import (
	"github.com/RajarshiParmar/groundtruth/internal/classify"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

type TicketsMetric struct {
	Matcher *classify.TicketMatcher
}

func (m *TicketsMetric) Name() string {
	return "tickets"
}

// Compute counts, for each ticket, the commits that mention it. Merge
// messages are included so pull request numbers are picked up.
func (m *TicketsMetric) Compute(commits []model.ClassifiedCommit) Result {
	counts := make(map[string]int)

	for _, c := range commits {
		for _, ref := range m.Matcher.Find(c.Message) {
			counts[ref]++
		}
	}

	return Result{
		Name:  m.Name(),
		Value: counts,
	}
}
