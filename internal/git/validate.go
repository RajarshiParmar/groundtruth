package git

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
)

// ValidateRepo checks whether the given path is a usable git repository.
func ValidateRepo(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("repository path does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", path)
	}

	repo, err := openRepo(path)
	if err != nil {
		return fmt.Errorf("not a git repository: %s", path)
	}

	// Ensure HEAD exists
	_, err = repo.Head()
	if err != nil {
		return fmt.Errorf("git repository has no HEAD: %s", path)
	}

	// Ensure commit log is readable (cheap sanity check)
	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return fmt.Errorf("failed to read commit log: %s", path)
	}
	defer iter.Close()

	// Try to read at least one commit
	_, err = iter.Next()
	if err != nil {
		return fmt.Errorf("git repository has no commits: %s", path)
	}

	return nil
}

// ResolveRepos returns the repositories to scan. For a single repository the
// path must be valid. With baseDir, every checkout found under path is
// validated; unusable ones are skipped and described in warnings, and an
// error is returned only if none are usable.
func ResolveRepos(path string, baseDir bool) (repos []string, warnings []string, err error) {
	if !baseDir {
		if err := ValidateRepo(path); err != nil {
			return nil, nil, err
		}
		return []string{path}, nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("repository path does not exist: %s", path)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("repository path is not a directory: %s", path)
	}

	found, skipped, err := DiscoverGitRepos(path)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range skipped {
		warnings = append(warnings, "skipped unreadable directory "+s)
	}

	if len(found) == 0 {
		return nil, warnings, fmt.Errorf("no git repositories found under: %s", path)
	}

	for _, r := range found {
		if err := ValidateRepo(r); err != nil {
			warnings = append(warnings, "skipped "+err.Error())
			continue
		}
		repos = append(repos, r)
	}

	if len(repos) == 0 {
		return nil, warnings, fmt.Errorf("no usable git repositories found under: %s", path)
	}

	return repos, warnings, nil
}

// openRepo opens a checkout, including linked worktrees whose .git is a file
func openRepo(path string) (*git.Repository, error) {
	return git.PlainOpenWithOptions(path, &git.PlainOpenOptions{EnableDotGitCommonDir: true})
}
