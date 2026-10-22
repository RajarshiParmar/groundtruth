package report

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/xuri/excelize/v2"
)

func sampleComparison() model.NumberTable {
	return model.NumberTable{
		Caption:   "A compared with B",
		KeyHeader: "metric",
		Columns:   []string{"current", "previous", "change"},
		Rows: []model.NumberRow{
			{Label: "commit_count", Values: []int{51, 42, 9}},
			{Label: "lines_changed.net", Values: []int{2140, 2530, -390}},
			{Label: "commit_by_type.feat", Values: []int{5, 5, 0}},
		},
	}
}

func sampleRepoTable() model.NumberTable {
	return model.NumberTable{
		KeyHeader: "repo",
		Columns:   []string{"commits", "change"},
		Rows:      []model.NumberRow{{Label: "api", Values: []int{4, 3}}},
	}
}

func TestSignedChange(t *testing.T) {
	for n, want := range map[int]string{9: "+9", -390: "-390", 0: "0"} {
		if got := formatSigned(n); got != want {
			t.Errorf("formatSigned(%d) = %q, want %q", n, got, want)
		}
	}
	if !signedChange(sampleComparison()) {
		t.Error("comparison: signedChange = false")
	}
	if signedChange(sampleRepoTable()) {
		t.Error("repo table: signedChange = true")
	}
}

func TestTablesFor_NumberTable(t *testing.T) {
	got, err := tablesFor(metrics.Result{Name: "comparison", Value: sampleComparison()})
	if err != nil || len(got) != 1 {
		t.Fatalf("tablesFor = %v, %v; want one table", got, err)
	}
	if want := []string{"metric", "current", "previous", "change"}; !reflect.DeepEqual(got[0].Header, want) {
		t.Errorf("header = %q, want %q", got[0].Header, want)
	}
	if want := []any{"commit_count", 51, 42, 9}; !reflect.DeepEqual(got[0].Rows[0], want) {
		t.Errorf("row 0 = %#v, want %#v", got[0].Rows[0], want)
	}
}

func TestNumberTableLines(t *testing.T) {
	want := []string{
		"A compared with B",
		"metric               current  previous  change",
		"commit_count              51        42      +9",
		"lines_changed.net       2140      2530    -390",
		"commit_by_type.feat        5         5       0",
	}
	if got := numberTableLines(sampleComparison()); !reflect.DeepEqual(got, want) {
		t.Errorf("lines =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	// No caption line, and a repo table's numbers are never signed
	want = []string{"repo  commits  change", "api         4       3"}
	if got := numberTableLines(sampleRepoTable()); !reflect.DeepEqual(got, want) {
		t.Errorf("repo lines =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestWriteSummary_NumberTable(t *testing.T) {
	out := captureStdout(t, func() {
		WriteSummary([]metrics.Result{{Name: "comparison", Value: sampleComparison()}})
	})
	for _, want := range []string{
		"  Comparison\n",
		"    A compared with B\n",
		"    lines_changed.net       2140      2530    -390\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestWriteMarkdown_NumberTable(t *testing.T) {
	dir := t.TempDir()
	if err := WriteMarkdown([]metrics.Result{{Name: "comparison", Value: sampleComparison()}}, dir, "P"); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	md := string(data)
	for _, want := range []string{
		"## Comparison\n\nA compared with B\n\n| metric |",
		"| commit\\_count | 51 | 42 | +9 |",
		"| lines\\_changed.net | 2140 | 2530 | -390 |",
		"| commit\\_by\\_type.feat | 5 | 5 | 0 |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report.md missing %q:\n%s", want, md)
		}
	}
}

func TestWriteCSV_NumberTable(t *testing.T) {
	dir := t.TempDir()
	if err := WriteCSV([]metrics.Result{{Name: "comparison", Value: sampleComparison()}}, dir); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	rows := readCSV(t, filepath.Join(dir, "comparison.csv"))
	want := [][]string{
		{"metric", "current", "previous", "change"},
		{"commit_count", "51", "42", "9"},
		{"lines_changed.net", "2140", "2530", "-390"},
		{"commit_by_type.feat", "5", "5", "0"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("comparison.csv = %q, want %q", rows, want)
	}
}

func TestWriteXLSX_NumberTable(t *testing.T) {
	dir := t.TempDir()
	if err := WriteXLSX([]metrics.Result{{Name: "comparison", Value: sampleComparison()}}, dir); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}
	f, err := excelize.OpenFile(filepath.Join(dir, "report.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for cell, want := range map[string]string{"A1": "metric", "B2": "51", "D3": "-390"} {
		if got, _ := f.GetCellValue("comparison", cell); got != want {
			t.Errorf("%s = %q, want %q", cell, got, want)
		}
	}
	// Numbers stay numbers, so the spreadsheet can sum them
	if typ, _ := f.GetCellType("comparison", "D3"); typ == excelize.CellTypeSharedString || typ == excelize.CellTypeInlineString {
		t.Errorf("D3 is stored as text (%v)", typ)
	}
}

func TestWritePDF_NumberTable(t *testing.T) {
	if err := WritePDF([]metrics.Result{{Name: "comparison", Value: sampleComparison()}}, t.TempDir()); err != nil {
		t.Fatalf("WritePDF: %v", err)
	}
}

// Long repo paths must not push the last column onto its own line
func TestPDFNumberTableFontSize(t *testing.T) {
	pdf := newReportPDF()
	width := pageTextWidth(pdf)

	wide := model.NumberTable{
		KeyHeader: "repo",
		Columns:   []string{"commits", "lines_added", "lines_removed", "files", "co_authored"},
		Rows:      []model.NumberRow{{Label: "clients/acme-corp/mobile-app-ios-v2", Values: []int{42, 3120, 980, 61, 0}}},
	}
	lines := numberTableLines(wide)
	size := pdfNumberTableFontSize(pdf, lines)
	if size >= 10 {
		t.Errorf("size = %v, want smaller than 10 for a %d-character line", size, len(lines[0]))
	}
	pdf.SetFont("Courier", "", size)
	for _, l := range lines {
		if w := pdf.GetStringWidth(l); w > width {
			t.Errorf("line is %.1fmm wide at %vpt, page text width is %.1fmm: %q", w, size, width, l)
		}
	}

	if got := pdfNumberTableFontSize(pdf, numberTableLines(sampleRepoTable())); got != 10 {
		t.Errorf("short table: size = %v, want 10", got)
	}
}

// Columns line up by character, so non-ASCII repo names do not shift a row
func TestNumberTableLines_NonASCII(t *testing.T) {
	table := model.NumberTable{
		KeyHeader: "repo",
		Columns:   []string{"commits"},
		Rows:      []model.NumberRow{{Label: "café", Values: []int{4}}, {Label: "api", Values: []int{2}}},
	}
	want := []string{"repo  commits", "café        4", "api         2"}
	if got := numberTableLines(table); !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}
