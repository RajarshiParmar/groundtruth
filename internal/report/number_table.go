package report

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// numberTableLines lays out a number table for the terminal and PDF: the
// caption, a header, then labels left-aligned and numbers right-aligned
func numberTableLines(t model.NumberTable) []string {
	cells := numberTableCells(t)

	labelWidth := len(t.KeyHeader)
	for _, r := range t.Rows {
		labelWidth = max(labelWidth, len(r.Label))
	}
	widths := make([]int, len(t.Columns))
	for i, c := range t.Columns {
		widths[i] = len(c)
		for _, row := range cells {
			widths[i] = max(widths[i], len(row[i]))
		}
	}

	line := func(label string, values []string) string {
		var b strings.Builder
		fmt.Fprintf(&b, "%-*s", labelWidth, label)
		for i, v := range values {
			fmt.Fprintf(&b, "  %*s", widths[i], v)
		}
		return b.String()
	}

	var lines []string
	if t.Caption != "" {
		lines = append(lines, t.Caption)
	}
	lines = append(lines, line(t.KeyHeader, t.Columns))
	for i, r := range t.Rows {
		lines = append(lines, line(r.Label, cells[i]))
	}
	return lines
}

// numberTableCells formats the values of each row for text writers; only a
// comparison's change column is signed
func numberTableCells(t model.NumberTable) [][]string {
	signed := signedChange(t)
	cells := make([][]string, len(t.Rows))
	for i, r := range t.Rows {
		cells[i] = make([]string, len(r.Values))
		for j, v := range r.Values {
			if signed && j == len(t.Columns)-1 {
				cells[i][j] = formatSigned(v)
			} else {
				cells[i][j] = strconv.Itoa(v)
			}
		}
	}
	return cells
}

// signedChange reports whether the last column is a comparison's change
func signedChange(t model.NumberTable) bool {
	return t.KeyHeader == "metric" && len(t.Columns) > 0 && t.Columns[len(t.Columns)-1] == "change"
}

// formatSigned writes n with an explicit sign, except for zero
func formatSigned(n int) string {
	if n > 0 {
		return "+" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
