package metrics

import (
	"reflect"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func inRepo(repo string, added, removed int, files ...string) model.ClassifiedCommit {
	return model.ClassifiedCommit{Commit: model.Commit{Repo: repo, LinesAdded: added, LinesRemoved: removed, FilesChanged: files}}
}

func breakdown(t *testing.T, m *RepoBreakdownMetric, commits []model.ClassifiedCommit) model.NumberTable {
	t.Helper()
	r := m.Compute(commits)
	if r.Name != "repo_breakdown" {
		t.Errorf("Name = %q", r.Name)
	}
	table, ok := r.Value.(model.NumberTable)
	if !ok {
		t.Fatalf("Value is %T, want model.NumberTable", r.Value)
	}
	return table
}

func TestRepoBreakdown_Columns(t *testing.T) {
	commits := []model.ClassifiedCommit{inRepo("a", 1, 0)}

	off := breakdown(t, &RepoBreakdownMetric{}, commits)
	if !reflect.DeepEqual(off.Columns, []string{"commits"}) || off.KeyHeader != "repo" || off.Caption != "" {
		t.Errorf("off: %+v", off)
	}

	on := breakdown(t, &RepoBreakdownMetric{LinesChanged: true, FilesChanged: true, CoAuthored: true}, commits)
	if want := []string{"commits", "lines_added", "lines_removed", "files", "co_authored"}; !reflect.DeepEqual(on.Columns, want) {
		t.Errorf("on: Columns = %q, want %q", on.Columns, want)
	}
}

func TestRepoBreakdown_Rows(t *testing.T) {
	paired := inRepo("cli", 1, 1, "x.go")
	paired.CoAuthored = true
	commits := []model.ClassifiedCommit{
		inRepo("web", 5, 1, "a.js", "b.js"),
		inRepo("api", 2, 0, "main.go"),
		inRepo("cli", 3, 2, "x.go", "y.go"),
		inRepo("web", 1, 1, "a.js"),
		inRepo("api", 4, 4, "main.go"),
		paired,
		inRepo("cli", 0, 0),
	}

	got := breakdown(t, &RepoBreakdownMetric{LinesChanged: true, FilesChanged: true, CoAuthored: true}, commits)
	want := []model.NumberRow{
		{Label: "cli", Values: []int{3, 4, 3, 2, 1}},
		{Label: "api", Values: []int{2, 6, 4, 1, 0}},
		{Label: "web", Values: []int{2, 6, 2, 2, 0}},
	}
	if !reflect.DeepEqual(got.Rows, want) {
		t.Errorf("rows = %+v, want %+v", got.Rows, want)
	}

	// The rows add up to the combined totals
	lines := (&LinesChangedMetric{}).Compute(commits).Value.(map[string]int)
	sums := make([]int, 3)
	for _, r := range got.Rows {
		for i := range sums {
			sums[i] += r.Values[i]
		}
	}
	if sums[0] != len(commits) || sums[1] != lines["added"] || sums[2] != lines["removed"] {
		t.Errorf("sums = %v, want %d commits, %d added, %d removed", sums, len(commits), lines["added"], lines["removed"])
	}
}

func TestRepoCount(t *testing.T) {
	commits := []model.ClassifiedCommit{inRepo("a", 0, 0), inRepo("a", 0, 0), inRepo("b", 0, 0), inRepo("", 0, 0)}
	if got := RepoCount(commits); got != 2 {
		t.Errorf("RepoCount = %d, want 2", got)
	}
}
