package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// makeOneCommitRepo creates a repository with a single commit by email
func makeOneCommitRepo(t *testing.T, dir, msg string, at time.Time) {
	t.Helper()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ui.txt"), []byte("x\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := wt.Add("ui.txt"); err != nil {
		t.Fatalf("add: %v", err)
	}
	sig := &object.Signature{Name: "T", Email: email, When: at}
	if _, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func readRows(t *testing.T, path string) [][]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var rows [][]string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		rows = append(rows, strings.Split(l, ","))
	}
	return rows
}

const breakdownMetrics = "  commit_count: true\n  highlights: true\n  repo_breakdown: true"

func TestRun_RepoBreakdown(t *testing.T) {
	base := t.TempDir()
	makeRepo(t, filepath.Join(base, "a"))
	makeOneCommitRepo(t, filepath.Join(base, "b"), "feat(ui): button", when.Add(time.Hour))
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, base, true, "include", breakdownMetrics)
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got, want := readRows(t, filepath.Join(out, "repo_breakdown.csv")), [][]string{{"repo", "commits"}, {"a", "4"}, {"b", "1"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("repo_breakdown.csv = %q, want %q", got, want)
	}
	if h := readRows(t, filepath.Join(out, "highlights.csv")); h[0][0] != "repo" {
		t.Errorf("highlights.csv header = %q, want repo first", h[0])
	}
}

func TestRun_OneActiveRepoUnchanged(t *testing.T) {
	base := t.TempDir()
	makeRepo(t, filepath.Join(base, "a"))
	makeOneCommitRepo(t, filepath.Join(base, "b"), "feat(ui): button", when.AddDate(0, 0, -5))
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, base, true, "include", breakdownMetrics)
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(out, "repo_breakdown.csv")); !os.IsNotExist(err) {
		t.Errorf("repo_breakdown.csv written with one active repo (stat err %v)", err)
	}
	if h := readRows(t, filepath.Join(out, "highlights.csv")); h[0][0] != "date" {
		t.Errorf("highlights.csv header = %q, want today's (date first)", h[0])
	}
}

// With one repository, repo_breakdown changes nothing in any file
func TestRun_SingleRepoUnchanged(t *testing.T) {
	repo := t.TempDir()
	makeRepo(t, repo)

	all := "  commit_count: true\n  commit_by_type: true\n  lines_changed: true\n  files_changed: true\n" +
		"  activity_timeline: true\n  merge_summaries: true\n  commit_by_scope: true\n  highlights: true\n  tickets: true"
	run := func(metrics string) string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "report")
		cfg := writeConfig(t, repo, false, "analyze", metrics)
		editConfig(t, cfg, func(s string) string { return strings.Replace(s, "formats: [csv]", "formats: [csv, md]", 1) })
		if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
			t.Fatalf("run: %v", err)
		}
		return out
	}
	without, with := run(all), run(all+"\n  repo_breakdown: true")

	entries, err := os.ReadDir(without)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		a, _ := os.ReadFile(filepath.Join(without, e.Name()))
		b, err := os.ReadFile(filepath.Join(with, e.Name()))
		if err != nil || !bytes.Equal(a, b) {
			t.Errorf("%s differs with repo_breakdown on (err %v)", e.Name(), err)
		}
	}
	if more, _ := os.ReadDir(with); len(more) != len(entries) {
		t.Errorf("repo_breakdown wrote %d files, want %d", len(more), len(entries))
	}
	// Both runs could change the same way, so also pin today's single-repo shape
	if h := readRows(t, filepath.Join(with, "highlights.csv")); h[0][0] != "date" {
		t.Errorf("highlights.csv header = %q, want today's (date first)", h[0])
	}
	md, _ := os.ReadFile(filepath.Join(with, "report.md"))
	if !strings.Contains(string(md), "## Highlights\n\n3 of 3 commits classified (100%)\n\n### api\n") {
		t.Errorf("report.md highlights are not grouped by scope only:\n%s", md)
	}
}
