package report

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestHighlightLinesPDF(t *testing.T) {
	want := []pdfLine{
		{"", "3 of 3 commits classified (100%)"},
		{"I", "auth  (feat: 2)"},
		{"", `- say "hi", then go [PROJ-1 #4]`},
		{"", "… 1 more"},
		{"I", "api  (refactor: 1)"},
		{"", "- drop v1"},
		{"B", "Breaking changes"},
		{"", "- api: drop v1 (2025-03-05)"},
	}
	if got := highlightLinesPDF(sampleHighlights()); !reflect.DeepEqual(got, want) {
		t.Errorf("lines =\n%q\nwant\n%q", got, want)
	}
}

func TestHighlightLinesPDF_Hints(t *testing.T) {
	got := highlightLinesPDF(model.Highlights{Total: 4})
	want := []pdfLine{{"", `none: 0 of 4 commits use Conventional Commits. Add "unknown" to highlights.types to include them.`}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nothing classified: %q, want %q", got, want)
	}

	low := model.Highlights{Total: 10, Classified: 3, Groups: []model.HighlightGroup{
		{Scope: "general", Counts: map[string]int{"feat": 1}, Items: []model.Highlight{{Subject: "x"}}},
	}}
	got = highlightLinesPDF(low)
	if len(got) < 2 || !strings.Contains(got[1].text, `Add "unknown" to highlights.types`) {
		t.Errorf("low coverage: %q, want the opt-in hint on the second line", got)
	}
}

func TestWritePDF_HighlightsUnicodeAndLongSubjects(t *testing.T) {
	h := sampleHighlights()
	h.Groups[0].Items[0].Subject = "café → naïve " + strings.Repeat("long ", 60)

	dir := t.TempDir()
	results := []metrics.Result{
		{Name: "highlights", Value: h},
		{Name: "merge_summaries", Value: []model.MergeSummary{{Message: "Merge « branch » " + strings.Repeat("x", 300)}}},
	}
	if err := WritePDF(results, dir); err != nil {
		t.Fatalf("WritePDF: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "report.pdf"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Error("report.pdf does not start with a PDF header")
	}
}
