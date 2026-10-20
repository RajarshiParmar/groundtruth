package git

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiscoverGitRepos returns the directories under basePath that are git
// checkouts. A `.git` directory or a `.git` file (linked worktrees,
// submodules) both count. Repositories are not searched for nested
// checkouts, so vendored clones and test fixtures are not picked up.
// Subdirectories that cannot be read are skipped and returned in skipped.
func DiscoverGitRepos(basePath string) (repos []string, skipped []string, err error) {
	err = filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if path == basePath {
				return err
			}
			skipped = append(skipped, fmt.Sprintf("%s: %v", path, err))
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if _, statErr := os.Lstat(filepath.Join(path, ".git")); statErr == nil {
			repos = append(repos, path)
			return filepath.SkipDir
		}

		return nil
	})

	return repos, skipped, err
}
