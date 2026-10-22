package classify

import (
	"reflect"
	"testing"
)

func TestTicketMatcher(t *testing.T) {
	m := NewTicketMatcher([]string{"PROJ", "OPS"})
	cases := []struct {
		msg  string
		want []string
	}{
		{"fix: thing #12", []string{"#12"}},
		{"fix: thing (#12)", []string{"#12"}},
		{"fixes #12, #13", []string{"#12", "#13"}},
		{"#7 at the start", []string{"#7"}},
		{"see page#3", nil},
		{"quote &#39; here", nil},
		{"feat: PROJ-12 and OPS-7", []string{"PROJ-12", "OPS-7"}},
		{"XPROJ-12 PROJ-12a proj-12", nil},
		{"PROJ-12\n\nRefs: PROJ-12, #4", []string{"PROJ-12", "#4"}},
		{"#4 then PROJ-2 then #4", []string{"#4", "PROJ-2"}},
	}
	for _, tc := range cases {
		if got := m.Find(tc.msg); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Find(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestTicketMatcher_NoPrefixes(t *testing.T) {
	if got := NewTicketMatcher(nil).Find("PROJ-12 #5"); !reflect.DeepEqual(got, []string{"#5"}) {
		t.Errorf("Find = %v, want [#5]", got)
	}
	var nilMatcher *TicketMatcher
	if got := nilMatcher.Find("#5"); got != nil {
		t.Errorf("nil matcher Find = %v, want nil", got)
	}
}
