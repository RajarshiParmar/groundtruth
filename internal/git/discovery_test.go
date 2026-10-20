package git

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return p
}

func TestDiscoverGitRepos(t *testing.T) {
	base := t.TempDir()

	// Plain repo with a nested fixture checkout that must not be reported
	mkdir(t, base, "app", ".git")
	mkdir(t, base, "app", "testdata", "fixture", ".git")

	// Linked worktree / submodule style checkout: .git is a file
	wt := mkdir(t, base, "app-feature")
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: ../app/.git/worktrees/feature\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	mkdir(t, base, "notes")

	repos, skipped, err := DiscoverGitRepos(base)
	if err != nil {
		t.Fatalf("DiscoverGitRepos: %v", err)
	}

	want := []string{filepath.Join(base, "app"), filepath.Join(base, "app-feature")}
	if !reflect.DeepEqual(repos, want) {
		t.Errorf("repos = %v, want %v", repos, want)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
}

func TestDiscoverGitRepos_SkipsUnreadableDirs(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions are not enforced for root")
	}

	base := t.TempDir()
	mkdir(t, base, "app", ".git")
	locked := mkdir(t, base, "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0755) })

	repos, skipped, err := DiscoverGitRepos(base)
	if err != nil {
		t.Fatalf("DiscoverGitRepos: %v", err)
	}
	if len(repos) != 1 || repos[0] != filepath.Join(base, "app") {
		t.Errorf("repos = %v, want [app]", repos)
	}
	if len(skipped) != 1 {
		t.Errorf("skipped = %v, want the locked directory", skipped)
	}
}

func TestMergeScans(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	scans := []RepoScan{
		{Path: "a", Commits: []model.Commit{{Hash: "h2", Timestamp: t2}, {Hash: "h1", Timestamp: t1}}},
		{Path: "a-fork", Commits: []model.Commit{{Hash: "h1", Timestamp: t1}}},
		{Path: "broken", Err: errors.New("reference not found")},
	}

	commits, warnings, duplicates, err := MergeScans(scans)
	if err != nil {
		t.Fatalf("MergeScans: %v", err)
	}
	if len(commits) != 2 || commits[0].Hash != "h1" || commits[1].Hash != "h2" {
		t.Errorf("commits = %v, want h1, h2 in time order", commits)
	}
	if duplicates != 1 {
		t.Errorf("duplicates = %d, want 1", duplicates)
	}
	if len(warnings) != 1 {
		t.Errorf("warnings = %v, want one for the broken repo", warnings)
	}
}

func TestMergeScans_AllFailed(t *testing.T) {
	if _, _, _, err := MergeScans([]RepoScan{{Path: "only", Err: errors.New("boom")}}); err == nil {
		t.Error("single failed repo: want error")
	}

	scans := []RepoScan{{Path: "a", Err: errors.New("x")}, {Path: "b", Err: errors.New("y")}}
	if _, _, _, err := MergeScans(scans); err == nil {
		t.Error("all repos failed: want error")
	}
}
