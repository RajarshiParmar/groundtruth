package git

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
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
