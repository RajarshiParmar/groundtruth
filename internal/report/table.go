package report

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// table is one tabular unit of output: a CSV file, an XLSX sheet or a
// Markdown table. Cells keep their type (int, float64, string, bool or
// time.Time, or map[string]int for per-type counts) so each writer can
// format them its own way.
type table struct {
	Suffix string // appended to the metric name: "" or "_weekly" / "_monthly"
	Header []string
	Rows   [][]any
}

// tablesFor turns a result into the tables the tabular writers render
func tablesFor(r metrics.Result) ([]table, error) {
	switch v := r.Value.(type) {

	case int:
		return []table{{Header: []string{"metric", "value"}, Rows: [][]any{{r.Name, v}}}}, nil

	case map[string]int:
		return []table{countTable("", "key", v)}, nil

	case map[string]map[string]int:
		var tables []table
		for _, period := range slices.Sorted(maps.Keys(v)) {
			tables = append(tables, countTable("_"+period, "period", v[period]))
		}
		return tables, nil

	case []model.MergeSummary:
		t := table{Header: []string{"hash", "timestamp", "message", "commit_count", "types"}}
		for _, m := range v {
			t.Rows = append(t.Rows, []any{m.Hash, m.Timestamp, m.Message, m.CommitCount, m.Types})
		}
		return []table{t}, nil

	case model.ContributionScore:
		t := table{Header: []string{"component", "score"}}
		for _, k := range slices.Sorted(maps.Keys(v.Breakdown)) {
			t.Rows = append(t.Rows, []any{k, v.Breakdown[k]})
		}
		t.Rows = append(t.Rows, []any{"total", v.Total}, []any{"commit_count", v.CommitCount})
		return []table{t}, nil

	case model.Highlights:
		return []table{highlightTable(v)}, nil

	case model.NumberTable:
		t := table{Header: append([]string{v.KeyHeader}, v.Columns...)}
		for _, r := range v.Rows {
			row := []any{r.Label}
			for _, n := range r.Values {
				row = append(row, n)
			}
			t.Rows = append(t.Rows, row)
		}
		return []table{t}, nil

	default:
		return nil, fmt.Errorf("unsupported metric output for %s", r.Name)
	}
}

// countTable renders a count map as one row per key, sorted by key
func countTable(suffix, keyHeader string, data map[string]int) table {
	t := table{Suffix: suffix, Header: []string{keyHeader, "count"}}
	for _, k := range slices.Sorted(maps.Keys(data)) {
		t.Rows = append(t.Rows, []any{k, data[k]})
	}
	return t
}

// highlightTable is the full highlights list: one row per item in group order
func highlightTable(h model.Highlights) table {
	t := table{Header: []string{"date", "scope", "type", "breaking", "subject", "tickets", "hash"}}
	if h.IncludesCoAuthored {
		t.Header = append(t.Header, "paired")
	}
	if h.ByRepo {
		t.Header = append([]string{"repo"}, t.Header...)
	}

	for _, g := range h.Groups {
		for _, it := range g.Items {
			row := []any{
				// The commit's own zone, so the date matches what its author saw
				it.Timestamp.Format("2006-01-02"),
				g.Scope,
				it.Type,
				it.Breaking,
				it.Subject,
				strings.Join(it.Tickets, " "),
				it.Hash,
			}
			if h.IncludesCoAuthored {
				row = append(row, it.Paired)
			}
			if h.ByRepo {
				row = append([]any{g.Repo}, row...)
			}
			t.Rows = append(t.Rows, row)
		}
	}

	return t
}
