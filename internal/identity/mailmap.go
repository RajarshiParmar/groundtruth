// Package identity decides which commits belong to the user: .mailmap
// canonical emails, case-insensitive matching and Co-authored-by trailers.
package identity

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Mailmap maps commit emails (and optionally names) to canonical emails.
// Only emails matter for matching, so proper names are not kept.
type Mailmap struct {
	entries map[string]mapping // lowercased commit email
}

type mapping struct {
	any    string            // proper email for any name ("" if none)
	byName map[string]string // commit name → proper email
}

// One or two "Name <email>" pairs; names may be empty
var mailmapLine = regexp.MustCompile(`^([^<>]*)<([^<>]*)>\s*(?:([^<>]*)<([^<>]*)>)?$`)

// ParseMailmap reads git's .mailmap format. Malformed lines are skipped and
// returned as warnings ("<source>:<line>: malformed mailmap entry").
func ParseMailmap(r io.Reader, source string) (Mailmap, []string) {
	m := Mailmap{entries: make(map[string]mapping)}
	var warnings []string

	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		p := mailmapLine.FindStringSubmatch(line)
		if p == nil {
			warnings = append(warnings, fmt.Sprintf("%s:%d: malformed mailmap entry", source, n))
			continue
		}

		if strings.Count(line, "<") == 1 {
			// "Proper Name <proper@x>" only renames, so the email stays as it is
			continue
		}
		m.set(strings.TrimSpace(p[3]), p[4], strings.ToLower(strings.TrimSpace(p[2])))
	}
	return m, warnings
}

func (m *Mailmap) set(name, email, proper string) {
	if proper == "" {
		return
	}
	key := strings.ToLower(strings.TrimSpace(email))
	e := m.entries[key]
	if name == "" {
		e.any = proper
	} else {
		if e.byName == nil {
			e.byName = make(map[string]string)
		}
		e.byName[name] = proper
	}
	m.entries[key] = e
}

// Canonical returns the mapped email, lowercased. A name+email entry beats
// an email-only one; without an entry it is the email lowercased.
func (m Mailmap) Canonical(name, email string) string {
	key := strings.ToLower(email)
	e, ok := m.entries[key]
	if !ok {
		return key
	}
	if p, ok := e.byName[name]; ok {
		return p
	}
	if e.any != "" {
		return e.any
	}
	return key
}

// Merge combines mailmaps; later maps win on conflicts
func Merge(maps ...Mailmap) Mailmap {
	out := Mailmap{entries: make(map[string]mapping)}
	for _, m := range maps {
		for key, e := range m.entries {
			if e.any != "" {
				out.set("", key, e.any)
			}
			for name, p := range e.byName {
				out.set(name, key, p)
			}
		}
	}
	return out
}

// Matcher decides whether a commit identity is one of the user's emails
type Matcher struct {
	mailmap Mailmap
	emails  map[string]bool
}

// NewMatcher accepts each listed email and its canonical form, so listing an
// old email still matches. No emails means every author matches.
func NewMatcher(emails []string, m Mailmap) *Matcher {
	set := make(map[string]bool)
	for _, e := range emails {
		set[strings.ToLower(e)] = true
		set[m.Canonical("", e)] = true
	}
	return &Matcher{mailmap: m, emails: set}
}

func (m *Matcher) Match(name, email string) bool {
	if len(m.emails) == 0 {
		return true
	}
	return m.emails[m.mailmap.Canonical(name, email)] || m.emails[strings.ToLower(email)]
}
