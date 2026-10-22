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
