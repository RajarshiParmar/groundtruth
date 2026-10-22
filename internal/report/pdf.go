package report

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/jung-kurt/gofpdf"
)

func WritePDF(results []metrics.Result, outputDir string) error {
	if err := ensureDir(outputDir); err != nil {
		return err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15)
	pdf.AddPage()

	// The core fonts only cover cp1252; translate UTF-8 commit text into it
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "groundtruth report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 11)

	for _, r := range results {
		writeMetricPDF(pdf, tr, r)
		pdf.Ln(8)
	}

	path := filepath.Join(outputDir, "report.pdf")
	return pdf.OutputFileAndClose(path)
}

func writeMetricPDF(pdf *gofpdf.Fpdf, tr func(string) string, r metrics.Result) {
	pdf.SetFont("Arial", "B", 13)
	pdf.Cell(0, 8, tr(r.Name))
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 11)

	switch v := r.Value.(type) {

	case int:
		pdf.Cell(0, 6, fmt.Sprintf("Value: %d", v))

	case map[string]int:
		writeKeyValuePDF(pdf, tr, v)

	case map[string]map[string]int:
		writeTimelinePDF(pdf, tr, v)

	case []model.MergeSummary:
		writeMergeSummaryPDF(pdf, tr, v)

	case model.ContributionScore:
		writeContributionScorePDF(pdf, tr, v)

	case model.Highlights:
		writeHighlightsPDF(pdf, tr, v)

	case model.NumberTable:
		// A fixed-width font keeps the columns aligned
		pdf.SetFont("Courier", "", 10)
		for _, l := range numberTableLines(v) {
			pdf.MultiCell(0, 5, tr(l), "", "L", false)
		}

	default:
		pdf.Cell(0, 6, "Unsupported metric format")
	}
}

func writeKeyValuePDF(pdf *gofpdf.Fpdf, tr func(string) string, data map[string]int) {
	for _, k := range slices.Sorted(maps.Keys(data)) {
		pdf.CellFormat(80, 6, tr(k), "", 0, "", false, 0, "")
		pdf.CellFormat(0, 6, fmt.Sprintf("%d", data[k]), "", 1, "", false, 0, "")
	}
}

func writeTimelinePDF(pdf *gofpdf.Fpdf, tr func(string) string, data map[string]map[string]int) {
	for _, period := range slices.Sorted(maps.Keys(data)) {
		pdf.SetFont("Arial", "I", 11)
		pdf.Cell(0, 6, tr(period))
		pdf.Ln(6)
		pdf.SetFont("Arial", "", 11)

		values := data[period]
		for _, k := range slices.Sorted(maps.Keys(values)) {
			pdf.CellFormat(80, 6, tr(k), "", 0, "", false, 0, "")
			pdf.CellFormat(0, 6, fmt.Sprintf("%d", values[k]), "", 1, "", false, 0, "")
		}
		pdf.Ln(4)
	}
}

func writeMergeSummaryPDF(pdf *gofpdf.Fpdf, tr func(string) string, data []model.MergeSummary) {
	for _, m := range data {
		// MultiCell wraps long merge messages instead of running off the page
		pdf.SetFont("Arial", "B", 11)
		pdf.MultiCell(0, 6, tr(m.Message), "", "L", false)

		pdf.SetFont("Arial", "", 10)
		pdf.Cell(0, 5, fmt.Sprintf("Commits: %d", m.CommitCount))
		pdf.Ln(5)
		pdf.Cell(0, 5, tr(fmt.Sprintf("Types: %v", m.Types)))
		pdf.Ln(7)
	}
}

func writeContributionScorePDF(pdf *gofpdf.Fpdf, tr func(string) string, data model.ContributionScore) {
	pdf.SetFont("Arial", "B", 11)
	pdf.Cell(0, 6, fmt.Sprintf("Total Score: %.1f", data.Total))
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 5, fmt.Sprintf("Commits: %d", data.CommitCount))
	pdf.Ln(6)

	if len(data.Breakdown) == 0 {
		return
	}

	pdf.SetFont("Arial", "I", 10)
	pdf.Cell(0, 5, "Breakdown by type:")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	keys := make([]string, 0, len(data.Breakdown))
	for k := range data.Breakdown {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pdf.CellFormat(80, 5, tr(k), "", 0, "", false, 0, "")
		pdf.CellFormat(0, 5, fmt.Sprintf("%.1f", data.Breakdown[k]), "", 1, "", false, 0, "")
	}
}

// pdfLine is one line of a PDF section; style is a gofpdf font style ("", "B", "I")
type pdfLine struct {
	style string
	text  string
}

// highlightLinesPDF lays out the short highlights view, the same content the
// terminal summary shows.
func highlightLinesPDF(h model.Highlights) []pdfLine {
	h = h.Summary()

	status, hint, empty := highlightsStatus(h)
	lines := []pdfLine{{"", status}}
	if empty {
		return lines
	}
	if hint != "" {
		lines = append(lines, pdfLine{"", hint})
	}

	for i, g := range h.Groups {
		if h.ByRepo && (i == 0 || h.Groups[i-1].Repo != g.Repo) {
			lines = append(lines, pdfLine{"B", g.Repo})
		}
		lines = append(lines, pdfLine{"I", g.Scope + "  (" + strings.Join(keyValueParts(g.Counts), "  ") + ")"})
		for _, it := range g.Items {
			lines = append(lines, pdfLine{"", "- " + it.Subject + pairedSuffix(it) + ticketSuffix(it.Tickets)})
		}
		if more := moreCount(g); more > 0 {
			lines = append(lines, pdfLine{"", fmt.Sprintf("… %d more", more)})
		}
	}

	if len(h.Breaking) > 0 {
		lines = append(lines, pdfLine{"B", "Breaking changes"})
		for _, it := range h.Breaking {
			lines = append(lines, pdfLine{"", "- " + breakingLabel(it)})
		}
	}

	return lines
}

func writeHighlightsPDF(pdf *gofpdf.Fpdf, tr func(string) string, h model.Highlights) {
	for _, l := range highlightLinesPDF(h) {
		pdf.SetFont("Arial", l.style, 10)
		// MultiCell wraps long subjects instead of running off the page
		pdf.MultiCell(0, 5, tr(l.text), "", "L", false)
	}
}
