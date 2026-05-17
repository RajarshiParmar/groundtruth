package report

import (
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func WriteXLSX(results []metrics.Result, outputDir string) error {
	f := excelize.NewFile()

	// Remove default sheet
	f.DeleteSheet("Sheet1")

	for _, r := range results {
		if err := writeMetricSheet(f, r); err != nil {
			return err
		}
	}

	path := filepath.Join(outputDir, "report.xlsx")
	return f.SaveAs(path)
}

func writeMetricSheet(f *excelize.File, r metrics.Result) error {
	switch v := r.Value.(type) {

	case int:
		return writeSingleValueSheet(f, r.Name, v)

	case map[string]int:
		return writeKeyValueSheet(f, r.Name, v)

	case map[string]map[string]int:
		return writeTimelineSheets(f, r.Name, v)

	case []model.MergeSummary:
		return writeMergeSummarySheet(f, r.Name, v)

	case map[string]any:
		return writeContributionScoreSheet(f, r.Name, v)

	default:
		return fmt.Errorf("unsupported metric output for %s", r.Name)
	}
}

func writeSingleValueSheet(f *excelize.File, name string, value int) error {
	f.NewSheet(name)
	f.SetCellValue(name, "A1", "metric")
	f.SetCellValue(name, "B1", "value")
	f.SetCellValue(name, "A2", name)
	f.SetCellValue(name, "B2", value)
	return nil
}

func writeKeyValueSheet(f *excelize.File, name string, data map[string]int) error {
	f.NewSheet(name)
	f.SetCellValue(name, "A1", "key")
	f.SetCellValue(name, "B1", "count")

	row := 2
	for k, v := range data {
		f.SetCellValue(name, fmt.Sprintf("A%d", row), k)
		f.SetCellValue(name, fmt.Sprintf("B%d", row), v)
		row++
	}
	return nil
}

func writeTimelineSheets(f *excelize.File, base string, data map[string]map[string]int) error {
	for period, values := range data {
		sheet := fmt.Sprintf("%s_%s", base, period)
		f.NewSheet(sheet)
		f.SetCellValue(sheet, "A1", "period")
		f.SetCellValue(sheet, "B1", "count")

		row := 2
		for k, v := range values {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), k)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), v)
			row++
		}
	}
	return nil
}

func writeMergeSummarySheet(f *excelize.File, name string, data []model.MergeSummary) error {
	f.NewSheet(name)

	headers := []string{
		"hash",
		"timestamp",
		"message",
		"commit_count",
		"types",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(name, cell, h)
	}

	row := 2
	for _, m := range data {
		f.SetCellValue(name, fmt.Sprintf("A%d", row), m.Hash)
		f.SetCellValue(name, fmt.Sprintf("B%d", row), m.Timestamp.Format("2006-01-02 15:04"))
		f.SetCellValue(name, fmt.Sprintf("C%d", row), m.Message)
		f.SetCellValue(name, fmt.Sprintf("D%d", row), m.CommitCount)
		f.SetCellValue(name, fmt.Sprintf("E%d", row), fmt.Sprintf("%v", m.Types))
		row++
	}

	return nil
}

func writeContributionScoreSheet(f *excelize.File, name string, data map[string]any) error {
	f.NewSheet(name)
	f.SetCellValue(name, "A1", "component")
	f.SetCellValue(name, "B1", "score")

	row := 2
	if breakdown, ok := data["breakdown"].(map[string]float64); ok {
		for k, v := range breakdown {
			f.SetCellValue(name, fmt.Sprintf("A%d", row), k)
			f.SetCellValue(name, fmt.Sprintf("B%d", row), v)
			row++
		}
	}

	if total, ok := data["total"].(float64); ok {
		f.SetCellValue(name, fmt.Sprintf("A%d", row), "total")
		f.SetCellValue(name, fmt.Sprintf("B%d", row), total)
		row++
	}

	if count, ok := data["commit_count"].(int); ok {
		f.SetCellValue(name, fmt.Sprintf("A%d", row), "commit_count")
		f.SetCellValue(name, fmt.Sprintf("B%d", row), count)
	}

	return nil
}
