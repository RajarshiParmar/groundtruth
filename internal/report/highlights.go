package report

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// highlightRows is the full highlights list shared by the CSV and XLSX
// writers: a header, then one row per item in group order.
func highlightRows(h model.Highlights) [][]string {
	rows := [][]string{{"date", "scope", "type", "breaking", "subject", "tickets", "hash"}}

	for _, g := range h.Groups {
		for _, it := range g.Items {
			rows = append(rows, []string{
				// The commit's own zone, so the date matches what its author saw
				it.Timestamp.Format("2006-01-02"),
				g.Scope,
				it.Type,
				strconv.FormatBool(it.Breaking),
				it.Subject,
				strings.Join(it.Tickets, " "),
				it.Hash,
			})
		}
	}

	return rows
}

const unknownHint = `Add "unknown" to highlights.types to include them.`

// highlightsStatus is the coverage line shared by the terminal and PDF
// writers. When empty is true the status explains why there is nothing to
// list; hint is set when most commits are unclassified.
func highlightsStatus(h model.Highlights) (status, hint string, empty bool) {
	switch {
	case h.Total == 0:
		return "no commits in range", "", true
	case len(h.Groups) == 0 && h.Classified == 0:
		return fmt.Sprintf("none: 0 of %d commits use Conventional Commits. %s", h.Total, unknownHint), "", true
	case len(h.Groups) == 0:
		return fmt.Sprintf("none: no commits match highlights.types (%d of %d commits classified)", h.Classified, h.Total), "", true
	}

	status = fmt.Sprintf("%d of %d commits classified (%d%%)", h.Classified, h.Total, h.Classified*100/h.Total)
	if h.Classified*2 < h.Total && !h.IncludesUnknown {
		hint = "Most commits don't use Conventional Commits. " + unknownHint
	}
	return status, hint, false
}

// moreCount is how many items the summary cut from g; Counts still covers them all
func moreCount(g model.HighlightGroup) int {
	n := 0
	for _, c := range g.Counts {
		n += c
	}
	return n - len(g.Items)
}

func ticketSuffix(tickets []string) string {
	if len(tickets) == 0 {
		return ""
	}
	return " [" + strings.Join(tickets, " ") + "]"
}

// breakingLabel is "scope: subject (date)", without the scope for unscoped commits
func breakingLabel(it model.Highlight) string {
	label := it.Subject
	if it.Scope != "" {
		label = it.Scope + ": " + label
	}
	return label + " (" + it.Timestamp.Format("2006-01-02") + ")"
}
