package metrics

import (
	"reflect"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/classify"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestCommitByScope(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Type: "feat", Scope: "auth"},
		{Type: "fix", Scope: "auth"},
		{Type: "feat"},
		{Type: "unknown"},
		{Commit: model.Commit{IsMerge: true}, Type: "unknown"},
	}
	r := (&CommitByScopeMetric{}).Compute(commits)
	want := map[string]int{"auth": 2, GeneralScope: 1}
	if r.Name != "commit_by_scope" || !reflect.DeepEqual(r.Value, want) {
		t.Errorf("got %s = %v, want commit_by_scope = %v", r.Name, r.Value, want)
	}
}

func TestTickets(t *testing.T) {
	commits := []model.ClassifiedCommit{
		{Commit: model.Commit{Message: "feat: a PROJ-1\n\nRefs: PROJ-1"}},
		{Commit: model.Commit{Message: "fix: b PROJ-1 #9"}},
		{Commit: model.Commit{Message: "Merge pull request #9 from x/y", IsMerge: true}},
	}

	m := &TicketsMetric{Matcher: classify.NewTicketMatcher([]string{"PROJ"})}
	r := m.Compute(commits)
	want := map[string]int{"PROJ-1": 2, "#9": 2}
	if r.Name != "tickets" || !reflect.DeepEqual(r.Value, want) {
		t.Errorf("got %s = %v, want tickets = %v", r.Name, r.Value, want)
	}

	// Without a matcher the result is an empty map, so writers still produce a file
	if got, ok := (&TicketsMetric{}).Compute(commits).Value.(map[string]int); !ok || got == nil || len(got) != 0 {
		t.Errorf("nil matcher: %#v, want empty map", got)
	}
}
