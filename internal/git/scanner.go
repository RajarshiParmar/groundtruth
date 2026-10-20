package git

import (
	"fmt"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Scanner struct {
	RepoPath string

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
