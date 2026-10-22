package config

import (
	"fmt"
	"time"
)

func TimeRangeToBounds(tr TimeRange) (time.Time, time.Time, error) {
	return TimeRangeBoundsAt(tr, time.Now())
}

// TimeRangeBoundsAt is TimeRangeToBounds with a fixed "now", so a caller can
// give PeriodLabel the same instant and the label matches the scanned range
func TimeRangeBoundsAt(tr TimeRange, now time.Time) (time.Time, time.Time, error) {

	if tr.Last != "" {
		years, months, days, ok := lastOffset(tr.Last)
		if !ok {
			return time.Time{}, time.Time{}, fmt.Errorf("time_range.last: unsupported value %q", tr.Last)
		}
		return now.AddDate(-years, -months, -days), now, nil
	}

	// Presence and ordering of from/to are checked in validate
	from, err := time.ParseInLocation("2006-01-02", tr.From, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("time_range.from: invalid date %q (expected YYYY-MM-DD)", tr.From)
	}

	to, err := time.ParseInLocation("2006-01-02", tr.To, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("time_range.to: invalid date %q (expected YYYY-MM-DD)", tr.To)
	}

	// Upper bound is exclusive, so include the whole of the "to" day
	return from, to.AddDate(0, 0, 1), nil
}

// lastOffset is how far back each supported time_range.last value reaches
func lastOffset(last string) (years, months, days int, ok bool) {
	switch last {
	case "30d":
		return 0, 0, 30, true
	case "3m":
		return 0, 3, 0, true
	case "6m":
		return 0, 6, 0, true
	case "1y":
		return 1, 0, 0, true
	}
	return 0, 0, 0, false
}

// PeriodLabel describes the report period for people, e.g. "2025-01-01 to 2025-06-30"
// or "last 6m (2025-04-22 to 2025-10-22)". It expects a validated TimeRange.
func PeriodLabel(tr TimeRange, now time.Time) string {
	if tr.Last == "" {
		// The config dates, so the inclusive "to" day is shown rather than the exclusive bound
		return tr.From + " to " + tr.To
	}

	years, months, days, ok := lastOffset(tr.Last)
	if !ok {
		return "last " + tr.Last
	}
	start := now.AddDate(-years, -months, -days)
	return fmt.Sprintf("last %s (%s to %s)", tr.Last, start.Format("2006-01-02"), now.Format("2006-01-02"))
}
