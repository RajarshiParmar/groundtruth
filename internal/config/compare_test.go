package config

import (
	"strings"
	"testing"
	"time"
)

// withRange swaps validBase's last: 30d for explicit dates and appends extra
func withRange(from, to, extra string) string {
	return strings.Replace(validBase, "  last: 30d\n", "  from: "+from+"\n  to: "+to+"\n", 1) + extra
}

func TestLoad_Compare(t *testing.T) {
	cfg, err := Load(writeConfig(t, validBase+"compare: previous\n"))
	if err != nil || !cfg.Compare.Previous || !cfg.Compare.Enabled() {
		t.Errorf("previous: err %v, %+v", err, cfg.Compare)
	}

	cfg, err = Load(writeConfig(t, withRange("2025-01-01", "2025-06-30", "compare:\n  from: 2024-01-01\n  to: 2024-06-30\n")))
	if err != nil || cfg.Compare.From != "2024-01-01" || cfg.Compare.To != "2024-06-30" || cfg.Compare.Previous || !cfg.Compare.Enabled() {
		t.Errorf("explicit: err %v, %+v", err, cfg.Compare)
	}

	cfg, err = Load(writeConfig(t, validBase))
	if err != nil || cfg.Compare.Enabled() {
		t.Errorf("absent: err %v, %+v", err, cfg.Compare)
	}
}

func TestLoad_CompareErrors(t *testing.T) {
	const shape = `compare: use "previous" or from/to dates`
	cases := map[string]struct{ config, want string }{
		"other scalar":  {validBase + "compare: yesterday\n", shape},
		"last":          {validBase + "compare:\n  last: 6m\n", shape},
		"missing to":    {validBase + "compare:\n  from: 2024-01-01\n", shape},
		"list":          {validBase + "compare: [previous]\n", shape},
		"bad from date": {withRange("2025-01-01", "2025-06-30", "compare:\n  from: 2024-13-01\n  to: 2024-12-31\n"), `compare.from: invalid date "2024-13-01" (expected YYYY-MM-DD)`},
		"bad to date":   {withRange("2025-01-01", "2025-06-30", "compare:\n  from: 2024-01-01\n  to: 2024-02-30\n"), `compare.to: invalid date "2024-02-30" (expected YYYY-MM-DD)`},
		"from after to": {withRange("2025-01-01", "2025-06-30", "compare:\n  from: 2024-06-30\n  to: 2024-01-01\n"), "compare: from must be before or equal to to"},
		"overlap":       {withRange("2025-01-01", "2025-06-30", "compare:\n  from: 2024-07-01\n  to: 2025-01-01\n"), "compare: the earlier period must end before time_range starts"},
	}
	for name, tc := range cases {
		_, err := Load(writeConfig(t, tc.config))
		if err == nil || !strings.HasSuffix(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want %q", name, err, tc.want)
		}
	}

	// Ending the day before time_range starts is fine
	if _, err := Load(writeConfig(t, withRange("2025-01-01", "2025-06-30", "compare:\n  from: 2024-07-01\n  to: 2024-12-31\n"))); err != nil {
		t.Errorf("adjacent range: %v", err)
	}
}

func day(s string) time.Time {
	d, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		panic(err)
	}
	return d
}

func TestCompareBounds(t *testing.T) {
	now := time.Date(2025, 7, 15, 12, 0, 0, 0, time.Local)
	previous := Compare{Previous: true}
	cases := []struct {
		name     string
		tr       TimeRange
		c        Compare
		from, to time.Time
	}{
		{"half year", TimeRange{From: "2025-01-01", To: "2025-06-30"}, previous, day("2024-07-01"), day("2025-01-01")},
		{"leap february", TimeRange{From: "2024-02-01", To: "2024-02-29"}, previous, day("2024-01-01"), day("2024-02-01")},
		{"14 days", TimeRange{From: "2025-03-10", To: "2025-03-23"}, previous, day("2025-02-24"), day("2025-03-10")},
		{"last 6m", TimeRange{Last: "6m"}, previous, now.AddDate(0, -12, 0), now.AddDate(0, -6, 0)},
		{"last 30d", TimeRange{Last: "30d"}, previous, now.AddDate(0, 0, -60), now.AddDate(0, 0, -30)},
		{"explicit", TimeRange{From: "2025-01-01", To: "2025-06-30"}, Compare{From: "2024-01-01", To: "2024-06-30"}, day("2024-01-01"), day("2024-07-01")},
	}
	for _, tc := range cases {
		from, to, err := CompareBounds(tc.tr, tc.c, now)
		if err != nil || !from.Equal(tc.from) || !to.Equal(tc.to) {
			t.Errorf("%s: [%v, %v) err %v, want [%v, %v)", tc.name, from, to, err, tc.from, tc.to)
		}
	}
	if got := now.AddDate(0, -6, 0); !got.Equal(time.Date(2025, 1, 15, 12, 0, 0, 0, time.Local)) {
		t.Errorf("last 6m sanity: %v", got)
	}
}

func TestBoundsLabel(t *testing.T) {
	if got := BoundsLabel(day("2024-07-01"), day("2025-01-01")); got != "2024-07-01 to 2024-12-31" {
		t.Errorf("BoundsLabel = %q", got)
	}
}

func TestCompareBounds_BadTimeRange(t *testing.T) {
	now := time.Now()
	for _, tr := range []TimeRange{{Last: "2w"}, {From: "2025-01-xx", To: "2025-02-01"}} {
		if _, _, err := CompareBounds(tr, Compare{Previous: true}, now); err == nil {
			t.Errorf("%+v: want an error", tr)
		}
	}
	if _, _, err := CompareBounds(TimeRange{}, Compare{From: "bad", To: "2024-01-01"}, now); err == nil || err.Error() != `compare.from: invalid date "bad" (expected YYYY-MM-DD)` {
		t.Errorf("bad explicit from: %v", err)
	}
}
