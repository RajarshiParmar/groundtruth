package report

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// WriteMarkdown writes report.md for pasting into an appraisal doc: the
// period, the full highlights list, then every other metric.
func WriteMarkdown(results []metrics.Result, outputDir, period string) error {
	if err := ensureDir(outputDir); err != nil {
		return err
	}

	// Blocks are separated by one blank line, so spacing is right by construction
	blocks := []string{"# groundtruth report", period}

	// Highlights lead the report wherever the metric sits in the results
	for _, r := range results {
		if h, ok := r.Value.(model.Highlights); ok {
			blocks = append(blocks, highlightBlocks(h)...)
			break
		}
	}

	for _, r := range results {
		if _, ok := r.Value.(model.Highlights); ok {
			continue
		}

		label := formatLabel(r.Name)
		blocks = append(blocks, "## "+label)

		if v, ok := r.Value.(int); ok {
			blocks = append(blocks, fmt.Sprintf("%s: **%d**", label, v))
			continue
		}

		if v, ok := r.Value.(model.NumberTable); ok {
			if v.Caption != "" {
				blocks = append(blocks, mdEscape(v.Caption))
			}
			blocks = append(blocks, markdownTable(mdNumberTable(v)))
			continue
		}

		tables, err := tablesFor(r)
		if err != nil {
			return err
		}
		for _, t := range tables {
			if t.Suffix != "" {
				blocks = append(blocks, "### "+strings.TrimPrefix(t.Suffix, "_"))
			}
			blocks = append(blocks, markdownTable(t))
		}
	}

	doc := strings.Join(blocks, "\n\n") + "\n"
	return writeFile(filepath.Join(outputDir, "report.md"), []byte(doc))
}

// highlightBlocks lists every highlight (not the short view): the reader
// edits the list down rather than looking up what was cut.
func highlightBlocks(h model.Highlights) []string {
	blocks := []string{"## Highlights"}

	status, hint, empty := highlightsStatus(h)
	blocks = append(blocks, status)
	if empty {
		return blocks
	}
	if hint != "" {
		blocks = append(blocks, hint)
	}

	// Scopes go one heading level deeper under a repo heading
	scopeHeading := "### "
	if h.ByRepo {
		scopeHeading = "#### "
	}
	for i, g := range h.Groups {
		if h.ByRepo && (i == 0 || h.Groups[i-1].Repo != g.Repo) {
			blocks = append(blocks, "### "+mdHeading(g.Repo))
		}
		blocks = append(blocks, scopeHeading+mdHeading(g.Scope), strings.Join(keyValueParts(g.Counts), " · "))

		items := make([]string, len(g.Items))
		for i, it := range g.Items {
			items[i] = "- " + mdItem(it.Subject) + pairedSuffix(it) + mdTickets(it.Tickets)
		}
		blocks = append(blocks, strings.Join(items, "\n"))
	}

	if len(h.Breaking) > 0 {
		items := make([]string, len(h.Breaking))
		for i, it := range h.Breaking {
			label := mdItem(it.Subject)
			if it.Scope != "" {
				label = mdItem(it.Scope) + ": " + mdEscape(it.Subject)
			}
			if it.Repo != "" {
				// The repo now leads the item, so it gets the list-marker escaping
				label = mdItem(it.Repo) + " · " + label
			}
			items[i] = "- " + label + pairedSuffix(it) + mdTickets(it.Tickets)
		}
		blocks = append(blocks, "### Breaking changes", strings.Join(items, "\n"))
	}

	return blocks
}

func mdTickets(tickets []string) string {
	if len(tickets) == 0 {
		return ""
	}
	escaped := make([]string, len(tickets))
	for i, t := range tickets {
		escaped[i] = mdEscape(t)
	}
	return " (" + strings.Join(escaped, ", ") + ")"
}

// markdownTable renders a GFM table; a table without rows keeps its header
func markdownTable(t table) string {
	lines := []string{
		"| " + strings.Join(t.Header, " | ") + " |",
		"|" + strings.Repeat("---|", len(t.Header)),
	}
	for _, row := range t.Rows {
		cells := make([]string, len(row))
		for i, v := range row {
			cells[i] = mdCell(v)
		}
		lines = append(lines, "| "+strings.Join(cells, " | ")+" |")
	}
	return strings.Join(lines, "\n")
}

// mdCell formats a table cell for Markdown; text from commits is escaped
func mdCell(v any) string {
	switch x := v.(type) {
	case int:
		return strconv.Itoa(x)
	case float64:
		return fmt.Sprintf("%.1f", x)
	case bool:
		return strconv.FormatBool(x)
	case time.Time:
		return x.Format("2006-01-02")
	case map[string]int:
		return mdEscape(strings.Join(keyValueParts(x), ", "))
	case string:
		return mdEscape(x)
	default:
		return mdEscape(fmt.Sprint(x))
	}
}

var mdEscaper = strings.NewReplacer(
	`\`, `\\`, "`", "\\`", `*`, `\*`, `_`, `\_`, `[`, `\[`, `]`, `\]`,
	`<`, `\<`, `>`, `\>`, `|`, `\|`, `~`, `\~`, `&`, `\&`,
	"\r\n", " ", "\n", " ", "\r", " ",
)

// mdEscape keeps commit text literal: no accidental emphasis, links, HTML or
// table cells, and no line breaks inside a list item or table row
func mdEscape(s string) string {
	return mdEscaper.Replace(s)
}

var orderedMarker = regexp.MustCompile(`^\d{1,9}[.)]`)

// mdItem escapes text that starts a list item, so a subject such as
// "# 2.0", "1. import" or "- fix" stays text instead of a heading or a
// nested list
func mdItem(s string) string {
	s = mdEscape(s)
	if s != "" && strings.ContainsRune("#+-", rune(s[0])) {
		return `\` + s
	}
	if m := orderedMarker.FindStringIndex(s); m != nil {
		return s[:m[1]-1] + `\` + s[m[1]-1:]
	}
	return s
}

// mdHeading escapes heading text; a trailing "#" is escaped so it is never
// read as a closing sequence and dropped
func mdHeading(s string) string {
	s = mdEscape(s)
	if strings.HasSuffix(s, "#") {
		return s[:len(s)-1] + `\#`
	}
	return s
}

// writeFile writes data, reporting write and close errors like writeRows does
func writeFile(path string, data []byte) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// mdNumberTable is the number table with text cells, so a comparison's
// change column keeps its sign
func mdNumberTable(v model.NumberTable) table {
	t := table{Header: append([]string{v.KeyHeader}, v.Columns...)}
	for i, cells := range numberTableCells(v) {
		row := []any{v.Rows[i].Label}
		for _, c := range cells {
			row = append(row, c)
		}
		t.Rows = append(t.Rows, row)
	}
	return t
}
