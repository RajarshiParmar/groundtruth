package report

import (
	"reflect"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestTablesFor(t *testing.T) {
	ts := time.Date(2025, 1, 2, 3, 4, 0, 0, time.UTC)
	cases := []struct {
		r    metrics.Result
		want []table
	}{
		{metrics.Result{Name: "commit_count", Value: 4},
			[]table{{Header: []string{"metric", "value"}, Rows: [][]any{{"commit_count", 4}}}}},
		{metrics.Result{Name: "t", Value: map[string]int{"b": 2, "a": 1}},
			[]table{{Header: []string{"key", "count"}, Rows: [][]any{{"a", 1}, {"b", 2}}}}},
		{metrics.Result{Name: "tl", Value: map[string]map[string]int{"weekly": {"W1": 1}, "monthly": {"M1": 2}}},
			[]table{
				{Suffix: "_monthly", Header: []string{"period", "count"}, Rows: [][]any{{"M1", 2}}},
				{Suffix: "_weekly", Header: []string{"period", "count"}, Rows: [][]any{{"W1", 1}}},
			}},
		{metrics.Result{Name: "m", Value: []model.MergeSummary{
			{Hash: "h", Timestamp: ts, Message: "Merge", CommitCount: 2, Types: map[string]int{"feat": 2}},
		}},
			[]table{{Header: []string{"hash", "timestamp", "message", "commit_count", "types"},
				Rows: [][]any{{"h", ts, "Merge", 2, map[string]int{"feat": 2}}}}}},
		{metrics.Result{Name: "s", Value: model.ContributionScore{
			Total: 3, Breakdown: map[string]float64{"fix": 2, "feat": 1}, CommitCount: 2,
		}},
			[]table{{Header: []string{"component", "score"},
				Rows: [][]any{{"feat", 1.0}, {"fix", 2.0}, {"total", 3.0}, {"commit_count", 2}}}}},
	}
	for _, tc := range cases {
		got, err := tablesFor(tc.r)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("tablesFor(%s) = %#v, %v; want %#v", tc.r.Name, got, err, tc.want)
		}
	}
}

func TestTablesFor_Highlights(t *testing.T) {
	got, err := tablesFor(metrics.Result{Name: "highlights", Value: sampleHighlights()})
	if err != nil || len(got) != 1 {
		t.Fatalf("tablesFor = %v, %v; want one table", got, err)
	}
	// Breaking stays a bool so XLSX writes a real boolean cell
	if b, ok := got[0].Rows[2][3].(bool); !ok || !b {
		t.Errorf("breaking cell = %#v, want true (bool)", got[0].Rows[2][3])
	}
}

func TestTablesFor_Unsupported(t *testing.T) {
	_, err := tablesFor(metrics.Result{Name: "weird", Value: 1.5})
	if err == nil || err.Error() != "unsupported metric output for weird" {
		t.Errorf("error = %v, want unsupported metric output for weird", err)
	}
}

func TestCSVCell(t *testing.T) {
	cases := []struct {
		v    any
		want string
	}{
		{4, "4"},
		{2.5, "2.50"},
		{true, "true"},
		{"x", "x"},
		{time.Date(2025, 1, 2, 3, 4, 0, 0, time.UTC), "2025-01-02T03:04:00Z"},
	}
	for _, tc := range cases {
		if got := csvCell(tc.v); got != tc.want {
			t.Errorf("csvCell(%#v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}
