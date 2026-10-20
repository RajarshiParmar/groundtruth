package report

import (
	"encoding/csv"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func WriteCSV(results []metrics.Result, outputDir string) error {
	if err := ensureDir(outputDir); err != nil {
		return err
	}

	for _, r := range results {
		if err := writeMetricCSV(r, outputDir); err != nil {
			return err
		}
	}

	return nil
}

func writeMetricCSV(r metrics.Result, dir string) error {
	switch v := r.Value.(type) {

	case int:
		return writeSingleValue(r.Name, v, dir)

	case map[string]int:
		return writeKeyValue(r.Name, v, dir)

	case map[string]map[string]int:
		return writeTimeline(r.Name, v, dir)

	case []model.MergeSummary:
		return writeMergeSummaries(r.Name, v, dir)

	case model.ContributionScore:
		return writeContributionScoreCSV(r.Name, v, dir)

	default:
		return fmt.Errorf("unsupported metric output for %s", r.Name)
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

func writeMergeSummaries(name string, data []model.MergeSummary, dir string) error {
	rows := [][]string{{"hash", "timestamp", "message", "commit_count", "types"}}

	for _, m := range data {
		rows = append(rows, []string{
			m.Hash,
			m.Timestamp.Format(time.RFC3339),
			m.Message,
			fmt.Sprintf("%d", m.CommitCount),
			fmt.Sprintf("%v", m.Types),
		})
	}

	return writeRows(filepath.Join(dir, name+".csv"), rows)
}

func writeSingleValue(name string, value int, dir string) error {
	return writeRows(filepath.Join(dir, name+".csv"), [][]string{
		{"metric", "value"},
		{name, fmt.Sprintf("%d", value)},
	})
}

func writeKeyValue(name string, data map[string]int, dir string) error {
	return writeRows(filepath.Join(dir, name+".csv"), countRows("key", data))
}

func writeTimeline(name string, data map[string]map[string]int, dir string) error {
	for _, period := range slices.Sorted(maps.Keys(data)) {
		path := filepath.Join(dir, fmt.Sprintf("%s_%s.csv", name, period))
		if err := writeRows(path, countRows("period", data[period])); err != nil {
			return err
		}
	}

	return nil
}

// countRows renders a count map as a header plus one row per key, sorted by key
func countRows(keyHeader string, data map[string]int) [][]string {
	rows := [][]string{{keyHeader, "count"}}
	for _, k := range slices.Sorted(maps.Keys(data)) {
		rows = append(rows, []string{k, fmt.Sprintf("%d", data[k])})
	}
	return rows
}

func writeContributionScoreCSV(name string, data model.ContributionScore, dir string) error {
	rows := [][]string{{"component", "score"}}

	for _, k := range slices.Sorted(maps.Keys(data.Breakdown)) {
		rows = append(rows, []string{k, fmt.Sprintf("%.2f", data.Breakdown[k])})
	}

	rows = append(rows,
		[]string{"total", fmt.Sprintf("%.2f", data.Total)},
		[]string{"commit_count", fmt.Sprintf("%d", data.CommitCount)},
	)

	return writeRows(filepath.Join(dir, name+".csv"), rows)
}
