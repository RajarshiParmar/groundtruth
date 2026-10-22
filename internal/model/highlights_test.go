package model

import "testing"

func TestHighlightsSummary(t *testing.T) {
	h := Highlights{
		Groups: []HighlightGroup{
			{Scope: "auth", Counts: map[string]int{"feat": 4}, Items: make([]Highlight, 4)},
			{Scope: "general", Counts: map[string]int{"fix": 1}, Items: make([]Highlight, 1)},
		},
		Breaking:     make([]Highlight, 5),
		Classified:   8,
		Total:        10,
		SummaryLimit: 3,
	}

	s := h.Summary()

	if len(s.Groups[0].Items) != 3 || len(s.Groups[1].Items) != 1 {
		t.Errorf("items = %d, %d; want 3, 1", len(s.Groups[0].Items), len(s.Groups[1].Items))
	}
	if len(s.Breaking) != 5 || s.Groups[0].Counts["feat"] != 4 || s.Classified != 8 || s.Total != 10 {
		t.Errorf("summary dropped data: %+v", s)
	}

	// The original is untouched, including the Groups backing array
	s.Groups[0].Scope = "changed"
	if len(h.Groups[0].Items) != 4 || h.Groups[0].Scope != "auth" {
		t.Errorf("Summary modified the original: %+v", h.Groups[0])
	}
}

func TestHighlightsSummary_NoLimit(t *testing.T) {
	h := Highlights{Groups: []HighlightGroup{{Items: make([]Highlight, 4)}}}
	if got := len(h.Summary().Groups[0].Items); got != 4 {
		t.Errorf("SummaryLimit 0: %d items, want all 4", got)
	}
}
