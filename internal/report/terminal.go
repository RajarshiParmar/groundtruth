package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

const lineWidth = 42

// WriteSummary prints a concise summary of all computed metrics to stdout.
func WriteSummary(results []metrics.Result) {
	border := strings.Repeat("\u2500", lineWidth)

	fmt.Printf("\u2500\u2500 groundtruth summary %s\n", border[:lineWidth-21])

	for _, r := range results {
		printMetricSummary(r)
	}

	fmt.Println(border)
}

func printMetricSummary(r metrics.Result) {
	switch v := r.Value.(type) {

	case int:
		fmt.Printf("  %-20s %d\n", formatLabel(r.Name), v)

	case map[string]int:
		parts := formatKeyValueInline(v)
		fmt.Printf("  %-20s %s\n", formatLabel(r.Name), parts)

	case map[string]map[string]int:
		for period, values := range v {
			label := fmt.Sprintf("%s (%s)", formatLabel(r.Name), period)
			parts := formatKeyValueInline(values)
			fmt.Printf("  %-20s %s\n", label, parts)
		}

	case []model.MergeSummary:
		fmt.Printf("  %-20s %d merges\n", formatLabel(r.Name), len(v))

	case map[string]any:
		printContributionScore(v)

	default:
		fmt.Printf("  %-20s (unknown format)\n", formatLabel(r.Name))
	}
}

func printContributionScore(data map[string]any) {
	total, _ := data["total"].(float64)
	commitCount, _ := data["commit_count"].(int)

	fmt.Printf("  %-20s %.1f  (%d commits)\n", "Contribution", total, commitCount)

	if breakdown, ok := data["breakdown"].(map[string]float64); ok {
		// Sort keys for stable output
		keys := make([]string, 0, len(breakdown))
		for k := range breakdown {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %.0f", k, breakdown[k]))
		}
		fmt.Printf("  %-20s %s\n", "", strings.Join(parts, "  "))
	}
}

func formatLabel(name string) string {
	return strings.ReplaceAll(strings.Title(strings.ReplaceAll(name, "_", " ")), " ", " ")
}

func formatKeyValueInline(data map[string]int) string {
	// Sort keys for stable output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %d", k, data[k]))
	}
	return strings.Join(parts, "  ")
}
