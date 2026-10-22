package report

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// byRepoHighlights has two repos; repo is the first one's name
func byRepoHighlights(repo string) model.Highlights {
	ts := time.Date(2025, 3, 5, 12, 0, 0, 0, time.UTC)
	login := model.Highlight{Hash: "a1", Repo: repo, Type: "feat", Scope: "auth", Subject: "add login", Timestamp: ts}
	sso := model.Highlight{Hash: "a2", Repo: repo, Type: "feat", Scope: "auth", Subject: "add SSO", Timestamp: ts}
	v1 := model.Highlight{Hash: "a3", Repo: repo, Type: "refactor", Scope: "auth", Subject: "drop v1", Breaking: true, Timestamp: ts}
	button := model.Highlight{Hash: "w1", Repo: "web", Type: "feat", Scope: "ui", Subject: "add button", Timestamp: ts}
	return model.Highlights{
		Groups: []model.HighlightGroup{
			{Repo: repo, Scope: "auth", Counts: map[string]int{"feat": 2, "refactor": 1}, Items: []model.Highlight{login, sso, v1}},
			{Repo: "web", Scope: "ui", Counts: map[string]int{"feat": 1}, Items: []model.Highlight{button}},
		},
		Breaking:   []model.Highlight{v1},
		Classified: 4,
		Total:      4,
		ByRepo:     true,
	}
}

func TestWriteSummary_ByRepo(t *testing.T) {
	out := captureStdout(t, func() {
		WriteSummary([]metrics.Result{{Name: "highlights", Value: byRepoHighlights("api")}})
	})
	for _, want := range []string{
		"    api\n      auth               feat: 2  refactor: 1\n        - add login\n",
		"    web\n      ui                 feat: 1\n        - add button\n",
		"      - api · auth: drop v1 (2025-03-05)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestHighlightLinesPDF_ByRepo(t *testing.T) {
	lines := highlightLinesPDF(byRepoHighlights("api"))
	i := slices.Index(lines, pdfLine{"B", "api"})
	if i < 0 || lines[i+1] != (pdfLine{"I", "auth  (feat: 2  refactor: 1)"}) {
		t.Errorf("want a bold api line before its scope:\n%q", lines)
	}
	if !slices.Contains(lines, pdfLine{"", "- api · auth: drop v1 (2025-03-05)"}) {
		t.Errorf("breaking item lacks the repo:\n%q", lines)
	}
}

func TestHighlightBlocks_ByRepo(t *testing.T) {
	md := strings.Join(highlightBlocks(byRepoHighlights("api")), "\n\n")
	for _, want := range []string{"### api\n\n#### auth\n\nfeat: 2 · refactor: 1\n\n- add login", "### web\n\n#### ui", "- api · auth: drop v1"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestHighlightBlocks_ByRepoEscapes(t *testing.T) {
	md := strings.Join(highlightBlocks(byRepoHighlights("my_repo")), "\n\n")
	for _, want := range []string{"### my\\_repo\n", "- my\\_repo · auth: drop v1"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestHighlightTable_RepoColumn(t *testing.T) {
	h := byRepoHighlights("api")
	got := highlightTable(h)
	if want := []string{"repo", "date", "scope", "type", "breaking", "subject", "tickets", "hash"}; !reflect.DeepEqual(got.Header, want) {
		t.Errorf("header = %q, want %q", got.Header, want)
	}
	if got.Rows[0][0] != "api" || got.Rows[3][0] != "web" {
		t.Errorf("repo cells = %v, %v", got.Rows[0][0], got.Rows[3][0])
	}

	h.ByRepo = false
	if got := highlightTable(h); !reflect.DeepEqual(got.Header, []string{"date", "scope", "type", "breaking", "subject", "tickets", "hash"}) {
		t.Errorf("without ByRepo: header = %q", got.Header)
	}
}
