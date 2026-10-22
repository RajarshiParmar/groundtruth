package metrics

import (
	"cmp"
	"maps"
	"slices"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// RepoBreakdownMetric is one row per repository: where the work went. The
// switches add the columns of the metrics that are turned on.
type RepoBreakdownMetric struct {
	LinesChanged bool
	FilesChanged bool
	CoAuthored   bool
}

func (m *RepoBreakdownMetric) Name() string {
	return "repo_breakdown"
}

func (m *RepoBreakdownMetric) Compute(commits []model.ClassifiedCommit) Result {
	type totals struct {
		commits, added, removed, coAuthored int
		files                               map[string]bool
	}
	repos := make(map[string]*totals)
	for _, c := range commits {
		t, ok := repos[c.Repo]
		if !ok {
			t = &totals{files: make(map[string]bool)}
			repos[c.Repo] = t
		}
		t.commits++
		t.added += c.LinesAdded
		t.removed += c.LinesRemoved
		for _, f := range c.FilesChanged {
			t.files[f] = true
		}
		if c.CoAuthored {
			t.coAuthored++
		}
	}

	table := model.NumberTable{KeyHeader: "repo", Columns: []string{"commits"}}
	if m.LinesChanged {
		table.Columns = append(table.Columns, "lines_added", "lines_removed")
	}
	if m.FilesChanged {
		table.Columns = append(table.Columns, "files")
	}
	if m.CoAuthored {
		table.Columns = append(table.Columns, "co_authored")
	}

	for _, name := range slices.Sorted(maps.Keys(repos)) {
		t := repos[name]
		values := []int{t.commits}
		if m.LinesChanged {
			values = append(values, t.added, t.removed)
		}
		if m.FilesChanged {
			values = append(values, len(t.files))
		}
		if m.CoAuthored {
			values = append(values, t.coAuthored)
		}
		table.Rows = append(table.Rows, model.NumberRow{Label: name, Values: values})
	}

	// Most commits first; the sort is stable, so ties stay in name order
	slices.SortStableFunc(table.Rows, func(a, b model.NumberRow) int {
		return cmp.Compare(b.Values[0], a.Values[0])
	})

	return Result{Name: m.Name(), Value: table}
}

// RepoCount is the number of distinct repositories the commits come from
func RepoCount(commits []model.ClassifiedCommit) int {
	repos := make(map[string]bool)
	for _, c := range commits {
		if c.Repo != "" {
			repos[c.Repo] = true
		}
	}
	return len(repos)
}
