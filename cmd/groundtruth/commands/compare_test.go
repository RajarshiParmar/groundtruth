package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/go-git/go-git/v5"
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

// makeDatedRepo commits "feat: <date>" by email at noon on each day
func makeDatedRepo(t *testing.T, dir string, days ...string) {
	t.Helper()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	for i, d := range days {
		at, err := time.ParseInLocation("2006-01-02", d, time.Local)
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("f%d.txt", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(d+"\n"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := wt.Add(name); err != nil {
			t.Fatalf("add: %v", err)
		}
		sig := &object.Signature{Name: "T", Email: email, When: at.Add(12 * time.Hour)}
		if _, err := wt.Commit("feat: "+d, &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
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
