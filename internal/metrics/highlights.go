package metrics

import (
	"cmp"
	"slices"

	"github.com/RajarshiParmar/groundtruth/internal/classify"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// DefaultHighlightTypes are the commit types listed as highlights when
// highlights.types is not set. Breaking changes are always included.
var DefaultHighlightTypes = []string{"feat", "fix", "perf"}

// DefaultHighlightsPerScope is the number of subjects per scope in the short view
const DefaultHighlightsPerScope = 3

// HighlightsMetric lists the commits worth mentioning, grouped by scope.
// Zero values fall back to the defaults above; a nil Matcher finds no tickets.
type HighlightsMetric struct {
	Types    []string
	PerScope int
	Matcher  *classify.TicketMatcher

	// CoAuthored is identity.co_authored; writers then show which items were paired
	CoAuthored bool
}

func (m *HighlightsMetric) Name() string {
	return "highlights"
}

func (m *HighlightsMetric) Compute(commits []model.ClassifiedCommit) Result {
	types := m.Types
	if len(types) == 0 {
		types = DefaultHighlightTypes
	}
	limit := m.PerScope
	if limit == 0 {
		limit = DefaultHighlightsPerScope
	}

	h := model.Highlights{
		SummaryLimit:       limit,
		IncludesUnknown:    slices.Contains(types, "unknown"),
		IncludesCoAuthored: m.CoAuthored,
	}
	groups := make(map[string]*model.HighlightGroup)

	for _, c := range commits {
		if c.IsMerge {
			continue
		}
		h.Total++
		if c.Type != "unknown" {
			h.Classified++
		}

		if !c.Breaking && !slices.Contains(types, c.Type) {
			continue
		}

		item := model.Highlight{
			Hash:      c.Hash,
			Type:      c.Type,
			Scope:     c.Scope,
			Subject:   c.Subject,
			Breaking:  c.Breaking,
			Timestamp: c.Timestamp,
			Tickets:   m.Matcher.Find(c.Message),
			Paired:    c.CoAuthored,
		}

		scope := scopeOrGeneral(c.Scope)
		g, ok := groups[scope]
		if !ok {
			g = &model.HighlightGroup{Scope: scope, Counts: make(map[string]int)}
			groups[scope] = g
		}
		g.Counts[c.Type]++
		g.Items = append(g.Items, item)

		if c.Breaking {
			h.Breaking = append(h.Breaking, item)
		}
	}

	for _, g := range groups {
		slices.SortFunc(g.Items, newestFirst)
		h.Groups = append(h.Groups, *g)
	}
	slices.SortFunc(h.Groups, func(a, b model.HighlightGroup) int {
		if n := cmp.Compare(len(b.Items), len(a.Items)); n != 0 {
			return n
		}
		return cmp.Compare(a.Scope, b.Scope)
	})
	slices.SortFunc(h.Breaking, newestFirst)

	return Result{
		Name:  m.Name(),
		Value: h,
	}
}

// newestFirst orders by timestamp descending, then hash so ties are stable
func newestFirst(a, b model.Highlight) int {
	if n := b.Timestamp.Compare(a.Timestamp); n != 0 {
		return n
	}
	return cmp.Compare(a.Hash, b.Hash)
}
