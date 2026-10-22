package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

// GeneralScope groups commits that have no conventional commit scope
const GeneralScope = "general"

type CommitByScopeMetric struct{}

func (m *CommitByScopeMetric) Name() string {
	return "commit_by_scope"
}

// Compute counts classified, non-merge commits per scope. Unclassified commits
// have no scope at all and are already visible as "unknown" in commit_by_type.
func (m *CommitByScopeMetric) Compute(commits []model.ClassifiedCommit) Result {
	counts := make(map[string]int)

	for _, c := range commits {
		if c.IsMerge || c.Type == "unknown" {
			continue
		}
		counts[scopeOrGeneral(c.Scope)]++
	}

	return Result{
		Name:  m.Name(),
		Value: counts,
	}
}

func scopeOrGeneral(scope string) string {
	if scope == "" {
		return GeneralScope
	}
	return scope
}
