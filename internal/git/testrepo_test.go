package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	testEmail = "me@example.com"
	mainRef   = plumbing.ReferenceName("refs/heads/main")
)

// deleted marks a file to remove in a testCommit
const deleted = "\x00deleted"

// base is the time of day(0); commits are spaced a day apart
var base = time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)

func day(n int) time.Time { return base.AddDate(0, 0, n) }

// testRepo builds a real repository in a temp dir, so scanner tests exercise
// go-git exactly as a run does.
type testRepo struct {
	t    *testing.T
	dir  string
	repo *git.Repository
	wt   *git.Worktree
}

type testCommit struct {
	msg   string
	email string // defaults to testEmail
	name  string // defaults to "Test"
	when  time.Time
	files map[string]string // path → contents, or deleted
	// parents defaults to HEAD; set two or more for a merge
	parents []plumbing.Hash
}

func newTestRepo(t *testing.T) *testRepo {
	t.Helper()
	return newTestRepoAt(t, t.TempDir())
}

// newTestRepoAt initialises a repository at dir, creating it if needed
func newTestRepoAt(t *testing.T, dir string) *testRepo {
	t.Helper()
	repo, err := git.PlainInitWithOptions(dir, &git.PlainInitOptions{
		InitOptions: git.InitOptions{DefaultBranch: mainRef},
	})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	return &testRepo{t: t, dir: dir, repo: repo, wt: wt}
}

func (r *testRepo) commit(c testCommit) plumbing.Hash {
	r.t.Helper()

	for name, contents := range c.files {
		path := filepath.Join(r.dir, name)
		if contents == deleted {
			if _, err := r.wt.Remove(name); err != nil {
				r.t.Fatalf("remove %s: %v", name, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			r.t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
			r.t.Fatalf("write %s: %v", name, err)
		}
		if _, err := r.wt.Add(name); err != nil {
			r.t.Fatalf("add %s: %v", name, err)
		}
	}

	email := c.email
	if email == "" {
		email = testEmail
	}
	name := c.name
	if name == "" {
		name = "Test"
	}
	sig := &object.Signature{Name: name, Email: email, When: c.when}

	h, err := r.wt.Commit(c.msg, &git.CommitOptions{
		Author:            sig,
		Committer:         sig,
		Parents:           c.parents,
		AllowEmptyCommits: true,
	})
	if err != nil {
		r.t.Fatalf("commit %q: %v", c.msg, err)
	}
	return h
}

// checkout switches to branch, creating it at HEAD when create is set
func (r *testRepo) checkout(branch string, create bool) {
	r.t.Helper()
	err := r.wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
		Create: create,
	})
	if err != nil {
		r.t.Fatalf("checkout %s: %v", branch, err)
	}
}

func (r *testRepo) head() plumbing.Hash {
	r.t.Helper()
	ref, err := r.repo.Head()
	if err != nil {
		r.t.Fatalf("head: %v", err)
	}
	return ref.Hash()
}

// merge records a merge of branch into the current branch
func (r *testRepo) merge(branch string, when time.Time) plumbing.Hash {
	r.t.Helper()
	ref, err := r.repo.Reference(plumbing.NewBranchReferenceName(branch), true)
	if err != nil {
		r.t.Fatalf("ref %s: %v", branch, err)
	}
	return r.commit(testCommit{
		msg:     "Merge branch '" + branch + "'\n\nsecond line",
		when:    when,
		parents: []plumbing.Hash{r.head(), ref.Hash()},
	})
}
