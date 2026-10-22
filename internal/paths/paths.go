// Package paths decides which files stay out of the lines and files counts.
package paths

import (
	"fmt"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Excluder matches repo paths against exclude_paths globs. A glob without
// "/" matches the base name at any depth (like .gitignore); a glob with "/"
// matches the whole path from the repo root.
type Excluder struct {
	globs []string
}

func NewExcluder(globs []string) (*Excluder, error) {
	for _, g := range globs {
		if !doublestar.ValidatePattern(g) {
			return nil, fmt.Errorf("exclude_paths: invalid pattern %q", g)
		}
	}
	return &Excluder{globs: globs}, nil
}

// Match reports whether p (slash-separated, relative to the repo root) is excluded
func (e *Excluder) Match(p string) bool {
	if e == nil {
		return false
	}
	for _, g := range e.globs {
		target := p
		if !strings.Contains(g, "/") {
			target = path.Base(p)
		}
		// Patterns were validated in NewExcluder, so Match cannot fail here
		if ok, _ := doublestar.Match(g, target); ok {
			return true
		}
	}
	return false
}
