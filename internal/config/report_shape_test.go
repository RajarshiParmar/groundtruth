package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

const sectionsWarning = "report.sections is no longer used: metrics.* decides what appears in every report. Remove the report: block."

func TestLoad_MarkdownFormat(t *testing.T) {
	cfgYAML := strings.Replace(validBase, "formats: [csv]", "formats: [csv, md]", 1)
	if _, err := Load(writeConfig(t, cfgYAML)); err != nil {
		t.Errorf("md format: %v", err)
	}
}

func TestLoad_ReportSectionsDeprecated(t *testing.T) {
	// Every flag false must still load and only warn, never hide content
	cfg, err := Load(writeConfig(t, validBase+"report:\n  sections:\n    summary: false\n    metrics: false\n    timeline: false\n    merges: false\n"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(cfg.Warnings, []string{sectionsWarning}) {
		t.Errorf("Warnings = %q, want the report.sections warning", cfg.Warnings)
	}
	if !cfg.Metrics.CommitCount {
		t.Error("report.sections must not turn metrics off")
	}

	if _, err := Load(writeConfig(t, validBase+"report:\n  sections:\n    bogus: true\n")); err == nil || !strings.Contains(err.Error(), "bogus") {
		t.Errorf("typo under report.sections: error %v, want unknown field", err)
	}

	cfg, err = Load(writeConfig(t, validBase))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Warnings) != 0 {
		t.Errorf("no report block: Warnings = %q, want none", cfg.Warnings)
	}
}

func TestPeriodLabel(t *testing.T) {
	now := time.Date(2025, 10, 22, 15, 4, 0, 0, time.Local)
	cases := []struct {
		tr   TimeRange
		now  time.Time
		want string
	}{
		{TimeRange{From: "2025-01-01", To: "2025-06-30"}, now, "2025-01-01 to 2025-06-30"},
		{TimeRange{Last: "30d"}, now, "last 30d (2025-09-22 to 2025-10-22)"},
		{TimeRange{Last: "3m"}, now, "last 3m (2025-07-22 to 2025-10-22)"},
		{TimeRange{Last: "6m"}, now, "last 6m (2025-04-22 to 2025-10-22)"},
		{TimeRange{Last: "1y"}, now, "last 1y (2024-10-22 to 2025-10-22)"},
		// AddDate normalises "Feb 31" to Mar 3, exactly as TimeRangeToBounds does
		{TimeRange{Last: "3m"}, time.Date(2025, 5, 31, 9, 0, 0, 0, time.Local), "last 3m (2025-03-03 to 2025-05-31)"},
		// Validation rejects unknown values first; the label still degrades gracefully
		{TimeRange{Last: "2w"}, now, "last 2w"},
	}
	for _, tc := range cases {
		if got := PeriodLabel(tc.tr, tc.now); got != tc.want {
			t.Errorf("PeriodLabel(%+v) = %q, want %q", tc.tr, got, tc.want)
		}
	}
}

func TestTimeRangeBoundsAt(t *testing.T) {
	// One "now" for the bounds and the label, so they cannot disagree
	now := time.Date(2025, 5, 31, 23, 59, 0, 0, time.Local)
	from, to, err := TimeRangeBoundsAt(TimeRange{Last: "3m"}, now)
	if err != nil {
		t.Fatalf("TimeRangeBoundsAt: %v", err)
	}
	if !to.Equal(now) || from.Format("2006-01-02") != "2025-03-03" {
		t.Errorf("bounds = %v, %v; want 2025-03-03 to %v", from, to, now)
	}
	if want := "last 3m (" + from.Format("2006-01-02") + " to " + to.Format("2006-01-02") + ")"; PeriodLabel(TimeRange{Last: "3m"}, now) != want {
		t.Errorf("label does not match the bounds %q", want)
	}
}
