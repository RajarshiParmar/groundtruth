package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestLoad_HighlightsAndTickets(t *testing.T) {
	cfg, err := Load(writeConfig(t, validBase+`
highlights:
  types: [feat, fix, unknown]
  per_scope: 5
tickets:
  prefixes: [PROJ, OPS2]
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := []string{"feat", "fix", "unknown"}; !reflect.DeepEqual(cfg.Highlights.Types, want) {
		t.Errorf("Types = %v, want %v", cfg.Highlights.Types, want)
	}
	if cfg.Highlights.PerScope != 5 {
		t.Errorf("PerScope = %d, want 5", cfg.Highlights.PerScope)
	}
	if want := []string{"PROJ", "OPS2"}; !reflect.DeepEqual(cfg.Tickets.Prefixes, want) {
		t.Errorf("Prefixes = %v, want %v", cfg.Tickets.Prefixes, want)
	}
}

func TestLoad_HighlightsValidation(t *testing.T) {
	cases := []struct{ yaml, wantErr string }{
		{"highlights:\n  per_scope: -1", "highlights.per_scope must be non-negative"},
		{"highlights:\n  types: [Feat]", `highlights.types: "Feat" must be a lowercase word`},
		{"highlights:\n  types: [feat-x]", `highlights.types: "feat-x" must be a lowercase word`},
		{"tickets:\n  prefixes: [proj]", `tickets.prefixes: "proj" must match ^[A-Z][A-Z0-9]*$ (e.g. PROJ)`},
		{"tickets:\n  prefixes: [PROJ-]", `tickets.prefixes: "PROJ-" must match`},
		{"tickets:\n  prefixes: [1AB]", `tickets.prefixes: "1AB" must match`},
	}
	for _, tc := range cases {
		_, err := Load(writeConfig(t, validBase+"\n"+tc.yaml+"\n"))
		if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%q: error %v, want %q", tc.yaml, err, tc.wantErr)
		}
	}
}

// Only one of the new metrics enabled satisfies "at least one metric"
func TestLoad_NewMetricsCountAsEnabled(t *testing.T) {
	const metricsBlock = "  commit_count: true\n  contribution_score: true\n"
	if !strings.Contains(validBase, metricsBlock) {
		t.Fatal("validBase metrics block changed")
	}
	for _, key := range []string{"commit_by_scope", "highlights", "tickets"} {
		cfgYAML := strings.Replace(validBase, metricsBlock, "  "+key+": true\n", 1)
		if _, err := Load(writeConfig(t, cfgYAML)); err != nil {
			t.Errorf("only %s enabled: %v", key, err)
		}
	}
}
