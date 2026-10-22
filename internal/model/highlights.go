package model

import "time"

// Highlight is one commit worth mentioning in an appraisal
type Highlight struct {
	Hash      string
	Type      string
	Scope     string
	Subject   string
	Breaking  bool
	Timestamp time.Time
	Tickets   []string
}

type HighlightGroup struct {
	Scope  string         // "general" when the commit has no scope
	Counts map[string]int // highlights per type
	Items  []Highlight    // newest first
}

// Highlights is the typed result emitted by HighlightsMetric
type Highlights struct {
	Groups       []HighlightGroup // most items first, then scope name ascending
	Breaking     []Highlight      // newest first, any type
	Classified   int              // non-merge commits whose Type != "unknown"
	Total        int              // non-merge commits in the report
	SummaryLimit int              // items per group in the short view; 0 means no limit
}

// Summary returns the short view: each group's Items cut to SummaryLimit.
// Counts, Breaking, Classified and Total are kept in full, and the receiver
// is not modified.
func (h Highlights) Summary() Highlights {
	groups := make([]HighlightGroup, len(h.Groups))
	for i, g := range h.Groups {
		if h.SummaryLimit > 0 && len(g.Items) > h.SummaryLimit {
			g.Items = g.Items[:h.SummaryLimit]
		}
		groups[i] = g
	}
	h.Groups = groups
	return h
}
