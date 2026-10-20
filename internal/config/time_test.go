package config

import (
	"strings"
	"testing"
	"time"
)

func TestTimeRangeToBounds_Explicit(t *testing.T) {
	from, to, err := TimeRangeToBounds(TimeRange{From: "2025-01-01", To: "2025-06-30"})
	if err != nil {
		t.Fatalf("TimeRangeToBounds: %v", err)
	}

	wantFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)
	// to is exclusive, so the whole of 2025-06-30 is included
	wantTo := time.Date(2025, 7, 1, 0, 0, 0, 0, time.Local)
	if !from.Equal(wantFrom) || !to.Equal(wantTo) {
		t.Errorf("bounds = [%v, %v), want [%v, %v)", from, to, wantFrom, wantTo)
	}
}

func TestTimeRangeToBounds_Last(t *testing.T) {
	cases := map[string]time.Time{
		"30d": time.Now().AddDate(0, 0, -30),
		"3m":  time.Now().AddDate(0, -3, 0),
		"6m":  time.Now().AddDate(0, -6, 0),
		"1y":  time.Now().AddDate(-1, 0, 0),
	}
	for last, wantFrom := range cases {
		from, to, err := TimeRangeToBounds(TimeRange{Last: last})
		if err != nil {
			t.Errorf("%s: %v", last, err)
			continue
		}
		if d := from.Sub(wantFrom); d < -time.Minute || d > time.Minute {
			t.Errorf("%s: from = %v, want about %v", last, from, wantFrom)
		}
		if d := time.Since(to); d < 0 || d > time.Minute {
			t.Errorf("%s: to = %v, want about now", last, to)
		}
	}
}

func TestTimeRangeToBounds_Errors(t *testing.T) {
	cases := []struct {
		tr      TimeRange
		wantErr string
	}{
		{TimeRange{Last: "2w"}, "time_range.last"},
		{TimeRange{From: "2025-13-01", To: "2025-12-31"}, "time_range.from"},
		{TimeRange{From: "2025-01-01", To: "31/12/2025"}, "time_range.to"},
	}
	for _, tc := range cases {
		if _, _, err := TimeRangeToBounds(tc.tr); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%+v: error %v, want %q", tc.tr, err, tc.wantErr)
		}
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	cases := []struct {
		name, old, new, wantErr string
	}{
		{"version", "version: 1", "version: 2", "unsupported config version"},
		{"no path", "  path: /tmp/repo\n", "", "repository.path is required"},
		{"both ranges", "  last: 30d", "  last: 30d\n  from: 2025-01-01", "not both"},
		{"no range", "  last: 30d", "", "one of from/to or last"},
		{"from only", "  last: 30d", "  from: 2025-01-01", "both from and to"},
		{"from after to", "  last: 30d", "  from: 2025-02-01\n  to: 2025-01-01", "from must be before"},
		{"unknown last", "last: 30d", "last: 2w", "time_range.last"},
		{"bad mode", "mode: ignore", "mode: squash", "merge_commits.mode"},
		{"no metrics", "  commit_count: true\n  contribution_score: true\n", "  commit_count: false\n", "at least one metric"},
		{"no formats", "formats: [csv]", "formats: []", "at least one format"},
		{"bad format", "formats: [csv]", "formats: [docx]", `unsupported format "docx"`},
		{"unknown key", "version: 1", "version: 1\nextra: true", "extra"},
	}
	for _, tc := range cases {
		cfgYAML := strings.Replace(validBase, tc.old, tc.new, 1)
		if cfgYAML == validBase {
			t.Fatalf("%s: replacement %q not found in base config", tc.name, tc.old)
		}
		_, err := Load(writeConfig(t, cfgYAML))
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%s: error %v, want %q", tc.name, err, tc.wantErr)
		}
	}
}

func TestLoad_SingleDayRange(t *testing.T) {
	cfgYAML := strings.Replace(validBase, "  last: 30d", "  from: 2025-01-01\n  to: 2025-01-01", 1)
	if _, err := Load(writeConfig(t, cfgYAML)); err != nil {
		t.Errorf("from == to should be a valid one-day range: %v", err)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(t.TempDir() + "/missing.yaml"); err == nil {
		t.Error("missing file: want error")
	}
}
