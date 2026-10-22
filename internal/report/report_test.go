package report

import (
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// sampleResults covers every value type the writers support
func sampleResults() []metrics.Result {
	return []metrics.Result{
		{Name: "commit_count", Value: 4},
		{Name: "commit_by_type", Value: map[string]int{"refactor": 1, "feat": 3, "test": 2, "fix": 1, "chore": 2}},
		{Name: "activity_timeline", Value: map[string]map[string]int{
			"weekly":  {"2025-W10": 1, "2025-W02": 1, "2025-W07": 1, "2025-W01": 1},
			"monthly": {"2025-03": 2, "2025-01": 1, "2025-02": 1},
		}},
		{Name: "merge_summaries", Value: []model.MergeSummary{{
			Hash:        "abc123",
			Message:     "Merge branch 'feature'",
			Timestamp:   time.Date(2025, 1, 2, 3, 4, 0, 0, time.UTC),
			CommitCount: 2,
			Types:       map[string]int{"feat": 2},
		}}},
		{Name: "contribution_score", Value: model.ContributionScore{
			Total:       11,
			Breakdown:   map[string]float64{"fix": 2, "feat": 9},
			CommitCount: 4,
		}},
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return rows
}

func TestWriteCSV(t *testing.T) {
	// The directory does not exist yet; the writer creates it
	dir := filepath.Join(t.TempDir(), "nested", "out")
	if err := WriteCSV(sampleResults(), dir); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	// Rows from maps are sorted by key so reports are stable between runs
	cases := []struct {
		file string
		want [][]string
	}{
		{"commit_count.csv", [][]string{{"metric", "value"}, {"commit_count", "4"}}},
		{"commit_by_type.csv", [][]string{
			{"key", "count"}, {"chore", "2"}, {"feat", "3"}, {"fix", "1"}, {"refactor", "1"}, {"test", "2"},
		}},
		{"activity_timeline_weekly.csv", [][]string{
			{"period", "count"}, {"2025-W01", "1"}, {"2025-W02", "1"}, {"2025-W07", "1"}, {"2025-W10", "1"},
		}},
		{"activity_timeline_monthly.csv", [][]string{
			{"period", "count"}, {"2025-01", "1"}, {"2025-02", "1"}, {"2025-03", "2"},
		}},
		{"merge_summaries.csv", [][]string{
			{"hash", "timestamp", "message", "commit_count", "types"},
			{"abc123", "2025-01-02T03:04:00Z", "Merge branch 'feature'", "2", "map[feat:2]"},
		}},
		{"contribution_score.csv", [][]string{
			{"component", "score"}, {"feat", "9.00"}, {"fix", "2.00"}, {"total", "11.00"}, {"commit_count", "4"},
		}},
	}
	for _, tc := range cases {
		got := readCSV(t, filepath.Join(dir, tc.file))
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s = %v, want %v", tc.file, got, tc.want)
		}
	}
}

func TestWriteXLSX(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	if err := WriteXLSX(sampleResults(), dir); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}

	f, err := excelize.OpenFile(filepath.Join(dir, "report.xlsx"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	sort.Strings(sheets)
	want := []string{
		"activity_timeline_monthly", "activity_timeline_weekly", "commit_by_type",
		"commit_count", "contribution_score", "merge_summaries",
	}
	if !reflect.DeepEqual(sheets, want) {
		t.Errorf("sheets = %v, want %v (no default Sheet1)", sheets, want)
	}

	for sheet, want := range map[string][]string{
		"commit_by_type":            {"key", "chore", "feat", "fix", "refactor", "test"},
		"activity_timeline_monthly": {"period", "2025-01", "2025-02", "2025-03"},
	} {
		cols, err := f.GetCols(sheet)
		if err != nil || len(cols) == 0 || !reflect.DeepEqual(cols[0], want) {
			t.Errorf("%s column A = %v, want sorted %v", sheet, cols, want)
		}
	}

	if v, _ := f.GetCellValue("commit_count", "B2"); v != "4" {
		t.Errorf("commit_count!B2 = %q, want 4", v)
	}
	if v, _ := f.GetCellValue("contribution_score", "B4"); v != "11" {
		t.Errorf("contribution_score!B4 (total) = %q, want 11", v)
	}
	if v, _ := f.GetCellValue("merge_summaries", "E2"); v != "map[feat:2]" {
		t.Errorf("merge_summaries E2 = %q, want map[feat:2]", v)
	}
	if v, _ := f.GetCellValue("merge_summaries", "B2"); v != "2025-01-02 03:04" {
		t.Errorf("merge_summaries!B2 = %q, want 2025-01-02 03:04", v)
	}
}

func TestWritePDF(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	if err := WritePDF(sampleResults(), dir); err != nil {
		t.Fatalf("WritePDF: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "report.pdf"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Errorf("report.pdf does not start with a PDF header")
	}
}

func TestWriters_UnsupportedValue(t *testing.T) {
	bad := []metrics.Result{{Name: "weird", Value: 1.5}}
	if err := WriteCSV(bad, t.TempDir()); err == nil {
		t.Error("WriteCSV: want error for unsupported value")
	}
	if err := WriteXLSX(bad, t.TempDir()); err == nil {
		t.Error("WriteXLSX: want error for unsupported value")
	}
}

// A failed write must be reported rather than leaving a truncated file behind
func TestWriteRows_ReportsWriteErrors(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full not available")
	}
	if err := writeRows("/dev/full", [][]string{{"metric", "value"}}); err == nil {
		t.Error("writeRows(/dev/full): want error")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(out)
}

func TestWriteSummary(t *testing.T) {
	out := captureStdout(t, func() { WriteSummary(sampleResults()) })
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")

	for _, i := range []int{0, len(lines) - 1} {
		if n := len([]rune(lines[i])); n != lineWidth {
			t.Errorf("line %d is %d runes wide, want %d: %q", i, n, lineWidth, lines[i])
		}
	}

	for _, want := range []string{
		"Commit Count         4",
		"Commit By Type       chore: 2  feat: 3  fix: 1  refactor: 1  test: 2",
		"Activity Timeline (monthly) 2025-01: 1  2025-02: 1  2025-03: 2",
		"Merge Summaries      1 merges",
		"Contribution         11.0  (4 commits)",
		"feat: 9  fix: 2",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestFormatLabel(t *testing.T) {
	cases := map[string]string{
		"commit_count":   "Commit Count",
		"files":          "Files",
		"double__under_": "Double  Under ",
	}
	for in, want := range cases {
		if got := formatLabel(in); got != want {
			t.Errorf("formatLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
