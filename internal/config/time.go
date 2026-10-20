package config

import (
	"fmt"
	"time"
)

func TimeRangeToBounds(tr TimeRange) (time.Time, time.Time, error) {
	now := time.Now()

	if tr.Last != "" {
		switch tr.Last {
		case "30d":
			return now.AddDate(0, 0, -30), now, nil
		case "3m":
			return now.AddDate(0, -3, 0), now, nil
		case "6m":
			return now.AddDate(0, -6, 0), now, nil
		case "1y":
			return now.AddDate(-1, 0, 0), now, nil
		default:
			return time.Time{}, time.Time{}, fmt.Errorf("unsupported time_range.last value")
		}
	}

	// Presence and ordering of from/to are checked in validate
	from, err := time.ParseInLocation("2006-01-02", tr.From, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time_range.from: %q (expected YYYY-MM-DD)", tr.From)
	}

	to, err := time.ParseInLocation("2006-01-02", tr.To, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time_range.to: %q (expected YYYY-MM-DD)", tr.To)
	}

	// Upper bound is exclusive, so include the whole of the "to" day
	return from, to.AddDate(0, 0, 1), nil
}
