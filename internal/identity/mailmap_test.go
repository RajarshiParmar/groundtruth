package identity

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
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

// Lines git reads that a stricter parser would drop
func TestParseMailmap_GitCompat(t *testing.T) {
	text := "\ufeff# saved by Notepad\n<bom@new.com> <bom@old.com>\n" +
		"C# Team <c@new.com> <c@old.com>\n" +
		"<d@new.com> <d@old.com> old laptop\n" +
		"Me <me@new.com> me <shared@x.com>\n" +
		"  # indented comment\n"
	m, warnings := ParseMailmap(strings.NewReader(text), "f")
	if len(warnings) != 0 {
		t.Errorf("warnings = %q, want none", warnings)
	}
	for _, tc := range []struct{ name, email, want string }{
		{"", "bom@old.com", "bom@new.com"},   // UTF-8 BOM, as Notepad saves it
		{"", "c@old.com", "c@new.com"},       // "#" inside a name is not a comment
		{"", "d@old.com", "d@new.com"},       // text after the last ">" is ignored
		{"ME", "shared@x.com", "me@new.com"}, // names compare case-insensitively, as in git
		{"Bob", "shared@x.com", "shared@x.com"},
	} {
		if got := m.Canonical(tc.name, tc.email); got != tc.want {
			t.Errorf("Canonical(%q, %q) = %q, want %q", tc.name, tc.email, got, tc.want)
		}
	}
}

func TestParseMailmap_ReadError(t *testing.T) {
	_, warnings := ParseMailmap(iotest.ErrReader(errors.New("boom")), "f")
	if want := []string{"f: boom"}; !reflect.DeepEqual(warnings, want) {
		t.Errorf("warnings = %q, want %q", warnings, want)
	}

	long := "<a@new.com> <a@old.com>\n" + strings.Repeat("x", 70*1024) + "\n"
	m, warnings := ParseMailmap(strings.NewReader(long), "f")
	if len(warnings) != 1 || !strings.HasPrefix(warnings[0], "f: ") || m.Canonical("", "a@old.com") != "a@new.com" {
		t.Errorf("long line: warnings %q, Canonical %q; want one read warning and the first entry kept", warnings, m.Canonical("", "a@old.com"))
	}
}
