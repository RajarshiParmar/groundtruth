package classify

import (
	"regexp"
	"sort"
	"strings"
)

// A '#' preceded by a letter, digit, '_' or '&' is a URL anchor or an HTML entity, not a reference
var hashRefPattern = regexp.MustCompile(`(?:^|[^\w&])#(\d+)\b`)

// TicketMatcher finds issue references such as #123 and PROJ-123 in commit messages
type TicketMatcher struct {
	prefixed *regexp.Regexp // nil when no prefixes are configured
}

// NewTicketMatcher matches "#123" plus KEY-123 for each prefix. Prefixes are
// expected to be validated (^[A-Z][A-Z0-9]*$) by the config package.
func NewTicketMatcher(prefixes []string) *TicketMatcher {
	m := &TicketMatcher{}
	if len(prefixes) > 0 {
		quoted := make([]string, len(prefixes))
		for i, p := range prefixes {
			quoted[i] = regexp.QuoteMeta(p)
		}
		m.prefixed = regexp.MustCompile(`\b(?:` + strings.Join(quoted, "|") + `)-\d+\b`)
	}
	return m
}

// Find returns the distinct references in message, in order of first appearance
func (m *TicketMatcher) Find(message string) []string {
	if m == nil {
		return nil
	}

	type ref struct {
		pos  int
		text string
	}
	var refs []ref

	for _, loc := range hashRefPattern.FindAllStringSubmatchIndex(message, -1) {
		// loc[2] is the start of the digits; the '#' sits just before it
		refs = append(refs, ref{loc[2] - 1, "#" + message[loc[2]:loc[3]]})
	}
	if m.prefixed != nil {
		for _, loc := range m.prefixed.FindAllStringIndex(message, -1) {
			refs = append(refs, ref{loc[0], message[loc[0]:loc[1]]})
		}
	}

	sort.Slice(refs, func(i, j int) bool { return refs[i].pos < refs[j].pos })

	var out []string
	seen := make(map[string]bool)
	for _, r := range refs {
		if !seen[r.text] {
			seen[r.text] = true
			out = append(out, r.text)
		}
	}
	return out
}
