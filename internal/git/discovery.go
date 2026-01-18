package git

import (
	"os"
	"path/filepath"
)

// DiscoverGitRepos returns a list of directories that contain a .git folder.
func DiscoverGitRepos(basePath string) ([]string, error) {
	var repos []string

	err := filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// Detect .git directory
		if d.Name() == ".git" {
			repos = append(repos, filepath.Dir(path))
			return filepath.SkipDir
		}

		return nil
	})

	return repos, err
}
