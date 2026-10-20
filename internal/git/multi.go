package git

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

// RepoScan is the outcome of scanning one repository
type RepoScan struct {
	Path     string
	Commits  []model.Commit
	Warnings []string
	Err      error
}

// ScanRepos scans repositories concurrently and returns one result per path,
// in the same order as paths.
func ScanRepos(paths []string, from, to time.Time, allowedEmails map[string]bool, analyzeMerges bool) []RepoScan {
	results := make([]RepoScan, len(paths))

	workers := runtime.NumCPU()
	if workers > len(paths) {
		workers = len(paths)
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				scanner := NewScanner(paths[i])
				scanner.AnalyzeMerges = analyzeMerges
				commits, err := scanner.Scan(from, to, allowedEmails)
				results[i] = RepoScan{Path: paths[i], Commits: commits, Warnings: scanner.Warnings, Err: err}
			}
		}()
	}

	for i := range paths {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return results
}

// MergeScans combines per-repository commits, oldest → newest. Commits seen
// in more than one repository (forks, backup clones) are kept once. Failed
// repositories are skipped and reported in warnings; it is an error only if
// every repository failed.
func MergeScans(scans []RepoScan) (commits []model.Commit, warnings []string, duplicates int, err error) {
	seen := make(map[string]bool)
	var lastErr error
	failed := 0

	for _, s := range scans {
		for _, w := range s.Warnings {
			warnings = append(warnings, fmt.Sprintf("%s: %s", s.Path, w))
		}

		if s.Err != nil {
			failed++
			lastErr = fmt.Errorf("%s: %w", s.Path, s.Err)
			if len(scans) > 1 {
				warnings = append(warnings, fmt.Sprintf("%s: skipped repository: %v", s.Path, s.Err))
			}
			continue
		}

		for _, c := range s.Commits {
			if seen[c.Hash] {
				duplicates++
				continue
			}
			seen[c.Hash] = true
			commits = append(commits, c)
		}
	}

	if failed > 0 && failed == len(scans) {
		if failed == 1 {
			return nil, warnings, 0, lastErr
		}
		return nil, warnings, 0, fmt.Errorf("all %d repositories failed to scan", failed)
	}

	sort.SliceStable(commits, func(i, j int) bool {
		return commits[i].Timestamp.Before(commits[j].Timestamp)
	})

	return commits, warnings, duplicates, nil
}
