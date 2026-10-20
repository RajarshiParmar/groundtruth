package classify

import (
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
