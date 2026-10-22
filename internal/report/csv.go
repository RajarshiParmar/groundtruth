package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
)

// WriteCSV writes one file per table: <metric><suffix>.csv
func WriteCSV(results []metrics.Result, outputDir string) error {
	if err := ensureDir(outputDir); err != nil {
		return err
	}

	for _, r := range results {
		tables, err := tablesFor(r)
		if err != nil {
			return err
		}
		for _, t := range tables {
			rows := [][]string{t.Header}
			for _, row := range t.Rows {
				cells := make([]string, len(row))
				for i, v := range row {
					cells[i] = csvCell(v)
				}
				rows = append(rows, cells)
			}
			if err := writeRows(filepath.Join(outputDir, r.Name+t.Suffix+".csv"), rows); err != nil {
				return err
			}
		}
	}

	return nil
}

// csvCell formats a table cell for CSV
func csvCell(v any) string {
	switch x := v.(type) {
	case int:
		return strconv.Itoa(x)
	case float64:
		return fmt.Sprintf("%.2f", x)
	case bool:
		return strconv.FormatBool(x)
	case time.Time:
		return x.Format(time.RFC3339)
	default:
		return fmt.Sprint(x)
	}
}

// writeRows writes a whole CSV file, reporting write, flush and close errors
// so a full disk cannot leave a truncated report behind silently.
func writeRows(path string, rows [][]string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	if err := csv.NewWriter(f).WriteAll(rows); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
