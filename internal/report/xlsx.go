package report

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
)

// WriteXLSX writes report.xlsx with one sheet per table: <metric><suffix>
func WriteXLSX(results []metrics.Result, outputDir string) error {
	if err := ensureDir(outputDir); err != nil {
		return err
	}

	f := excelize.NewFile()

	for _, r := range results {
		tables, err := tablesFor(r)
		if err != nil {
			return err
		}
		for _, t := range tables {
			if err := writeSheet(f, r.Name+t.Suffix, t); err != nil {
				return err
			}
		}
	}

	// Remove the default sheet once real sheets exist (excelize keeps the last one)
	if len(f.GetSheetList()) > 1 {
		if err := f.DeleteSheet("Sheet1"); err != nil {
			return err
		}
		f.SetActiveSheet(0)
	}

	path := filepath.Join(outputDir, "report.xlsx")
	return f.SaveAs(path)
}

// writeSheet writes a table as typed cells, so numbers stay numbers and
// booleans stay booleans in the spreadsheet
func writeSheet(f *excelize.File, name string, t table) error {
	if _, err := f.NewSheet(name); err != nil {
		return err
	}

	header := make([]any, len(t.Header))
	for i, h := range t.Header {
		header[i] = h
	}
	rows := append([][]any{header}, t.Rows...)

	for i, row := range rows {
		values := make([]any, len(row))
		for j, v := range row {
			switch c := v.(type) {
			case time.Time:
				v = c.Format("2006-01-02 15:04")
			case map[string]int:
				v = fmt.Sprint(c)
			}
			values[j] = v
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow(name, cell, &values); err != nil {
			return err
		}
	}

	return nil
}
