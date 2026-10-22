package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func readMarkdown(t *testing.T, results []metrics.Result, period string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "out")
	if err := WriteMarkdown(results, dir, period); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "report.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}

func TestMdEscape(t *testing.T) {
	cases := map[string]string{
		"fix *all* the [things]": `fix \*all\* the \[things\]`,
		"merge_commits.mode":     `merge\_commits.mode`,
		"use a|b":                `use a\|b`,
		"<b>bold</b>":            `\<b\>bold\</b\>`,
		`back\slash`:             `back\\slash`,
		"code `x`":               "code \\`x\\`",
		"two\nlines":             "two lines",
		"crlf\r\nline":           "crlf line",
		"plain #12, PROJ-3":      "plain #12, PROJ-3",
		"~~old~~ &copy;":         `\~\~old\~\~ \&copy;`,
	}
	for in, want := range cases {
		if got := mdEscape(in); got != want {
			t.Errorf("mdEscape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMdItem(t *testing.T) {
	// Text right after "- " must not start a heading, nested list or quote
	cases := map[string]string{
		"# 2.0 release": `\# 2.0 release`,
		"1. first pass": `1\. first pass`,
		"2) retry":      `2\) retry`,
		"- fixed typo":  `\- fixed typo`,
		"+ thing":       `\+ thing`,
		"> quoted":      `\> quoted`,
		"2024 roadmap":  "2024 roadmap",
		"fix #12 crash": "fix #12 crash",
		"":              "",
	}
	for in, want := range cases {
		if got := mdItem(in); got != want {
			t.Errorf("mdItem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMdHeading(t *testing.T) {
	// A trailing " #" would be read as a closing sequence and dropped
	cases := map[string]string{"c#": `c\#`, "foo #": `foo \#`, "a_b": `a\_b`, "auth": "auth"}
	for in, want := range cases {
		if got := mdHeading(in); got != want {
			t.Errorf("mdHeading(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteMarkdown_EscapesListMarkers(t *testing.T) {
	item := model.Highlight{Subject: "1. Initial import", Scope: "# core", Breaking: true}
	h := model.Highlights{Total: 1, Classified: 1,
		Groups:   []model.HighlightGroup{{Scope: "ui #", Counts: map[string]int{"unknown": 1}, Items: []model.Highlight{{Subject: "- Fixed login"}}}},
		Breaking: []model.Highlight{item},
	}
	got := readMarkdown(t, []metrics.Result{{Name: "highlights", Value: h}}, "p")
	for _, want := range []string{"### ui \\#\n", "\n- \\- Fixed login\n", "\n- \\# core: 1. Initial import\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

func TestWriteMarkdown(t *testing.T) {
	results := []metrics.Result{
		{Name: "commit_count", Value: 3},
		{Name: "commit_by_type", Value: map[string]int{"fix": 1, "feat": 2}},
		{Name: "tickets", Value: map[string]int{}},
		{Name: "activity_timeline", Value: map[string]map[string]int{"monthly": {"2025-03": 3}}},
		// Last in results, first in the file; SummaryLimit 1 but every item is listed
		{Name: "highlights", Value: sampleHighlights()},
	}

	want := "# groundtruth report\n\n2025-03-01 to 2025-03-31\n\n" +
		"## Highlights\n\n3 of 3 commits classified (100%)\n\n" +
		"### auth\n\nfeat: 2\n\n" +
		"- say \"hi\", then go (PROJ-1, #4)\n" +
		"- line one line two\n\n" +
		"### api\n\nrefactor: 1\n\n- drop v1\n\n" +
		"### Breaking changes\n\n- api: drop v1\n\n" +
		"## Commit Count\n\nCommit Count: **3**\n\n" +
		"## Commit By Type\n\n| key | count |\n|---|---|\n| feat | 2 |\n| fix | 1 |\n\n" +
		"## Tickets\n\n| key | count |\n|---|---|\n\n" +
		"## Activity Timeline\n\n### monthly\n\n| period | count |\n|---|---|\n| 2025-03 | 3 |\n"

	if got := readMarkdown(t, results, "2025-03-01 to 2025-03-31"); got != want {
		t.Errorf("report.md =\n%s\nwant\n%s", got, want)
	}
}

func TestWriteMarkdown_EscapesTableCells(t *testing.T) {
	results := []metrics.Result{{Name: "merge_summaries", Value: []model.MergeSummary{{
		Hash: "h1", Message: "Merge a|b *now*", CommitCount: 1,
		Timestamp: time.Date(2025, 3, 2, 9, 0, 0, 0, time.UTC), Types: map[string]int{"feat": 1, "fix": 2},
	}}}}

	out := readMarkdown(t, results, "p")
	row := ""
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "| h1 ") {
			row = l
		}
	}
	if !strings.Contains(row, `Merge a\|b \*now\*`) {
		t.Fatalf("message not escaped in row %q", row)
	}
	// Unescaped pipes delimit cells: 5 columns means 6 delimiters
	if n := strings.Count(strings.ReplaceAll(row, `\|`, ""), "|"); n != 6 {
		t.Errorf("row %q has %d delimiters, want 6", row, n)
	}
	if !strings.HasSuffix(row, "| feat: 1, fix: 2 |") {
		t.Errorf("types cell not readable in %q", row)
	}
	if !strings.Contains(row, "| 2025-03-02 |") {
		t.Errorf("timestamp not formatted as a date in %q", row)
	}
}

func TestWriteMarkdown_HighlightStates(t *testing.T) {
	// No highlights result: no Highlights section at all
	got := readMarkdown(t, []metrics.Result{{Name: "commit_count", Value: 1}}, "p")
	if want := "# groundtruth report\n\np\n\n## Commit Count\n\nCommit Count: **1**\n"; got != want {
		t.Errorf("without highlights:\n%q\nwant\n%q", got, want)
	}

	// Nothing to list: only the status line
	got = readMarkdown(t, []metrics.Result{{Name: "highlights", Value: model.Highlights{Total: 4}}}, "p")
	want := "# groundtruth report\n\np\n\n## Highlights\n\n" +
		"none: 0 of 4 commits use Conventional Commits. Add \"unknown\" to highlights.types to include them.\n"
	if got != want {
		t.Errorf("empty highlights:\n%q\nwant\n%q", got, want)
	}

	// Low coverage: hint paragraph after the status; unscoped breaking change with a ticket
	item := model.Highlight{Subject: "drop v1", Breaking: true, Tickets: []string{"PROJ-9"}}
	low := model.Highlights{Total: 10, Classified: 3,
		Groups:   []model.HighlightGroup{{Scope: "general", Counts: map[string]int{"feat": 1}, Items: []model.Highlight{item}}},
		Breaking: []model.Highlight{item},
	}
	got = readMarkdown(t, []metrics.Result{{Name: "highlights", Value: low}}, "p")
	for _, want := range []string{
		"3 of 10 commits classified (30%)\n\nMost commits don't use Conventional Commits. Add \"unknown\" to highlights.types to include them.\n\n### general",
		"### Breaking changes\n\n- drop v1 (PROJ-9)\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("low coverage: missing %q in\n%s", want, got)
		}
	}
}

func TestWriteMarkdown_Errors(t *testing.T) {
	err := WriteMarkdown([]metrics.Result{{Name: "weird", Value: 1.5}}, t.TempDir(), "p")
	if err == nil || err.Error() != "unsupported metric output for weird" {
		t.Errorf("unsupported value: error %v", err)
	}

	if _, statErr := os.Stat("/dev/full"); statErr != nil {
		t.Skip("/dev/full not available")
	}
	if err := writeFile("/dev/full", []byte("# groundtruth report\n")); err == nil {
		t.Error("writeFile(/dev/full): want error")
	}
}
