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

	repo, err := git.PlainOpen(path)
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

func ValidateBaseDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("repository path does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", path)
	}

	repos, err := DiscoverGitRepos(path)
	if err != nil {
		return err
	}

	if len(repos) == 0 {
		return fmt.Errorf("no git repositories found under: %s", path)
	}

	for _, r := range repos {
		if err := ValidateRepo(r); err != nil {
			return err
		}
	}

	return nil
}
