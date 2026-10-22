package metrics

import (
	"reflect"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestLinesChanged_Excluded(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{LinesAdded: 2, ExcludedLines: 3}},
		{Commit: model.Commit{ExcludedLines: 4}},
	}

	got := (&LinesChangedMetric{ShowExcluded: true}).Compute(commits).Value
	if want := map[string]int{"added": 2, "removed": 0, "net": 2, "excluded": 7}; !reflect.DeepEqual(got, want) {
		t.Errorf("ShowExcluded: got %v, want %v", got, want)
	}

	got = (&LinesChangedMetric{}).Compute(commits).Value
	if want := map[string]int{"added": 2, "removed": 0, "net": 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("without ShowExcluded: got %v, want %v", got, want)
	}
}

func TestFilesChanged_Excluded(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{FilesChanged: []string{"main.go"}, ExcludedFiles: []string{"go.sum"}}},
		{Commit: model.Commit{ExcludedFiles: []string{"go.sum", "yarn.lock"}}},
	}

	got := (&FilesChangedMetric{ShowExcluded: true}).Compute(commits).Value
	if want := map[string]int{"unique": 1, "total": 1, "excluded": 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("ShowExcluded: got %v, want %v", got, want)
	}

	got = (&FilesChangedMetric{ShowExcluded: true}).Compute(commits[:0]).Value
	if want := map[string]int{"unique": 0, "total": 0, "excluded": 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("ShowExcluded, nothing excluded: got %v, want %v", got, want)
	}

	got = (&FilesChangedMetric{}).Compute(commits).Value
	if want := map[string]int{"unique": 1, "total": 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("without ShowExcluded: got %v, want %v", got, want)
	}
}

func TestCoAuthored(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{CoAuthored: true}},
		{Commit: model.Commit{}},
		{Commit: model.Commit{CoAuthored: true}},
	}
	if got := (&CoAuthoredMetric{}).Compute(commits); !reflect.DeepEqual(got, Result{Name: "co_authored", Value: 2}) {
		t.Errorf("got %v, want co_authored = 2", got)
	}
}

func TestHighlights_Paired(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{Hash: "a", CoAuthored: true}, Type: "feat", Subject: "paired"},
		{Commit: model.Commit{Hash: "b"}, Type: "fix", Subject: "solo"},
	}

	for _, on := range []bool{true, false} {
		h := (&HighlightsMetric{CoAuthored: on}).Compute(commits).Value.(model.Highlights)
		if h.IncludesCoAuthored != on {
			t.Errorf("CoAuthored %v: IncludesCoAuthored = %v", on, h.IncludesCoAuthored)
		}
		paired := map[string]bool{}
		for _, g := range h.Groups {
			for _, it := range g.Items {
				paired[it.Subject] = it.Paired
			}
		}
		if want := map[string]bool{"paired": true, "solo": false}; !reflect.DeepEqual(paired, want) {
			t.Errorf("CoAuthored %v: Paired = %v, want %v", on, paired, want)
		}
	}
}

// A commit that only touched excluded files still counts as work
func TestFullyExcludedCommitStillCounts(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{Hash: "a", ExcludedLines: 40, ExcludedFiles: []string{"go.sum"}}, Type: "feat", Subject: "bump deps"},
	}

	if got := (&CommitCountMetric{}).Compute(commits).Value; got != 1 {
		t.Errorf("commit_count = %v, want 1", got)
	}
	h := (&HighlightsMetric{}).Compute(commits).Value.(model.Highlights)
	if len(h.Groups) != 1 || len(h.Groups[0].Items) != 1 || h.Groups[0].Items[0].Subject != "bump deps" {
		t.Errorf("highlights = %+v, want the bump deps commit", h.Groups)
	}
}
