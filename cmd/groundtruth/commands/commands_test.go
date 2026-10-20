package commands

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
)

const email = "me@example.com"

var when = time.Date(2025, 3, 10, 12, 0, 0, 0, time.Local)

// makeRepo creates a repository with two commits on a feature branch merged
// into main, plus one commit by another author.
func makeRepo(t *testing.T, dir string) {
	t.Helper()
	repo, err := git.PlainInitWithOptions(dir, &git.PlainInitOptions{
		InitOptions: git.InitOptions{DefaultBranch: plumbing.NewBranchReferenceName("main")},
	})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	n := 0
	commit := func(msg, author string, parents ...plumbing.Hash) plumbing.Hash {
		t.Helper()
		n++
		name := fmt.Sprintf("f%d.txt", n)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("line\n"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := wt.Add(name); err != nil {
			t.Fatalf("add: %v", err)
		}
		sig := &object.Signature{Name: "T", Email: author, When: when.Add(time.Duration(n) * time.Hour)}
		h, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig, Parents: parents})
		if err != nil {
			t.Fatalf("commit: %v", err)
		}
		return h
	}
	checkout := func(branch string, create bool) {
		t.Helper()
		if err := wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Create: create}); err != nil {
			t.Fatalf("checkout: %v", err)
		}
	}

	commit("chore: init", email)
	checkout("feature", true)
	commit("feat: one", email)
	side := commit("fix: two", email)
	checkout("main", false)
	commit("docs: other author", "other@example.com")
	head, _ := repo.Head()
	commit("Merge branch 'feature'", email, head.Hash(), side)
}

func writeConfig(t *testing.T, repoPath string, baseDir bool, mode, metrics string) string {
	t.Helper()
	cfg := fmt.Sprintf(`version: 1
repository:
  path: %q
  base_dir: %v
identity:
  authors:
    - email: %s
time_range:
  from: 2025-03-10
  to: 2025-03-10
merge_commits:
  mode: %s
metrics:
%s
output:
  formats: [csv]
  directory: ./unused
`, repoPath, baseDir, email, mode, metrics)

	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(cfg), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func execute(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd.Execute()
}

func csvValue(t *testing.T, path string, key string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, r := range rows[1:] {
		if r[0] == key {
			return r[1]
		}
	}
	return ""
}

func TestRun_AnalyzeMerges(t *testing.T) {
	repo := t.TempDir()
	makeRepo(t, repo)
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, repo, false, "analyze", "  commit_count: true\n  commit_by_type: true\n  merge_summaries: true")
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}

	// init, one, two and the merge; the other author's commit is excluded
	if got := csvValue(t, filepath.Join(out, "commit_count.csv"), "commit_count"); got != "4" {
		t.Errorf("commit_count = %s, want 4", got)
	}

	rows, err := os.ReadFile(filepath.Join(out, "merge_summaries.csv"))
	if err != nil {
		t.Fatalf("merge_summaries.csv: %v", err)
	}
	if !strings.Contains(string(rows), "Merge branch 'feature',2,map[feat:1 fix:1]") {
		t.Errorf("merge_summaries.csv does not attribute one and two to the merge:\n%s", rows)
	}
}

func TestRun_IgnoreMerges(t *testing.T) {
	repo := t.TempDir()
	makeRepo(t, repo)
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, repo, false, "ignore", "  commit_count: true")
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := csvValue(t, filepath.Join(out, "commit_count.csv"), "commit_count"); got != "3" {
		t.Errorf("commit_count = %s, want 3 (merge excluded)", got)
	}
}

func TestRun_BaseDirDedupAndDryRun(t *testing.T) {
	base := t.TempDir()
	makeRepo(t, filepath.Join(base, "a"))
	makeRepo(t, filepath.Join(base, "b"))
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, base, true, "include", "  commit_count: true")
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out, "--dry-run"); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("dry run created %s", out)
	}

	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := csvValue(t, filepath.Join(out, "commit_count.csv"), "commit_count"); got != "4" {
		t.Errorf("commit_count = %s, want 4: identical histories (e.g. a backup clone) count once", got)
	}
}

func TestValidate(t *testing.T) {
	repo := t.TempDir()
	makeRepo(t, repo)

	if err := execute(t, ValidateCmd(), "--config", writeConfig(t, repo, false, "ignore", "  commit_count: true")); err != nil {
		t.Errorf("valid config: %v", err)
	}

	err := execute(t, ValidateCmd(), "--config", writeConfig(t, t.TempDir(), false, "ignore", "  commit_count: true"))
	if err == nil || !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("plain directory: error %v, want not a git repository", err)
	}

	err = execute(t, ValidateCmd(), "--config", writeConfig(t, repo, false, "ignore", "  merge_summaries: true"))
	if err == nil || !strings.Contains(err.Error(), "analyze") {
		t.Errorf("merge_summaries without analyze: error %v", err)
	}
}
