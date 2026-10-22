package metrics

import (
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestHighlights_ByRepo(t *testing.T) {
	feat := func(hash, repo string) model.ClassifiedCommit {
		return model.ClassifiedCommit{Commit: model.Commit{Hash: hash, Repo: repo}, Type: "feat", Scope: "auth", Subject: hash}
	}
	commits := []model.ClassifiedCommit{feat("w1", "web"), feat("a1", "api"), feat("a2", "api")}

	h := (&HighlightsMetric{}).Compute(commits).Value.(model.Highlights)
	if !h.ByRepo || len(h.Groups) != 2 {
		t.Fatalf("ByRepo %v, %d groups; want true, 2", h.ByRepo, len(h.Groups))
	}
	for i, want := range []string{"api", "web"} {
		g := h.Groups[i]
		if g.Repo != want || g.Scope != "auth" {
			t.Errorf("group %d = %s/%s, want %s/auth", i, g.Repo, g.Scope, want)
		}
		for _, it := range g.Items {
			if it.Repo != want {
				t.Errorf("group %d item %s Repo = %q", i, it.Hash, it.Repo)
			}
		}
	}

	single := (&HighlightsMetric{}).Compute(commits[1:]).Value.(model.Highlights)
	if single.ByRepo || len(single.Groups) != 1 || single.Groups[0].Repo != "" || single.Groups[0].Items[0].Repo != "" {
		t.Errorf("one repo: %+v; want no repo grouping and empty Repo fields", single)
	}
}
