package git

import (
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Scanner struct {
	RepoPath string
}

func NewScanner(path string) *Scanner {
	return &Scanner{RepoPath: path}
}

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

		if ts.Before(from) || ts.After(to) {
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
			stats, err := c.Stats()
			if err == nil {
				for _, s := range stats {
					commit.LinesAdded += s.Addition
					commit.LinesRemoved += s.Deletion
				}
			}
		}

		if !commit.IsMerge {
			files, err := c.Files()
			if err == nil {
				_ = files.ForEach(func(f *object.File) error {
					commit.FilesChanged = append(commit.FilesChanged, f.Name)
					return nil
				})
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
