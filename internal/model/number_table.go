package model

// NumberTable is labelled rows of integers with named columns: the per-repo
// breakdown and the comparison between two periods
type NumberTable struct {
	Caption   string   // optional line shown above the table by text writers
	KeyHeader string   // first column's header: "repo" or "metric"
	Columns   []string // value columns
	Rows      []NumberRow
}

type NumberRow struct {
	Label  string
	Values []int // one per column
}
