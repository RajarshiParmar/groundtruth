package report

import (
	"fmt"
	"path/filepath"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/jung-kurt/gofpdf"
)

func WritePDF(results []metrics.Result, outputDir string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "groundtruth report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 11)

	for _, r := range results {
		writeMetricPDF(pdf, r)
		pdf.Ln(8)
	}

	path := filepath.Join(outputDir, "report.pdf")
	return pdf.OutputFileAndClose(path)
}

func writeMetricPDF(pdf *gofpdf.Fpdf, r metrics.Result) {
	pdf.SetFont("Arial", "B", 13)
	pdf.Cell(0, 8, r.Name)
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 11)

	switch v := r.Value.(type) {

	case int:
		pdf.Cell(0, 6, fmt.Sprintf("Value: %d", v))

	case map[string]int:
		writeKeyValuePDF(pdf, v)

	case map[string]map[string]int:
		writeTimelinePDF(pdf, v)

	case []model.MergeSummary:
		writeMergeSummaryPDF(pdf, v)

	default:
		pdf.Cell(0, 6, "Unsupported metric format")
	}
}

func writeKeyValuePDF(pdf *gofpdf.Fpdf, data map[string]int) {
	for k, v := range data {
		pdf.CellFormat(80, 6, k, "", 0, "", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("%d", v), "", 1, "", false, 0, "")
	}
}

func writeTimelinePDF(pdf *gofpdf.Fpdf, data map[string]map[string]int) {
	for period, values := range data {
		pdf.SetFont("Arial", "I", 11)
		pdf.Cell(0, 6, period)
		pdf.Ln(6)
		pdf.SetFont("Arial", "", 11)

		for k, v := range values {
			pdf.CellFormat(80, 6, k, "", 0, "", false, 0, "")
			pdf.CellFormat(0, 6, fmt.Sprintf("%d", v), "", 1, "", false, 0, "")
		}
		pdf.Ln(4)
	}
}

func writeMergeSummaryPDF(pdf *gofpdf.Fpdf, data []model.MergeSummary) {
	for _, m := range data {
		pdf.SetFont("Arial", "B", 11)
		pdf.Cell(0, 6, m.Message)
		pdf.Ln(6)

		pdf.SetFont("Arial", "", 10)
		pdf.Cell(0, 5, fmt.Sprintf("Commits: %d", m.CommitCount))
		pdf.Ln(5)
		pdf.Cell(0, 5, fmt.Sprintf("Types: %v", m.Types))
		pdf.Ln(7)
	}
}
