package report

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func sampleHighlights() model.Highlights {
	pst := time.FixedZone("PST", -8*3600)
	return model.Highlights{
		SummaryLimit: 1,
		Total:        3,
		Classified:   3,
		Groups: []model.HighlightGroup{
			{Scope: "auth", Counts: map[string]int{"feat": 2}, Items: []model.Highlight{
				// Quotes and commas must survive CSV; the date is the commit's own local day
				{Hash: "h2", Type: "feat", Scope: "auth", Subject: `say "hi", then go`,
					Timestamp: time.Date(2025, 3, 9, 23, 30, 0, 0, pst), Tickets: []string{"PROJ-1", "#4"}},
				{Hash: "h1", Type: "feat", Scope: "auth", Subject: "line one\nline two",
					Timestamp: time.Date(2025, 3, 1, 9, 0, 0, 0, time.UTC)},
			}},
			{Scope: "api", Counts: map[string]int{"refactor": 1}, Items: []model.Highlight{
				{Hash: "h3", Type: "refactor", Scope: "api", Subject: "drop v1", Breaking: true,
					Timestamp: time.Date(2025, 3, 5, 9, 0, 0, 0, time.UTC)},
			}},
		},
		Breaking: []model.Highlight{
			{Hash: "h3", Type: "refactor", Scope: "api", Subject: "drop v1", Breaking: true,
				Timestamp: time.Date(2025, 3, 5, 9, 0, 0, 0, time.UTC)},
		},
	}
}

func TestWriteCSV_Highlights(t *testing.T) {
	dir := t.TempDir()
	if err := WriteCSV([]metrics.Result{{Name: "highlights", Value: sampleHighlights()}}, dir); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	// The full list: SummaryLimit only applies to the short view
	want := [][]string{
		{"date", "scope", "type", "breaking", "subject", "tickets", "hash"},
		{"2025-03-09", "auth", "feat", "false", `say "hi", then go`, "PROJ-1 #4", "h2"},
		{"2025-03-01", "auth", "feat", "false", "line one\nline two", "", "h1"},
		{"2025-03-05", "api", "refactor", "true", "drop v1", "", "h3"},
	}
	if got := readCSV(t, filepath.Join(dir, "highlights.csv")); !reflect.DeepEqual(got, want) {
		t.Errorf("highlights.csv = %q, want %q", got, want)
	}
}

func TestWriteXLSX_Highlights(t *testing.T) {
	dir := t.TempDir()
	if err := WriteXLSX([]metrics.Result{{Name: "highlights", Value: sampleHighlights()}}, dir); err != nil {
		t.Fatalf("WriteXLSX: %v", err)
	}

	f, err := excelize.OpenFile(filepath.Join(dir, "report.xlsx"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	for cell, want := range map[string]string{
		"A1": "date", "G1": "hash",
		"A2": "2025-03-09", "E2": `say "hi", then go`, "F2": "PROJ-1 #4",
		"E3": "line one\nline two", "B4": "api",
	} {
		if got, _ := f.GetCellValue("highlights", cell); got != want {
			t.Errorf("highlights!%s = %q, want %q", cell, got, want)
		}
	}

	if typ, err := f.GetCellType("highlights", "D4"); err != nil || typ != excelize.CellTypeBool {
		t.Errorf("highlights!D4 type = %v, %v; want bool", typ, err)
	}
	if got, _ := f.GetCellValue("highlights", "D4"); got != "TRUE" {
		t.Errorf("highlights!D4 = %q, want TRUE", got)
	}
}
