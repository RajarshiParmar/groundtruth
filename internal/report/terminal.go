package report

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

const (
	lineWidth = 42
	titleText = "groundtruth summary"
)

// WriteSummary prints a concise summary of all computed metrics to stdout.
func WriteSummary(results []metrics.Result) {
	border := strings.Repeat("\u2500", lineWidth)

	// Compose a header line of exactly lineWidth runes:
	// "── <title> " followed by enough box-drawing characters to fill.
	prefix := "\u2500\u2500 " + titleText + " "
	prefixRunes := len([]rune(prefix))
	fillCount := lineWidth - prefixRunes
	if fillCount < 0 {
		fillCount = 0
	}
	fmt.Printf("%s%s\n", prefix, strings.Repeat("\u2500", fillCount))

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
		periods := make([]string, 0, len(v))
		for p := range v {
			periods = append(periods, p)
		}
		sort.Strings(periods)
		for _, period := range periods {
			label := fmt.Sprintf("%s (%s)", formatLabel(r.Name), period)
			parts := formatKeyValueInline(v[period])
			fmt.Printf("  %-20s %s\n", label, parts)
		}

	case []model.MergeSummary:
		fmt.Printf("  %-20s %d merges\n", formatLabel(r.Name), len(v))

	case model.ContributionScore:
		printContributionScore(v)

	default:
		fmt.Printf("  %-20s (unknown format)\n", formatLabel(r.Name))
	}
}

func printContributionScore(data model.ContributionScore) {
	fmt.Printf("  %-20s %.1f  (%d commits)\n", "Contribution", data.Total, data.CommitCount)

	if len(data.Breakdown) == 0 {
		return
	}

	keys := make([]string, 0, len(data.Breakdown))
	for k := range data.Breakdown {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %.0f", k, data.Breakdown[k]))
	}
	fmt.Printf("  %-20s %s\n", "", strings.Join(parts, "  "))
}

// formatLabel converts a snake_case metric name to a Title Case label.
// Uses a manual word walker instead of the deprecated strings.Title.
func formatLabel(name string) string {
	words := strings.Split(name, "_")
	for i, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func formatKeyValueInline(data map[string]int) string {
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
