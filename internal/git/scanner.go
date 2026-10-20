package git

import (
	"fmt"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Scanner struct {
	RepoPath string

	// AnalyzeMerges records, for each merge commit, the commits it brought in
	AnalyzeMerges bool

	// Warnings collects per-commit problems that did not stop the scan
	Warnings []string
}

func NewScanner(path string) *Scanner {
	return &Scanner{RepoPath: path}
}

// Scan returns commits authored in [from, to) by one of allowedEmails
// (all authors if empty), ordered oldest → newest.
func (s *Scanner) Scan(from, to time.Time, allowedEmails map[string]bool) ([]model.Commit, error) {
	repo, err := git.PlainOpen(s.RepoPath)
	if err != nil {
		return nil, err
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}

	head, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	var mainline map[plumbing.Hash][]string
	if s.AnalyzeMerges {
		mainline, err = mainlineMerges(head)
		if err != nil {
			return nil, fmt.Errorf("merge analysis: %w", err)
		}
	}

	iter, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}

	var commits []model.Commit

	err = iter.ForEach(func(c *object.Commit) error {
		ts := c.Author.When

		if ts.Before(from) || !ts.Before(to) {
			return nil
		}

		if len(allowedEmails) > 0 && !allowedEmails[c.Author.Email] {
			return nil
		}

		commit := model.Commit{
			Hash:        c.Hash.String(),
			AuthorName:  c.Author.Name,
			AuthorEmail: c.Author.Email,
			Message:     c.Message,
			Timestamp:   ts,
			ParentCount: len(c.ParentHashes),
			IsMerge:     len(c.ParentHashes) > 1,
		}

		if !commit.IsMerge {
			s.addDiffStats(c, &commit)
		}

		if commit.IsMerge && s.AnalyzeMerges {
			if merged, ok := mainline[c.Hash]; ok {
				commit.MergedHashes = merged
			} else {
				// Merge inside a side branch: fall back to a walk bounded by the range
				merged, err := mergedCommits(c, from)
				if err != nil {
					s.warn(c, "merge analysis", err)
				}
				commit.MergedHashes = merged
			}
		}

		commits = append(commits, commit)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Reverse so commits are oldest → newest
	for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
		commits[i], commits[j] = commits[j], commits[i]
	}

	return commits, nil
}

// addDiffStats fills in the files and lines a commit changed relative to its
// first parent. Failures (e.g. blobs missing in a partial clone) are recorded
// as warnings so one unreadable commit does not abort the report.
func (s *Scanner) addDiffStats(c *object.Commit, commit *model.Commit) {
	tree, err := c.Tree()
	if err != nil {
		s.warn(c, "tree", err)
		return
	}

	var parentTree *object.Tree
	if c.NumParents() > 0 {
		parent, err := c.Parent(0)
		if err != nil {
			s.warn(c, "parent", err)
			return
		}
		if parentTree, err = parent.Tree(); err != nil {
			s.warn(c, "parent tree", err)
			return
		}
	}

	// Only trees are needed here; renames show up as the old and new path
	changes, err := object.DiffTree(parentTree, tree)
	if err != nil {
		s.warn(c, "diff", err)
		return
	}

	var contentChanges object.Changes
	for _, ch := range changes {
		name := ch.To.Name
		if name == "" {
			name = ch.From.Name
		}
		commit.FilesChanged = append(commit.FilesChanged, name)

		// Submodule entries point at commits in another repo, so they have no blob to diff
		if ch.From.TreeEntry.Mode != filemode.Submodule && ch.To.TreeEntry.Mode != filemode.Submodule {
			contentChanges = append(contentChanges, ch)
		}
	}

	patch, err := contentChanges.Patch()
	if err != nil {
		s.warn(c, "line stats", err)
		return
	}
	for _, st := range patch.Stats() {
		commit.LinesAdded += st.Addition
		commit.LinesRemoved += st.Deletion
	}
}

func (s *Scanner) warn(c *object.Commit, what string, err error) {
	s.Warnings = append(s.Warnings, fmt.Sprintf("%s: skipped %s: %v", c.Hash.String()[:8], what, err))
}

// mainlineMerges walks HEAD's first-parent chain from the root and, for every
// merge on it, records the commits it brought in (git log P1..P2). Each commit
// in history is visited once, so the cost is linear in history size.
func mainlineMerges(head *object.Commit) (map[plumbing.Hash][]string, error) {
	var chain []*object.Commit
	for c := head; ; {
		chain = append(chain, c)
		if c.NumParents() == 0 {
			break
		}
		p, err := c.Parent(0)
		if err != nil {
			return nil, err
		}
		c = p
	}

	// reached holds everything reachable from the mainline commit processed last
	reached := make(map[plumbing.Hash]bool)
	result := make(map[plumbing.Hash][]string)

	for i := len(chain) - 1; i >= 0; i-- {
		c := chain[i]
		reached[c.Hash] = true

		if c.NumParents() < 2 {
			continue
		}

		var merged []string
		var queue []*object.Commit
		for j := 1; j < c.NumParents(); j++ {
			p, err := c.Parent(j)
			if err != nil {
				return nil, err
			}
			queue = append(queue, p)
		}

		for len(queue) > 0 {
			n := queue[0]
			queue = queue[1:]
			if reached[n.Hash] {
				continue
			}
			reached[n.Hash] = true
			merged = append(merged, n.Hash.String())

			err := n.Parents().ForEach(func(p *object.Commit) error {
				queue = append(queue, p)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}

		result[c.Hash] = merged
	}

	return result, nil
}

// mergedCommits returns the commits reachable from the merge's side parents
// but not from its first parent, limited to commits newer than since. It is
// used for merges that are not on HEAD's first-parent chain.
func mergedCommits(merge *object.Commit, since time.Time) ([]string, error) {
	first, err := merge.Parent(0)
	if err != nil {
		return nil, err
	}

	var sides []*object.Commit
	for i := 1; i < merge.NumParents(); i++ {
		p, err := merge.Parent(i)
		if err != nil {
			return nil, err
		}
		sides = append(sides, p)
	}

	hidden, _, err := reachableSince([]*object.Commit{first}, since)
	if err != nil {
		return nil, err
	}

	_, sideOrder, err := reachableSince(sides, since)
	if err != nil {
		return nil, err
	}

	var merged []string
	for _, h := range sideOrder {
		if !hidden[h] {
			merged = append(merged, h.String())
		}
	}
	return merged, nil
}

func reachableSince(starts []*object.Commit, since time.Time) (map[plumbing.Hash]bool, []plumbing.Hash, error) {
	seen := make(map[plumbing.Hash]bool)
	var order []plumbing.Hash
	queue := append([]*object.Commit(nil), starts...)

	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]

		if seen[c.Hash] || c.Committer.When.Before(since) {
			continue
		}
		seen[c.Hash] = true
		order = append(order, c.Hash)

		err := c.Parents().ForEach(func(p *object.Commit) error {
			queue = append(queue, p)
			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	return seen, order, nil
}
