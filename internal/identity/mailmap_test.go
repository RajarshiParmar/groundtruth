package identity

import (
	"reflect"
	"strings"
	"testing"
)

const sampleMailmap = `# team mailmap
Proper <proper@x.com>
<new@x.com> <OLD@x.com>
Me <me@x.com> <laptop@host.local>

Me <me@x.com> Bot Name <ci@x.com>
not a mailmap line
<e1@x.com> <ci@x.com>
Me <me@x.com> <a@x.com> # trailing comment
`

func parse(t *testing.T, text string) Mailmap {
	t.Helper()
	m, warnings := ParseMailmap(strings.NewReader(text), "repo/.mailmap")
	if want := []string{"repo/.mailmap:7: malformed mailmap entry"}; !reflect.DeepEqual(warnings, want) {
		t.Errorf("warnings = %q, want %q", warnings, want)
	}
	return m
}

func TestParseMailmap(t *testing.T) {
	for name, text := range map[string]string{
		"lf":   sampleMailmap,
		"crlf": strings.ReplaceAll(sampleMailmap, "\n", "\r\n"),
	} {
		m := parse(t, text)
		cases := []struct{ name, email, want string }{
			// The name-only form leaves the email alone
			{"A", "proper@x.com", "proper@x.com"},
			{"Any", "old@X.com", "new@x.com"},
			{"", "laptop@host.local", "me@x.com"},
			// A name+email entry applies only when both match, and beats email-only
			{"Bot Name", "ci@x.com", "me@x.com"},
			{"Other", "ci@x.com", "e1@x.com"},
			{"", "a@x.com", "me@x.com"},
			{"A", "Who@X.com", "who@x.com"},
		}
		for _, tc := range cases {
			if got := m.Canonical(tc.name, tc.email); got != tc.want {
				t.Errorf("%s: Canonical(%q, %q) = %q, want %q", name, tc.name, tc.email, got, tc.want)
			}
		}
	}
}

func TestParseMailmap_NameEntryOnly(t *testing.T) {
	m, _ := ParseMailmap(strings.NewReader("Me <me@x.com> Bot Name <ci@x.com>\n"), "m")
	if got := m.Canonical("Other", "ci@x.com"); got != "ci@x.com" {
		t.Errorf("Canonical(Other) = %q, want ci@x.com", got)
	}
}

func TestMerge(t *testing.T) {
	repo, _ := ParseMailmap(strings.NewReader("<repo@x.com> <old@x.com>\n"), "repo")
	personal, _ := ParseMailmap(strings.NewReader("<mine@x.com> <old@x.com>\n"), "personal")
	if got := Merge(repo, personal).Canonical("", "old@x.com"); got != "mine@x.com" {
		t.Errorf("Merge: Canonical = %q, want mine@x.com (later map wins)", got)
	}
	if got := Merge(repo, Mailmap{}).Canonical("", "old@x.com"); got != "repo@x.com" {
		t.Errorf("Merge with empty: Canonical = %q, want repo@x.com", got)
	}
}

func TestMatcher(t *testing.T) {
	if m := NewMatcher([]string{"Me@Work.com"}, Mailmap{}); !m.Match("A", "me@work.com") || m.Match("A", "you@work.com") {
		t.Error("case-insensitive listed email")
	}

	mm, _ := ParseMailmap(strings.NewReader("<me@work.com> <old@home.net>\n"), "m")
	if !NewMatcher([]string{"me@work.com"}, mm).Match("A", "OLD@home.net") {
		t.Error("mailmapped old email should match")
	}
	// Listing the old email still reaches the canonical address
	if !NewMatcher([]string{"old@home.net"}, mm).Match("A", "me@work.com") {
		t.Error("listed old email should match its canonical address")
	}
	if !NewMatcher(nil, mm).Match("Anyone", "x@y.z") {
		t.Error("no emails should match everyone")
	}
}

func TestMailmap_EdgeEntries(t *testing.T) {
	// An empty proper email maps nothing
	m, w := ParseMailmap(strings.NewReader("Me <> <z@x.com>\n"), "m")
	if len(w) != 0 || m.Canonical("", "z@x.com") != "z@x.com" {
		t.Errorf("empty proper email: %q, warnings %q", m.Canonical("", "z@x.com"), w)
	}

	// Name+email entries survive Merge
	named, _ := ParseMailmap(strings.NewReader("Me <me@x.com> Bot <ci@x.com>\n"), "m")
	if got := Merge(Mailmap{}, named).Canonical("Bot", "ci@x.com"); got != "me@x.com" {
		t.Errorf("Merge kept name entry: %q, want me@x.com", got)
	}
}
