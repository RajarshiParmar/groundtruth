package metrics

import (
	"reflect"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/classify"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

var hlBase = time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)

func hlCommit(hash, typ, scope string, day int) model.ClassifiedCommit {
	c := commitAt(hash, typ, hlBase.AddDate(0, 0, day))
	c.Scope = scope
	c.Subject = "subject " + hash
	return c
}

func computeHighlights(t *testing.T, m *HighlightsMetric, commits []model.ClassifiedCommit) model.Highlights {
	t.Helper()
	r := m.Compute(commits)
	if r.Name != "highlights" {
		t.Errorf("Name = %q, want highlights", r.Name)
	}
	h, ok := r.Value.(model.Highlights)
	if !ok {
		t.Fatalf("value is %T, want model.Highlights", r.Value)
	}
	return h
}

// groupHashes renders groups as scope → item hashes for compact comparisons
func groupHashes(h model.Highlights) [][]string {
	var out [][]string
	for _, g := range h.Groups {
		row := []string{g.Scope}
		for _, it := range g.Items {
			row = append(row, it.Hash)
		}
		out = append(out, row)
	}
	return out
}

func TestHighlights_Selection(t *testing.T) {
	breaking := hlCommit("r4", "refactor", "api", 4)
	breaking.Breaking = true
	merge := hlCommit("m6", "unknown", "", 6)
	merge.IsMerge = true

	commits := []model.ClassifiedCommit{
		hlCommit("f1", "feat", "auth", 1),
		hlCommit("x2", "fix", "auth", 2),
		hlCommit("d3", "docs", "", 3),
		breaking,
		hlCommit("u5", "unknown", "", 5),
		merge,
	}

	h := computeHighlights(t, &HighlightsMetric{}, commits)

	if h.Total != 5 || h.Classified != 4 {
		t.Errorf("Total, Classified = %d, %d; want 5, 4 (merge excluded)", h.Total, h.Classified)
	}
	if want := [][]string{{"auth", "x2", "f1"}, {"api", "r4"}}; !reflect.DeepEqual(groupHashes(h), want) {
		t.Errorf("groups = %v, want %v", groupHashes(h), want)
	}
	if !reflect.DeepEqual(h.Groups[0].Counts, map[string]int{"feat": 1, "fix": 1}) {
		t.Errorf("auth counts = %v", h.Groups[0].Counts)
	}
	if len(h.Breaking) != 1 || h.Breaking[0].Hash != "r4" || !h.Breaking[0].Breaking {
		t.Errorf("Breaking = %+v, want r4", h.Breaking)
	}
	if h.SummaryLimit != DefaultHighlightsPerScope {
		t.Errorf("SummaryLimit = %d, want %d", h.SummaryLimit, DefaultHighlightsPerScope)
	}
	if it := h.Groups[0].Items[0]; it.Subject != "subject x2" || it.Type != "fix" || it.Scope != "auth" {
		t.Errorf("item = %+v, want subject/type/scope copied from the commit", it)
	}

	// Opting in to unclassified commits; the breaking change is always selected
	h = computeHighlights(t, &HighlightsMetric{Types: []string{"unknown"}}, commits)
	if want := [][]string{{"api", "r4"}, {GeneralScope, "u5"}}; !reflect.DeepEqual(groupHashes(h), want) {
		t.Errorf("unknown opt-in groups = %v, want %v", groupHashes(h), want)
	}
}

func TestHighlights_Ordering(t *testing.T) {
	commits := []model.ClassifiedCommit{
		hlCommit("b1", "feat", "beta", 1),
		hlCommit("z1", "feat", "zeta", 1),
		hlCommit("z3", "feat", "zeta", 3),
		hlCommit("a1", "feat", "alpha", 2),
		hlCommit("z2b", "feat", "zeta", 2),
		hlCommit("z2a", "feat", "zeta", 2),
	}

	h := computeHighlights(t, &HighlightsMetric{}, commits)

	// Most items first, then scope name; newest first, then hash
	want := [][]string{{"zeta", "z3", "z2a", "z2b", "z1"}, {"alpha", "a1"}, {"beta", "b1"}}
	if !reflect.DeepEqual(groupHashes(h), want) {
		t.Errorf("groups = %v, want %v", groupHashes(h), want)
	}
}

func TestHighlights_ScopeCasingMerges(t *testing.T) {
	c := classify.NewConventionalClassifier()
	commits := []model.ClassifiedCommit{
		c.Classify(model.Commit{Hash: "a", Message: "feat(Auth): a", Timestamp: hlBase}),
		c.Classify(model.Commit{Hash: "b", Message: "fix(auth): b", Timestamp: hlBase}),
	}

	h := computeHighlights(t, &HighlightsMetric{}, commits)
	if len(h.Groups) != 1 || h.Groups[0].Scope != "auth" || len(h.Groups[0].Items) != 2 {
		t.Errorf("groups = %v, want one auth group with 2 items", groupHashes(h))
	}
}

func TestHighlights_TicketsAndDefaults(t *testing.T) {
	c := hlCommit("t1", "feat", "", 1)
	c.Message = "feat: x PROJ-3 #4"
	commits := []model.ClassifiedCommit{c}

	h := computeHighlights(t, &HighlightsMetric{Matcher: classify.NewTicketMatcher([]string{"PROJ"})}, commits)
	if got := h.Groups[0].Items[0].Tickets; !reflect.DeepEqual(got, []string{"PROJ-3", "#4"}) {
		t.Errorf("Tickets = %v, want [PROJ-3 #4]", got)
	}

	h = computeHighlights(t, &HighlightsMetric{PerScope: 7}, commits)
	if h.Groups[0].Items[0].Tickets != nil || h.SummaryLimit != 7 {
		t.Errorf("nil matcher: Tickets %v, SummaryLimit %d; want nil, 7", h.Groups[0].Items[0].Tickets, h.SummaryLimit)
	}

	h = computeHighlights(t, &HighlightsMetric{}, nil)
	if h.Total != 0 || len(h.Groups) != 0 || len(h.Breaking) != 0 {
		t.Errorf("no commits: %+v, want empty", h)
	}
}

// Writers use IncludesUnknown to stop suggesting an opt-in the user already made
func TestHighlights_IncludesUnknown(t *testing.T) {
	commits := []model.ClassifiedCommit{hlCommit("u1", "unknown", "", 1)}
	if h := computeHighlights(t, &HighlightsMetric{}, commits); h.IncludesUnknown {
		t.Error("default types: IncludesUnknown = true, want false")
	}
	if h := computeHighlights(t, &HighlightsMetric{Types: []string{"feat", "unknown"}}, commits); !h.IncludesUnknown {
		t.Error(`types with "unknown": IncludesUnknown = false, want true`)
	}
}
