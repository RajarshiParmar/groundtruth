package git

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/identity"
	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/RajarshiParmar/groundtruth/internal/paths"
	"github.com/go-git/go-git/v5"
)

func excluder(t *testing.T, globs ...string) *paths.Excluder {
	t.Helper()
	e, err := paths.NewExcluder(globs)
	if err != nil {
		t.Fatalf("NewExcluder: %v", err)
	}
	return e
}

func mailmap(text string) identity.Mailmap {
	m, _ := identity.ParseMailmap(strings.NewReader(text), "test")
	return m
}

func subjects(commits []model.Commit) []string {
	out := make([]string, len(commits))
	for i, c := range commits {
		out[i] = c.Message
	}
	return out
}

func TestScan_ExcludesPaths(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "add", when: day(0), files: map[string]string{"main.go": "a\nb\n", "go.sum": "1\n2\n3\n"}})
	r.commit(testCommit{msg: "drop sum", when: day(1), files: map[string]string{"go.sum": deleted}})
	r.commit(testCommit{msg: "vendor", when: day(2), files: map[string]string{"main.go": deleted, "vendor/m.go": "a\nb\n"}})

	got := scan(t, r, ScanOptions{Exclude: excluder(t, "go.sum", "vendor/**")})
	if len(got) != 3 {
		t.Fatalf("got %d commits, want 3", len(got))
	}

	add := got[0]
	if !reflect.DeepEqual(add.FilesChanged, []string{"main.go"}) || add.LinesAdded != 2 ||
		!reflect.DeepEqual(add.ExcludedFiles, []string{"go.sum"}) || add.ExcludedLines != 3 {
		t.Errorf("add: files %q +%d, excluded %q %d; want [main.go] +2, [go.sum] 3",
			add.FilesChanged, add.LinesAdded, add.ExcludedFiles, add.ExcludedLines)
	}

	drop := got[1]
	if len(drop.FilesChanged) != 0 || drop.LinesRemoved != 0 ||
		!reflect.DeepEqual(drop.ExcludedFiles, []string{"go.sum"}) || drop.ExcludedLines != 3 {
		t.Errorf("drop: files %q -%d, excluded %q %d; want [] -0, [go.sum] 3",
			drop.FilesChanged, drop.LinesRemoved, drop.ExcludedFiles, drop.ExcludedLines)
	}

	// DiffTree does no rename detection, so the move is a delete plus an add
	move := got[2]
	if !reflect.DeepEqual(move.FilesChanged, []string{"main.go"}) || move.LinesRemoved != 2 || move.LinesAdded != 0 ||
		!reflect.DeepEqual(move.ExcludedFiles, []string{"vendor/m.go"}) || move.ExcludedLines != 2 {
		t.Errorf("move: files %q +%d -%d, excluded %q %d; want [main.go] +0 -2, [vendor/m.go] 2",
			move.FilesChanged, move.LinesAdded, move.LinesRemoved, move.ExcludedFiles, move.ExcludedLines)
	}
}

func TestScan_NoExcluderKeepsAllFiles(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "add", when: day(0), files: map[string]string{"main.go": "a\n", "go.sum": "1\n"}})

	got := scan(t, r, ScanOptions{})
	if c := got[0]; len(c.FilesChanged) != 2 || c.LinesAdded != 2 || c.ExcludedFiles != nil || c.ExcludedLines != 0 {
		t.Errorf("files %q +%d, excluded %q %d; want both files counted", c.FilesChanged, c.LinesAdded, c.ExcludedFiles, c.ExcludedLines)
	}
}

func TestScan_RepoMailmap(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "mailmap", when: day(0), files: map[string]string{".mailmap": "<me@example.com> <old@laptop.local>\n"}})
	r.commit(testCommit{msg: "old", email: "old@laptop.local", when: day(1)})
	r.commit(testCommit{msg: "theirs", email: "you@x.com", when: day(2)})

	got := subjects(scan(t, r, ScanOptions{Emails: []string{testEmail}}))
	if want := []string{"mailmap", "old"}; !reflect.DeepEqual(got, want) {
		t.Errorf("kept %q, want %q", got, want)
	}
}

