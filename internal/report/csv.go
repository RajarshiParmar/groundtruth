package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func WriteCSV(results []metrics.Result, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
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

	default:
		return fmt.Errorf("unsupported metric output for %s", r.Name)
	}
}

func writeMergeSummaries(name string, data []model.MergeSummary, dir string) error {
	path := filepath.Join(dir, name+".csv")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{
		"hash",
		"timestamp",
		"message",
		"commit_count",
		"types",
	})

	for _, m := range data {
		w.Write([]string{
			m.Hash,
			m.Timestamp.Format(time.RFC3339),
			m.Message,
			fmt.Sprintf("%d", m.CommitCount),
			fmt.Sprintf("%v", m.Types),
		})
	}

	return nil
}

func writeSingleValue(name string, value int, dir string) error {
	path := filepath.Join(dir, name+".csv")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"metric", "value"})
	w.Write([]string{name, fmt.Sprintf("%d", value)})

	return nil
}

func writeKeyValue(name string, data map[string]int, dir string) error {
	path := filepath.Join(dir, name+".csv")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"key", "count"})
	for k, v := range data {
		w.Write([]string{k, fmt.Sprintf("%d", v)})
	}

	return nil
}

func writeTimeline(name string, data map[string]map[string]int, dir string) error {
	for period, values := range data {
		path := filepath.Join(dir, fmt.Sprintf("%s_%s.csv", name, period))
		f, err := os.Create(path)
		if err != nil {
			return err
		}

		w := csv.NewWriter(f)
		w.Write([]string{"period", "count"})

		for k, v := range values {
			w.Write([]string{k, fmt.Sprintf("%d", v)})
		}

		w.Flush()
		f.Close()
	}

	return nil
}

func writeList(name string, data []interface{}, dir string) error {
	path := filepath.Join(dir, name+".csv")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"hash", "message", "commit_count", "types"})

	for _, item := range data {
		m := item.(map[string]interface{})
		w.Write([]string{
			m["hash"].(string),
			m["message"].(string),
			fmt.Sprintf("%d", m["commit_count"].(int)),
			fmt.Sprintf("%v", m["types"]),
		})
	}

	return nil
}
