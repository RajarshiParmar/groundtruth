package commands

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
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
	commit("feat(api): one PROJ-7", email)
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

func TestRun_HighlightsAndTickets(t *testing.T) {
	repo := t.TempDir()
	makeRepo(t, repo)
	out := filepath.Join(t.TempDir(), "report")

	// The metrics block is followed by top-level keys, so tickets: can ride along
	metrics := "  commit_by_scope: true\n  highlights: true\n  tickets: true\ntickets:\n  prefixes: [PROJ]"
	cfg := writeConfig(t, repo, false, "analyze", metrics)
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}

	f, err := os.Open(filepath.Join(out, "highlights.csv"))
	if err != nil {
		t.Fatalf("highlights.csv: %v", err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// chore: init is not a highlight type; the docs commit is by another author
	var got [][]string
	for _, r := range rows[1:] {
		got = append(got, []string{r[1], r[2], r[4], r[5]}) // scope, type, subject, tickets
	}
	want := [][]string{{"api", "feat", "one PROJ-7", "PROJ-7"}, {"general", "fix", "two", ""}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("highlights.csv rows = %q, want %q", got, want)
	}

	if got := csvValue(t, filepath.Join(out, "tickets.csv"), "PROJ-7"); got != "1" {
		t.Errorf("tickets PROJ-7 = %q, want 1", got)
	}
	scopes := filepath.Join(out, "commit_by_scope.csv")
	if api, general := csvValue(t, scopes, "api"), csvValue(t, scopes, "general"); api != "1" || general != "2" {
		t.Errorf("commit_by_scope api=%q general=%q, want 1 and 2 (chore: init and fix: two)", api, general)
	}
}

// editConfig rewrites a config written by writeConfig
func editConfig(t *testing.T, path string, edit func(string) string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := os.WriteFile(path, []byte(edit(string(data))), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// captureStderr swaps os.Stderr for a pipe while fn runs
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()
	w.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(out)
}

func TestRun_Markdown(t *testing.T) {
	repo := t.TempDir()
	makeRepo(t, repo)
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, repo, false, "analyze", "  commit_count: true\n  highlights: true")
	editConfig(t, cfg, func(s string) string { return strings.Replace(s, "formats: [csv]", "formats: [md]", 1) })

	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(out, "report.md"))
	if err != nil {
		t.Fatalf("report.md: %v", err)
	}
	md := string(data)
	if !strings.HasPrefix(md, "# groundtruth report\n\n2025-03-10 to 2025-03-10\n\n## Highlights\n") {
		t.Errorf("report.md does not start with the title, period and highlights:\n%s", md)
	}
	for _, want := range []string{"- one PROJ-7\n", "- two\n", "## Commit Count\n"} {
		if !strings.Contains(md, want) {
			t.Errorf("report.md missing %q:\n%s", want, md)
		}
	}
}

func TestValidate_WarnsAboutReportSections(t *testing.T) {
	repo := t.TempDir()
	makeRepo(t, repo)

	cfg := writeConfig(t, repo, false, "ignore", "  commit_count: true")
	editConfig(t, cfg, func(s string) string { return s + "report:\n  sections:\n    summary: true\n" })

	var err error
	stderr := captureStderr(t, func() { err = execute(t, ValidateCmd(), "--config", cfg) })
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !strings.Contains(stderr, "warning: report.sections is no longer used") {
		t.Errorf("stderr = %q, want the report.sections warning", stderr)
	}
}

// makeAccurateRepo has a .mailmap for an old email, a go.sum change and a
// commit by another author that names email as a co-author
func makeAccurateRepo(t *testing.T, dir string) {
	t.Helper()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	n := 0
	commit := func(msg, author string, files map[string]string) {
		t.Helper()
		n++
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
				t.Fatalf("write: %v", err)
			}
			if _, err := wt.Add(name); err != nil {
				t.Fatalf("add: %v", err)
			}
		}
		sig := &object.Signature{Name: "T", Email: author, When: when.Add(time.Duration(n) * time.Hour)}
		if _, err := wt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}

	commit("chore: init", email, map[string]string{
		".mailmap": "<" + email + "> <old@laptop.local>\n",
		"main.go":  "package main\n",
		"go.sum":   "a\nb\n",
	})
	commit("fix: from the old laptop", "old@laptop.local", map[string]string{"f.txt": "x\n"})
	commit("feat: paired\n\nCo-authored-by: Me <"+email+">\n", "pair@example.com", map[string]string{"g.txt": "y\n"})
	commit("docs: other author", "other@example.com", map[string]string{"h.txt": "z\n"})
}

func TestRun_AccurateNumbers(t *testing.T) {
	repo := t.TempDir()
	makeAccurateRepo(t, repo)
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, repo, false, "ignore", "  commit_count: true\n  lines_changed: true\n  files_changed: true\nexclude_paths: [go.sum]")
	editConfig(t, cfg, func(s string) string {
		return strings.Replace(s, "identity:\n", "identity:\n  co_authored: true\n", 1)
	})
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}

	// init, the old-email commit (via .mailmap) and the paired one; not the docs commit
	if got := csvValue(t, filepath.Join(out, "commit_count.csv"), "commit_count"); got != "3" {
		t.Errorf("commit_count = %s, want 3", got)
	}
	if got := csvValue(t, filepath.Join(out, "co_authored.csv"), "co_authored"); got != "1" {
		t.Errorf("co_authored = %q, want 1", got)
	}
	lines := filepath.Join(out, "lines_changed.csv")
	if excluded, added := csvValue(t, lines, "excluded"), csvValue(t, lines, "added"); excluded != "2" || added != "4" {
		t.Errorf("lines_changed excluded=%q added=%q, want 2 (go.sum) and 4", excluded, added)
	}
	if got := csvValue(t, filepath.Join(out, "files_changed.csv"), "excluded"); got != "1" {
		t.Errorf("files_changed excluded = %q, want 1", got)
	}
}

// A config without the new keys reports exactly what it did before
func TestRun_NoNewKeysUnchanged(t *testing.T) {
	repo := t.TempDir()
	makeAccurateRepo(t, repo)
	out := filepath.Join(t.TempDir(), "report")

	cfg := writeConfig(t, repo, false, "ignore", "  commit_count: true\n  lines_changed: true\n  files_changed: true")
	if err := execute(t, RunCmd(), "--config", cfg, "--output", out); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, name := range []string{"lines_changed.csv", "files_changed.csv"} {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Contains(string(data), "excluded") {
			t.Errorf("%s has an excluded row without exclude_paths:\n%s", name, data)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "co_authored.csv")); !os.IsNotExist(err) {
		t.Errorf("co_authored.csv written without identity.co_authored (stat err %v)", err)
	}
	// The repo .mailmap still applies: init and the old-email commit
	if got := csvValue(t, filepath.Join(out, "commit_count.csv"), "commit_count"); got != "2" {
		t.Errorf("commit_count = %s, want 2", got)
	}
}
