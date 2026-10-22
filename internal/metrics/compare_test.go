package metrics

import (
	"reflect"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestCompare_Rows(t *testing.T) {
	lists := []Result{
		{Name: "highlights", Value: model.Highlights{}},
		{Name: "repo_breakdown", Value: model.NumberTable{}},
		{Name: "contribution_score", Value: model.ContributionScore{Total: 9}},
		{Name: "activity_timeline", Value: map[string]map[string]int{"weekly": {"2025-W10": 3}}},
	}
	current := append([]Result{
		{Name: "commit_count", Value: 51},
		{Name: "commit_by_type", Value: map[string]int{"feat": 18, "fix": 3}},
	}, lists...)
	previous := append([]Result{
		{Name: "commit_count", Value: 42},
		{Name: "commit_by_type", Value: map[string]int{"feat": 12, "docs": 2}},
	}, lists...)

	got := Compare(current, previous, "A compared with B")
	want := model.NumberTable{
		Caption:   "A compared with B",
		KeyHeader: "metric",
		Columns:   []string{"current", "previous", "change"},
		Rows: []model.NumberRow{
			{Label: "commit_count", Values: []int{51, 42, 9}},
			{Label: "commit_by_type.docs", Values: []int{0, 2, -2}},
			{Label: "commit_by_type.feat", Values: []int{18, 12, 6}},
			{Label: "commit_by_type.fix", Values: []int{3, 0, 3}},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Compare =\n%+v\nwant\n%+v", got, want)
	}
}

func TestCompare_EmptyPrevious(t *testing.T) {
	current := []Result{{Name: "commit_count", Value: 3}, {Name: "lines_changed", Value: map[string]int{"added": 5, "net": 4, "removed": 1}}}
	previous := []Result{{Name: "commit_count", Value: 0}, {Name: "lines_changed", Value: map[string]int{"added": 0, "net": 0, "removed": 0}}}

	for _, r := range Compare(current, previous, "").Rows {
		if r.Values[1] != 0 || r.Values[2] != r.Values[0] {
			t.Errorf("%s = %v, want previous 0 and change = current", r.Label, r.Values)
		}
	}
	if got := len(Compare(current, previous, "").Rows); got != 4 {
		t.Errorf("%d rows, want 4", got)
	}
}

// Tickets compare as one total, the number of distinct tickets, not a row per ticket
func TestCompare_TicketsAsOneRow(t *testing.T) {
	current := []Result{{Name: "tickets", Value: map[string]int{"PROJ-1": 3, "PROJ-2": 1, "#7": 1}}}
	previous := []Result{{Name: "tickets", Value: map[string]int{"PROJ-1": 2, "PROJ-9": 4}}}

	got := Compare(current, previous, "").Rows
	if want := []model.NumberRow{{Label: "tickets", Values: []int{3, 2, 1}}}; !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %+v, want %+v", got, want)
	}
}
