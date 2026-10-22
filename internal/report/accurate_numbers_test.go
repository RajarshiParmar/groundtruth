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

// pairedHighlights has one paired item with a ticket and one paired breaking change
func pairedHighlights() model.Highlights {
	ts := time.Date(2025, 3, 5, 12, 0, 0, 0, time.UTC)
	sso := model.Highlight{Hash: "a1", Type: "feat", Scope: "auth", Subject: "add SSO login", Timestamp: ts, Tickets: []string{"PROJ-12"}, Paired: true}
	v1 := model.Highlight{Hash: "b2", Type: "feat", Scope: "api", Subject: "drop v1", Breaking: true, Timestamp: ts, Paired: true}
	return model.Highlights{
		Groups: []model.HighlightGroup{
			{Scope: "auth", Counts: map[string]int{"feat": 1}, Items: []model.Highlight{sso}},
			{Scope: "api", Counts: map[string]int{"feat": 1}, Items: []model.Highlight{v1}},
		},
		Breaking:           []model.Highlight{v1},
		Classified:         2,
		Total:              2,
		IncludesCoAuthored: true,
	}
}

func TestPairedSuffix(t *testing.T) {
	if got := pairedSuffix(model.Highlight{Paired: true}); got != " (paired)" {
		t.Errorf("paired: %q", got)
	}
	if got := pairedSuffix(model.Highlight{}); got != "" {
		t.Errorf("not paired: %q", got)
	}
}

func TestWriteSummary_Paired(t *testing.T) {
	out := captureStdout(t, func() {
		WriteSummary([]metrics.Result{{Name: "highlights", Value: pairedHighlights()}})
	})
	for _, want := range []string{
		"      - add SSO login (paired) [PROJ-12]\n",
		"      - api: drop v1 (paired) (2025-03-05)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestHighlightLinesPDF_Paired(t *testing.T) {
	lines := highlightLinesPDF(pairedHighlights())
	for _, want := range []string{"- add SSO login (paired) [PROJ-12]", "- api: drop v1 (paired) (2025-03-05)"} {
		if !slices.Contains(lines, pdfLine{"", want}) {
			t.Errorf("lines missing %q:\n%q", want, lines)
		}
	}
}

func TestHighlightBlocks_Paired(t *testing.T) {
	md := strings.Join(highlightBlocks(pairedHighlights()), "\n\n")
	for _, want := range []string{"- add SSO login (paired) (PROJ-12)\n", "- api: drop v1 (paired)"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestHighlightTable_PairedColumn(t *testing.T) {
	h := pairedHighlights()
	got := highlightTable(h)
	if want := []string{"date", "scope", "type", "breaking", "subject", "tickets", "hash", "paired"}; !reflect.DeepEqual(got.Header, want) {
		t.Errorf("header = %q, want %q", got.Header, want)
	}
	for _, row := range got.Rows {
		if p, ok := row[len(row)-1].(bool); !ok || !p || len(row) != 8 {
			t.Errorf("row %v: want 8 cells ending in true (bool)", row)
		}
	}

	h.IncludesCoAuthored = false
	got = highlightTable(h)
	if want := []string{"date", "scope", "type", "breaking", "subject", "tickets", "hash"}; !reflect.DeepEqual(got.Header, want) {
		t.Errorf("without co_authored: header = %q, want %q", got.Header, want)
	}
	if len(got.Rows[0]) != 7 {
		t.Errorf("without co_authored: row has %d cells, want 7", len(got.Rows[0]))
	}
}
