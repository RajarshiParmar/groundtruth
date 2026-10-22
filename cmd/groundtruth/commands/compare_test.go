package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestSplitByTime(t *testing.T) {
	at := time.Date(2025, 3, 10, 0, 0, 0, 0, time.Local)
	commits := []model.Commit{{Hash: "before", Timestamp: at.Add(-time.Second)}, {Hash: "at", Timestamp: at}, {Hash: "after", Timestamp: at.Add(time.Second)}}

	earlier, current := splitByTime(commits, at)
	if len(earlier) != 1 || earlier[0].Hash != "before" || len(current) != 2 || current[0].Hash != "at" {
		t.Errorf("earlier %v, current %v; want [before] and [at after]", earlier, current)
	}
}

// makeDatedRepo commits by email at noon on each day
func makeDatedRepo(t *testing.T, dir string, days ...string) {
	t.Helper()
	var times []time.Time
	for _, d := range days {
		at, err := time.ParseInLocation("2006-01-02", d, time.Local)
		if err != nil {
			t.Fatal(err)
		}
		times = append(times, at.Add(12*time.Hour))
	}
	makeTimedRepo(t, dir, times...)
}

// makeTimedRepo commits "feat: <time>" by email at each instant
func makeTimedRepo(t *testing.T, dir string, times ...time.Time) {
	t.Helper()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	for i, at := range times {
		name := fmt.Sprintf("f%d.txt", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(at.String()+"\n"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := wt.Add(name); err != nil {
			t.Fatalf("add: %v", err)
		}
		sig := &object.Signature{Name: "T", Email: email, When: at}
		if _, err := wt.Commit("feat: "+at.Format(time.RFC3339), &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
}

// weekConfig covers 2025-03-10..2025-03-16 with csv and md output
func weekConfig(t *testing.T, path string, baseDir bool, metrics, extra string) string {
	t.Helper()
	cfg := writeConfig(t, path, baseDir, "include", metrics)
	editConfig(t, cfg, func(s string) string {
		s = strings.Replace(s, "to: 2025-03-10", "to: 2025-03-16", 1)
		return strings.Replace(s, "formats: [csv]", "formats: [csv, md]", 1) + extra
	})
	return cfg
}

func runWeek(t *testing.T, cfg string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "report")
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}
	return out
}

func TestRun_Compare(t *testing.T) {
	repo := t.TempDir()
	makeDatedRepo(t, repo, "2025-03-04", "2025-03-05", "2025-03-10", "2025-03-12", "2025-03-16")
	out := runWeek(t, weekConfig(t, repo, false, "  commit_count: true\n  commit_by_type: true", "compare: previous\n"))

	rows := readRows(t, filepath.Join(out, "comparison.csv"))
	want := [][]string{{"metric", "current", "previous", "change"}, {"commit_count", "3", "2", "1"}, {"commit_by_type.feat", "3", "2", "1"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("comparison.csv = %q, want %q", rows, want)
	}
	if got := csvValue(t, filepath.Join(out, "commit_count.csv"), "commit_count"); got != "3" {
		t.Errorf("commit_count = %s, want 3 (current period only)", got)
	}
	md, _ := os.ReadFile(filepath.Join(out, "report.md"))
	if !strings.Contains(string(md), "2025-03-10 to 2025-03-16 compared with 2025-03-03 to 2025-03-09") {
		t.Errorf("report.md lacks the caption:\n%s", md)
	}
}

func TestRun_CompareRepoOnlyEarlier(t *testing.T) {
	base := t.TempDir()
	makeDatedRepo(t, filepath.Join(base, "a"), "2025-03-05", "2025-03-12")
	makeDatedRepo(t, filepath.Join(base, "b"), "2025-03-04", "2025-03-06")
	out := runWeek(t, weekConfig(t, base, true, "  commit_count: true\n  repo_breakdown: true", "compare: previous\n"))

	if _, err := os.Stat(filepath.Join(out, "repo_breakdown.csv")); !os.IsNotExist(err) {
		t.Errorf("repo_breakdown.csv written with one repo active this period (stat err %v)", err)
	}
	rows := readRows(t, filepath.Join(out, "comparison.csv"))
	if want := []string{"commit_count", "1", "3", "-2"}; !reflect.DeepEqual(rows[1], want) {
		t.Errorf("commit_count row = %q, want %q (b counts in the earlier period)", rows[1], want)
	}
}

func TestRun_NoCompareUnchanged(t *testing.T) {
	repo := t.TempDir()
	makeDatedRepo(t, repo, "2025-03-04", "2025-03-05", "2025-03-10", "2025-03-12", "2025-03-16")
	out := runWeek(t, weekConfig(t, repo, false, "  commit_count: true\n  commit_by_type: true", ""))

	if _, err := os.Stat(filepath.Join(out, "comparison.csv")); !os.IsNotExist(err) {
		t.Errorf("comparison.csv written without compare (stat err %v)", err)
	}
	if got := csvValue(t, filepath.Join(out, "commit_count.csv"), "commit_count"); got != "3" {
		t.Errorf("commit_count = %s, want 3", got)
	}
	if md, _ := os.ReadFile(filepath.Join(out, "report.md")); strings.Contains(string(md), "compared with") {
		t.Errorf("report.md mentions a comparison:\n%s", md)
	}
}

// A commit at exactly 00:00 on time_range.from is in the current period
func TestRun_CompareBoundaryCommit(t *testing.T) {
	start := time.Date(2025, 3, 10, 0, 0, 0, 0, time.Local)
	repo := t.TempDir()
	makeTimedRepo(t, repo, start.Add(-time.Second), start)
	out := runWeek(t, weekConfig(t, repo, false, "  commit_count: true", "compare: previous\n"))

	if row := readRows(t, filepath.Join(out, "comparison.csv"))[1]; !reflect.DeepEqual(row, []string{"commit_count", "1", "1", "0"}) {
		t.Errorf("commit_count row = %q, want one commit in each period", row)
	}
}

// makeMergeRepo: init (03-04) and a side commit (03-05) in the earlier
// week; a fix and the merge of the side branch (03-11) in the current week
func makeMergeRepo(t *testing.T, dir string) {
	t.Helper()
	at := func(day int) time.Time { return time.Date(2025, 3, day, 12, 0, 0, 0, time.Local) }
	repo, err := git.PlainInitWithOptions(dir, &git.PlainInitOptions{InitOptions: git.InitOptions{DefaultBranch: plumbing.NewBranchReferenceName("main")}})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	commit := func(msg string, when time.Time, parents ...plumbing.Hash) plumbing.Hash {
		t.Helper()
		name := filepath.Join(dir, fmt.Sprintf("%d.txt", when.Unix()))
		if err := os.WriteFile(name, []byte(msg), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := wt.Add(filepath.Base(name)); err != nil {
			t.Fatal(err)
		}
		sig := &object.Signature{Name: "T", Email: email, When: when}
		h, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig, Parents: parents})
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	checkout := func(branch string, create bool) {
		t.Helper()
		if err := wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Create: create}); err != nil {
			t.Fatal(err)
		}
	}
	commit("chore: init", at(4))
	checkout("side", true)
	side := commit("feat: side work", at(5))
	checkout("main", false)
	main := commit("fix: on main", at(11))
	commit("Merge branch 'side'", at(11).Add(time.Hour), main, side)
}

func TestRun_CompareMergeModes(t *testing.T) {
	cases := map[string]struct{ current, previous string }{
		"ignore":  {"1", "2"}, // the fix; init and side work
		"analyze": {"2", "2"}, // the fix and the merge
	}
	for mode, want := range cases {
		repo := t.TempDir()
		makeMergeRepo(t, repo)
		metrics := "  commit_count: true"
		if mode == "analyze" {
			metrics += "\n  merge_summaries: true"
		}
		run := func(extra string) string {
			cfg := writeConfig(t, repo, false, mode, metrics)
			editConfig(t, cfg, func(s string) string { return strings.Replace(s, "to: 2025-03-10", "to: 2025-03-16", 1) + extra })
			return runWeek(t, cfg)
		}
		with, without := run("compare: previous\n"), run("")

		if row := readRows(t, filepath.Join(with, "comparison.csv"))[1]; row[1] != want.current || row[2] != want.previous {
			t.Errorf("%s: commit_count row = %q, want current %s, previous %s", mode, row, want.current, want.previous)
		}
		// compare must not change what the current period's merge summaries say
		if mode == "analyze" {
			a, _ := os.ReadFile(filepath.Join(with, "merge_summaries.csv"))
			b, _ := os.ReadFile(filepath.Join(without, "merge_summaries.csv"))
			if !bytes.Equal(a, b) {
				t.Errorf("merge_summaries.csv differs with compare:\n%s\nwithout:\n%s", a, b)
			}
		}
	}
}

// captureStdout swaps os.Stdout for a pipe while fn runs
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()
	fn()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestRun_CompareVerbose(t *testing.T) {
	repo := t.TempDir()
	makeDatedRepo(t, repo, "2025-03-04", "2025-03-05", "2025-03-10", "2025-03-12", "2025-03-16")
	cfg := weekConfig(t, repo, false, "  commit_count: true", "compare: previous\n")

	out := captureStdout(t, func() {
		if err := execute(t, RunCmd(), "--config", cfg, "--output", filepath.Join(t.TempDir(), "r"), "--verbose"); err != nil {
			t.Errorf("run: %v", err)
		}
	})
	if !strings.Contains(out, "scanned 3 commits (2 more in the earlier period)\n") {
		t.Errorf("verbose output lacks both periods:\n%s", out)
	}
}