func TestScan_EmailCaseInsensitive(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "upper", email: "ME@Example.com", when: day(0)})

	if got := scan(t, r, ScanOptions{Emails: []string{testEmail}}); len(got) != 1 {
		t.Errorf("kept %d commits, want 1", len(got))
	}
}

func TestScan_BareRepoMailmap(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "mailmap", when: day(0), files: map[string]string{".mailmap": "<me@example.com> <old@laptop.local>\n"}})
	r.commit(testCommit{msg: "old", email: "old@laptop.local", when: day(1)})

	bare := t.TempDir()
	if _, err := git.PlainClone(bare, true, &git.CloneOptions{URL: r.dir}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	got := subjects(scan(t, &testRepo{t: t, dir: bare}, ScanOptions{Emails: []string{testEmail}}))
	if want := []string{"mailmap", "old"}; !reflect.DeepEqual(got, want) {
		t.Errorf("kept %q, want %q", got, want)
	}
}

func TestScan_PersonalMailmapWins(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "mailmap", email: "repo@x.com", when: day(0), files: map[string]string{".mailmap": "<repo@x.com> <old@laptop.local>\n"}})
	r.commit(testCommit{msg: "old", email: "old@laptop.local", when: day(1)})

	opts := ScanOptions{Emails: []string{testEmail}, Mailmap: mailmap("<me@example.com> <old@laptop.local>\n")}
	if got, want := subjects(scan(t, r, opts)), []string{"old"}; !reflect.DeepEqual(got, want) {
		t.Errorf("kept %q, want %q", got, want)
	}
}

func TestScan_CoAuthored(t *testing.T) {
	const trailer = "\n\nCo-authored-by: Me <ME@example.com>\n"
	r := newTestRepo(t)
	r.commit(testCommit{msg: "paired" + trailer, email: "pair@x.com", name: "Pair", when: day(0), files: map[string]string{"a": "1\n"}})
	r.commit(testCommit{msg: "mine" + trailer, when: day(1), files: map[string]string{"b": "1\n"}})
	r.commit(testCommit{msg: "theirs", email: "pair@x.com", when: day(2), files: map[string]string{"c": "1\n"}})

	off := scan(t, r, ScanOptions{Emails: []string{testEmail}})
	if got, want := subjects(off), []string{"mine" + trailer}; !reflect.DeepEqual(got, want) || off[0].CoAuthored {
		t.Errorf("co_authored off: kept %q (CoAuthored %v), want %q", got, off[0].CoAuthored, want)
	}

	on := scan(t, r, ScanOptions{Emails: []string{testEmail}, CoAuthored: true})
	if got, want := subjects(on), []string{"paired" + trailer, "mine" + trailer}; !reflect.DeepEqual(got, want) {
		t.Fatalf("co_authored on: kept %q, want %q", got, want)
	}
	if !on[0].CoAuthored || on[0].LinesAdded != 1 {
		t.Errorf("paired: CoAuthored %v, +%d; want true, +1", on[0].CoAuthored, on[0].LinesAdded)
	}
	if on[1].CoAuthored {
		t.Error("mine: CoAuthored = true, want false: you are the author")
	}
}

func TestScan_MalformedRepoMailmapWarns(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "mailmap", when: day(0), files: map[string]string{".mailmap": "junk\n"}})

	s := NewScanner(r.dir)
	if _, err := s.Scan(day(-1), day(100)); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !slices.ContainsFunc(s.Warnings, func(w string) bool { return strings.HasSuffix(w, ".mailmap:1: malformed mailmap entry") }) {
		t.Errorf("warnings = %q, want the malformed .mailmap line", s.Warnings)
	}
}
