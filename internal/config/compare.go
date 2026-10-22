package config

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Compare is the optional earlier period: "previous" (the same length just
// before time_range) or explicit from/to dates. The zero value means off.
type Compare struct {
	Previous bool
	From, To string
}

var errCompareShape = errors.New(`compare: use "previous" or from/to dates`)

func (c *Compare) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if n.Value != "previous" {
			return errCompareShape
		}
		*c = Compare{Previous: true}
		return nil

	case yaml.MappingNode:
		var dates struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		}
		// Content alternates key and value nodes
		for i := 0; i < len(n.Content); i += 2 {
			if k := n.Content[i].Value; k != "from" && k != "to" {
				return errCompareShape
			}
		}
		if err := n.Decode(&dates); err != nil || dates.From == "" || dates.To == "" {
			return errCompareShape
		}
		*c = Compare{From: dates.From, To: dates.To}
		return nil
	}
	return errCompareShape
}

func (c Compare) Enabled() bool {
	return c.Previous || c.From != ""
}

// CompareBounds returns the earlier period as [from, to). "previous" with
// whole calendar months shifts back by months, so January–June compares with
// July–December; otherwise it shifts back by the same number of days, or by
// time_range.last's offset.
func CompareBounds(tr TimeRange, c Compare, now time.Time) (time.Time, time.Time, error) {
	if !c.Previous {
		from, err := parseDay("compare.from", c.From)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to, err := parseDay("compare.to", c.To)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		return from, to.AddDate(0, 0, 1), nil
	}

	if tr.Last != "" {
		years, months, days, ok := lastOffset(tr.Last)
		if !ok {
			return time.Time{}, time.Time{}, fmt.Errorf("time_range.last: unsupported value %q", tr.Last)
		}
		start := now.AddDate(-years, -months, -days)
		return start.AddDate(-years, -months, -days), start, nil
	}

	from, to, err := TimeRangeBoundsAt(tr, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	last := to.AddDate(0, 0, -1) // the inclusive "to" day
	if from.Day() == 1 && to.Day() == 1 {
		months := (last.Year()-from.Year())*12 + int(last.Month()-from.Month()) + 1
		return from.AddDate(0, -months, 0), from, nil
	}
	days := int(to.Sub(from).Hours()/24 + 0.5) // rounded, as a DST change makes a day 23 or 25 hours
	return from.AddDate(0, 0, -days), from, nil
}

// BoundsLabel names a [from, to) period by its first and last day
func BoundsLabel(from, to time.Time) string {
	return from.Format("2006-01-02") + " to " + to.AddDate(0, 0, -1).Format("2006-01-02")
}

func parseDay(key, value string) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: invalid date %q (expected YYYY-MM-DD)", key, value)
	}
	return d, nil
}
