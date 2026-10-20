package metrics

import (
	"reflect"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func commitAt(hash, typ string, ts time.Time) model.ClassifiedCommit {
	return model.ClassifiedCommit{Commit: model.Commit{Hash: hash, Timestamp: ts}, Type: typ}
}

func TestCommitCount(t *testing.T) {
	r := (&CommitCountMetric{}).Compute(make([]model.ClassifiedCommit, 3))
	if r.Name != "commit_count" || r.Value != 3 {
		t.Errorf("got %v, want commit_count = 3", r)
	}
}

func TestCommitByType(t *testing.T) {
	commits := []model.ClassifiedCommit{mkCommit("feat"), mkCommit("feat"), mkCommit("fix"), mkCommit("")}
	got := (&CommitByTypeMetric{}).Compute(commits).Value
	want := map[string]int{"feat": 2, "fix": 1, "unknown": 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLinesChanged(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{LinesAdded: 10, LinesRemoved: 4}},
		{Commit: model.Commit{LinesAdded: 1, LinesRemoved: 9}},
	}
	got := (&LinesChangedMetric{}).Compute(commits).Value
	want := map[string]int{"added": 11, "removed": 13, "net": -2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFilesChanged(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{FilesChanged: []string{"a.go", "b.go"}}},
		{Commit: model.Commit{FilesChanged: []string{"a.go"}}},
		{},
	}
	got := (&FilesChangedMetric{}).Compute(commits).Value
	want := map[string]int{"unique": 2, "total": 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestActivityTimeline(t *testing.T) {
	commits := []model.ClassifiedCommit{
		// 2024-12-30 belongs to ISO week 1 of 2025
		commitAt("a", "feat", time.Date(2024, 12, 30, 9, 0, 0, 0, time.UTC)),
		commitAt("b", "feat", time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC)),
		commitAt("c", "fix", time.Date(2025, 2, 10, 9, 0, 0, 0, time.UTC)),
	}
	got := (&ActivityTimelineMetric{}).Compute(commits).Value
	want := map[string]map[string]int{
		"weekly":  {"2025-W01": 2, "2025-W07": 1},
		"monthly": {"2024-12": 1, "2025-01": 1, "2025-02": 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMergeSummaries(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	inner := commitAt("inner", "", ts)
	inner.IsMerge = true

	merge := commitAt("m", "", ts)
	merge.IsMerge = true
	merge.Message = "Merge branch 'feature'\n\ndetails"
	// "elsewhere" was authored by someone else or outside the range, so it is not in the report
	merge.MergedHashes = []string{"a", "b", "c", "inner", "elsewhere"}

	commits := []model.ClassifiedCommit{
		commitAt("a", "feat", ts),
		commitAt("b", "feat", ts),
		commitAt("c", "", ts),
		inner,
		merge,
	}

	got, ok := (&MergeSummariesMetric{}).Compute(commits).Value.([]model.MergeSummary)
	if !ok {
		t.Fatalf("value is not []model.MergeSummary")
	}
	if len(got) != 2 {
		t.Fatalf("got %d summaries, want one per merge (2)", len(got))
	}

	s := got[1]
	if s.Hash != "m" || s.Message != "Merge branch 'feature'" {
		t.Errorf("summary = %s %q, want m with header only", s.Hash, s.Message)
	}
	if s.CommitCount != 3 {
		t.Errorf("CommitCount = %d, want 3 (merges and commits outside the report excluded)", s.CommitCount)
	}
	if want := map[string]int{"feat": 2, "unknown": 1}; !reflect.DeepEqual(s.Types, want) {
		t.Errorf("Types = %v, want %v", s.Types, want)
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.Register(&CommitCountMetric{})

	if m, ok := r.Get("commit_count"); !ok || m.Name() != "commit_count" {
		t.Errorf("Get(commit_count) = %v, %v", m, ok)
	}
	if _, ok := r.Get("missing"); ok {
		t.Error("Get(missing): want not found")
	}
}
