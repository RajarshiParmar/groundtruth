package identity

import (
	"reflect"
	"testing"
)

func TestCoAuthors(t *testing.T) {
	two := []Person{{"Ann", "ann@x.com"}, {"Bob", "BOB@x.com"}}
	body := "feat: x\n\nbody\n\nCo-authored-by: Ann <ann@x.com>\nCo-Authored-By: Bob <BOB@x.com>"
	cases := []struct {
		name string
		msg  string
		want []Person
	}{
		{"two trailers", body + "\n", two},
		{"trailing newlines", body + "\n\n  \n", two},
		{"crlf", "feat: x\r\n\r\nCo-authored-by: Ann <ann@x.com>\r\n", []Person{{"Ann", "ann@x.com"}}},
		{"quoted in body only", "feat: x\n\nCo-authored-by: Ann <ann@x.com>\n\nmore text\n", nil},
		{"subject only", "Co-authored-by: Ann <ann@x.com>", nil},
		{"no email", "feat: x\n\nCo-authored-by: Ann\n", nil},
	}
	for _, tc := range cases {
		if got := CoAuthors(tc.msg); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: CoAuthors = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// Forms git's trailer parser accepts
func TestCoAuthors_GitForms(t *testing.T) {
	want := []Person{{"Ann", "ann@x.com"}}
	for name, msg := range map[string]string{
		"whitespace-only separator": "feat: x\n \t\nCo-authored-by: Ann <ann@x.com>",
		"space before colon":        "feat: x\n\nCo-authored-by : Ann <ann@x.com>",
	} {
		if got := CoAuthors(msg); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
	if got := CoAuthors("feat: x\n \n"); got != nil {
		t.Errorf("subject and blank line only: got %v, want nil", got)
	}
}
