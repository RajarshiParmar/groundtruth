package classify

import (
	"reflect"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestConventionalBreaking(t *testing.T) {
	cases := []struct {
		message  string
		breaking bool
	}{
		{"feat!: drop v1", true},
		{"feat(api)!: drop v1", true},
		{"fix: handle a!: edge case", false},
		{"feat: two\n\nBREAKING CHANGE: drop v1", true},
		{"feat: two\n\nBREAKING-CHANGE: drop v1", true},
		{"feat: plain", false},
	}

	c := NewConventionalClassifier()
	for _, tc := range cases {
		got := c.Classify(model.Commit{Message: tc.message})
		if got.Breaking != tc.breaking {
			t.Errorf("%q: Breaking = %v, want %v", tc.message, got.Breaking, tc.breaking)
		}
	}
}

func TestConventionalTypeAndScope(t *testing.T) {
	cases := []struct {
		message, typ, scope string
	}{
		{"feat: add login", "feat", ""},
		{"fix(auth): handle expiry", "fix", "auth"},
		{"FEAT(API): shout", "feat", "api"},
		{"  docs: leading space\n\nbody", "docs", ""},
		{"refactor(core)!: rename", "refactor", "core"},
		{"Merge branch 'feature' into dev", "unknown", ""},
		{"feat:no space", "unknown", ""},
		{"feat(): empty scope", "unknown", ""},
		{"fix-typo: dashes are not a type", "unknown", ""},
		{"", "unknown", ""},
	}

	c := NewConventionalClassifier()
	for _, tc := range cases {
		got := c.Classify(model.Commit{Message: tc.message})
		if got.Type != tc.typ || got.Scope != tc.scope {
			t.Errorf("%q: type %q scope %q, want %q %q", tc.message, got.Type, got.Scope, tc.typ, tc.scope)
		}
	}
}

func TestConventionalKeepsCommit(t *testing.T) {
	in := model.Commit{Hash: "abc", Message: "feat: x", LinesAdded: 5}
	if got := NewConventionalClassifier().Classify(in); !reflect.DeepEqual(got.Commit, in) {
		t.Errorf("Commit = %+v, want %+v", got.Commit, in)
	}
}

func TestConventionalSubject(t *testing.T) {
	cases := map[string]string{
		"feat: add login":             "add login",
		"fix(auth)!: drop v1\n\nbody": "drop v1",
		"feat: crlf\r\n\r\nbody":      "crlf",
		"  docs: padded  ":            "padded",
		"Merge branch 'x' into dev":   "Merge branch 'x' into dev",
		"":                            "",
	}

	c := NewConventionalClassifier()
	for msg, want := range cases {
		if got := c.Classify(model.Commit{Message: msg}).Subject; got != want {
			t.Errorf("%q: Subject = %q, want %q", msg, got, want)
		}
	}
}
