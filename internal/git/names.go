package git

import "path/filepath"

// RepoNames names each repository for the report: in base_dir mode its path
// relative to base, slash-separated; otherwise, or when the base folder is
// itself the repository, the folder's own name.
func RepoNames(base string, paths []string, baseDir bool) []string {
	names := make([]string, len(paths))
	for i, p := range paths {
		if baseDir {
			if rel, err := filepath.Rel(base, p); err == nil && rel != "." {
				names[i] = filepath.ToSlash(rel)
				continue
			}
		}
		names[i] = folderName(p)
	}
	return names
}

func folderName(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return filepath.Base(filepath.Clean(p))
}

// NameCommits records each scan's repository name on the scan and on every
// commit in it, so commits keep their repo after MergeScans combines them
func NameCommits(scans []RepoScan, names []string) {
	for i := range scans {
		scans[i].Name = names[i]
		for j := range scans[i].Commits {
			scans[i].Commits[j].Repo = names[i]
		}
	}
}
