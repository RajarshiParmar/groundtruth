package metrics

import (
	"maps"
	"slices"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// Compare puts the totals of two periods side by side. Counts become one row
// per key ("commit_by_type.feat"), except tickets, whose total is how many
// distinct tickets were mentioned; lists, series and the contribution score
// are not totals and are left out. A key missing from one period counts as 0.
func Compare(current, previous []Result, caption string) model.NumberTable {
	t := model.NumberTable{
		Caption:   caption,
		KeyHeader: "metric",
		Columns:   []string{"current", "previous", "change"},
	}
	row := func(label string, cur, prev int) {
		t.Rows = append(t.Rows, model.NumberRow{Label: label, Values: []int{cur, prev, cur - prev}})
	}

	before := make(map[string]any, len(previous))
	for _, r := range previous {
		before[r.Name] = r.Value
	}

	for _, r := range current {
		switch v := r.Value.(type) {
		case int:
			prev, _ := before[r.Name].(int)
			row(r.Name, v, prev)

		case map[string]int:
			prev, _ := before[r.Name].(map[string]int)
			// A row per ticket would be a list, not a total
			if r.Name == "tickets" {
				row(r.Name, len(v), len(prev))
				continue
			}
			keys := maps.Clone(v)
			maps.Copy(keys, prev)
			for _, k := range slices.Sorted(maps.Keys(keys)) {
				row(r.Name+"."+k, v[k], prev[k])
			}
		}
	}
	return t
}
