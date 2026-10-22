package paths

import "testing"

func TestExcluder(t *testing.T) {
	e, err := NewExcluder([]string{"package-lock.json", "*.min.js", "vendor/**", "docs/gen/*.md"})
	if err != nil {
		t.Fatalf("NewExcluder: %v", err)
	}
	cases := map[string]bool{
		"package-lock.json":     true,
		"web/package-lock.json": true,
		"app/x.min.js":          true,
		"vendor/a/b.go":         true,
		// A glob with "/" is anchored at the repo root
		"src/vendor/a.go":   false,
		"docs/gen/a.md":     true,
		"docs/gen/sub/a.md": false,
		"main.go":           false,
	}
	for path, want := range cases {
		if got := e.Match(path); got != want {
			t.Errorf("Match(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestExcluder_Nil(t *testing.T) {
	var e *Excluder
	if e.Match("go.sum") {
		t.Error("nil Excluder matched")
	}
	e, err := NewExcluder(nil)
	if err != nil || e.Match("go.sum") {
		t.Errorf("empty Excluder: match or error %v", err)
	}
}

func TestNewExcluder_BadPattern(t *testing.T) {
	_, err := NewExcluder([]string{"go.sum", "a/[b"})
	if err == nil || err.Error() != `exclude_paths: invalid pattern "a/[b"` {
		t.Errorf("error = %v", err)
	}
}
