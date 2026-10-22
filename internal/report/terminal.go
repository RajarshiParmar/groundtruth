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

	// wrapWidth is the widest a summary line may get before key-value lists wrap
	wrapWidth = 80
	// valueIndent lines continuation rows up with the value column ("  " + 20-wide label + " ")
	valueIndent = 23
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
		printKeyValue(formatLabel(r.Name), v)

	case map[string]map[string]int:
		periods := make([]string, 0, len(v))
		for p := range v {
			periods = append(periods, p)
		}
		sort.Strings(periods)
		for _, period := range periods {
			printKeyValue(fmt.Sprintf("%s (%s)", formatLabel(r.Name), period), v[period])
		}

	case []model.MergeSummary:
		fmt.Printf("  %-20s %d merges\n", formatLabel(r.Name), len(v))

	case model.ContributionScore:
		printContributionScore(v)

	case model.Highlights:
		printHighlights(v.Summary())

	case model.NumberTable:
		fmt.Printf("  %s\n", formatLabel(r.Name))
		for _, l := range numberTableLines(v) {
			fmt.Printf("    %s\n", l)
		}

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

func keyValueParts(data map[string]int) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %d", k, data[k]))
	}
	return parts
}

// printKeyValue prints "label  k: v  k: v", wrapping so no line is wider than wrapWidth
func printKeyValue(label string, data map[string]int) {
	prefix := fmt.Sprintf("  %-20s ", label)
	lines := wrapParts(keyValueParts(data), wrapWidth-len([]rune(prefix)))
	if len(lines) == 0 {
		lines = []string{""}
	}

	fmt.Printf("%s%s\n", prefix, lines[0])
	for _, l := range lines[1:] {
		fmt.Printf("%s%s\n", strings.Repeat(" ", valueIndent), l)
	}
}

// wrapParts joins parts with two spaces into lines at most width runes wide.
// A part wider than width on its own gets a line to itself.
func wrapParts(parts []string, width int) []string {
	var lines []string
	var cur string
	for _, p := range parts {
		switch {
		case cur == "":
			cur = p
		case len([]rune(cur))+2+len([]rune(p)) <= width:
			cur += "  " + p
		default:
			lines = append(lines, cur)
			cur = p
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// printHighlights renders the short view: per scope its counts and newest
// subjects, then every breaking change.
func printHighlights(h model.Highlights) {
	status, hint, empty := highlightsStatus(h)
	fmt.Printf("  %-20s %s\n", "Highlights", status)
	if empty {
		return
	}
	if hint != "" {
		fmt.Printf("    %s\n", hint)
	}

	for _, g := range h.Groups {
		fmt.Printf("    %-18s %s\n", g.Scope, strings.Join(keyValueParts(g.Counts), "  "))
		for _, it := range g.Items {
			fmt.Printf("      - %s%s%s\n", it.Subject, pairedSuffix(it), ticketSuffix(it.Tickets))
		}
		if more := moreCount(g); more > 0 {
			fmt.Printf("      … %d more\n", more)
		}
	}

	if len(h.Breaking) == 0 {
		return
	}
	fmt.Println("  Breaking changes")
	for _, it := range h.Breaking {
		fmt.Printf("      - %s\n", breakingLabel(it))
	}
}
