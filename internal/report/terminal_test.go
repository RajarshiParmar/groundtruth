package report

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/metrics"
	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestWrapParts(t *testing.T) {
	cases := []struct {
		parts []string
		width int
		want  []string
	}{
		{[]string{"aa", "bb", "cc"}, 6, []string{"aa  bb", "cc"}},
		{[]string{"toolongpart", "x"}, 4, []string{"toolongpart", "x"}},
		{[]string{"é1", "é2"}, 6, []string{"é1  é2"}}, // width counts runes, not bytes
		{nil, 10, nil},
	}
	for _, tc := range cases {
		if got := wrapParts(tc.parts, tc.width); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("wrapParts(%q, %d) = %q, want %q", tc.parts, tc.width, got, tc.want)
		}
	}
}

func TestWriteSummary_WrapsLongMaps(t *testing.T) {
	tickets := make(map[string]int)
	for i := 100; i < 130; i++ {
		tickets[fmt.Sprintf("#%d", i)] = 1
	}
	out := captureStdout(t, func() { WriteSummary([]metrics.Result{{Name: "tickets", Value: tickets}}) })

	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	body := lines[1 : len(lines)-1] // between the borders
	if len(body) < 2 {
		t.Fatalf("30 tickets fit on one line; want wrapping:\n%s", out)
	}
	for _, l := range body {
		if n := len([]rune(l)); n > wrapWidth {
			t.Errorf("line is %d runes, want <= %d: %q", n, wrapWidth, l)
		}
	}
	for _, l := range body[1:] {
		if !strings.HasPrefix(l, strings.Repeat(" ", 23)+"#") {
			t.Errorf("continuation line not aligned to the value column: %q", l)
		}
	}
	for k := range tickets {
		if !strings.Contains(out, k+": 1") {
			t.Errorf("missing %s", k)
		}
	}
}

func TestWriteSummary_Highlights(t *testing.T) {
	out := captureStdout(t, func() {
		WriteSummary([]metrics.Result{{Name: "highlights", Value: sampleHighlights()}})
	})

	for _, want := range []string{
		"  Highlights           3 of 3 commits classified (100%)\n",
		"    auth               feat: 2\n",
		"      - say \"hi\", then go [PROJ-1 #4]\n",
		"      … 1 more\n",
		"    api                refactor: 1\n",
		"      - drop v1\n",
		"  Breaking changes\n",
		"      - api: drop v1 (2025-03-05)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "line one") {
		t.Errorf("summary shows an item the per_scope limit should cut:\n%s", out)
	}
	if strings.Contains(out, "Add \"unknown\"") {
		t.Errorf("full coverage should not show the opt-in hint:\n%s", out)
	}
}

func TestWriteSummary_HighlightHints(t *testing.T) {
	oneGroup := []model.HighlightGroup{{Scope: "general", Counts: map[string]int{"feat": 1},
		Items: []model.Highlight{{Subject: "x"}}}}

	cases := []struct {
		name    string
		h       model.Highlights
		want    []string
		notWant string
	}{
		{"no commits", model.Highlights{},
			[]string{"  Highlights           no commits in range\n"}, ""},
		{"nothing classified", model.Highlights{Total: 4},
			[]string{`  Highlights           none: 0 of 4 commits use Conventional Commits. Add "unknown" to highlights.types to include them.`}, ""},
		{"classified but none selected", model.Highlights{Total: 4, Classified: 3},
			[]string{"  Highlights           none: no commits match highlights.types (3 of 4 commits classified)\n"}, `Add "unknown"`},
		{"low coverage", model.Highlights{Total: 10, Classified: 3, Groups: oneGroup},
			[]string{"3 of 10 commits classified (30%)", `Add "unknown" to highlights.types`}, ""},
		{"enough coverage", model.Highlights{Total: 10, Classified: 6, Groups: oneGroup},
			[]string{"6 of 10 commits classified (60%)"}, `Add "unknown"`},
	}
	for _, tc := range cases {
		out := captureStdout(t, func() { WriteSummary([]metrics.Result{{Name: "highlights", Value: tc.h}}) })
		for _, w := range tc.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s: missing %q:\n%s", tc.name, w, out)
			}
		}
		if tc.notWant != "" && strings.Contains(out, tc.notWant) {
			t.Errorf("%s: unexpected %q:\n%s", tc.name, tc.notWant, out)
		}
	}
}
