package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateRepo(t *testing.T) {
	good := newTestRepo(t)
	good.commit(testCommit{msg: "feat: a", when: day(0), files: map[string]string{"a": "1"}})

	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cases := []struct {
		name, path, wantErr string
	}{
		{"valid", good.dir, ""},
		{"missing", filepath.Join(t.TempDir(), "nope"), "does not exist"},
		{"file", file, "not a directory"},
		{"plain dir", t.TempDir(), "not a git repository"},
		{"no commits", newTestRepo(t).dir, "no HEAD"},
	}
	for _, tc := range cases {
		err := ValidateRepo(tc.path)
		switch {
		case tc.wantErr == "" && err != nil:
			t.Errorf("%s: unexpected error %v", tc.name, err)
		case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
			t.Errorf("%s: error %v, want %q", tc.name, err, tc.wantErr)
		}
	}
}

func TestResolveRepos_Single(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "feat: a", when: day(0), files: map[string]string{"a": "1"}})

	repos, warnings, err := ResolveRepos(r.dir, false)
	if err != nil || !reflect.DeepEqual(repos, []string{r.dir}) || len(warnings) != 0 {
		t.Errorf("ResolveRepos = %v, %v, %v; want [%s], none, nil", repos, warnings, err, r.dir)
	}

	if _, _, err := ResolveRepos(t.TempDir(), false); err == nil {
		t.Error("plain directory without base_dir: want error")
	}
}

func TestResolveRepos_BaseDir(t *testing.T) {
	base := t.TempDir()

	good := filepath.Join(base, "good")
	newTestRepoAt(t, good).commit(testCommit{msg: "feat: a", when: day(0), files: map[string]string{"a": "1"}})

	// An initialised repository without commits is skipped with a warning
	newTestRepoAt(t, filepath.Join(base, "empty"))
	mkdir(t, base, "notes")

	repos, warnings, err := ResolveRepos(base, true)
	if err != nil {
		t.Fatalf("ResolveRepos: %v", err)
	}
	if !reflect.DeepEqual(repos, []string{good}) {
		t.Errorf("repos = %v, want [%s]", repos, good)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "empty") {
		t.Errorf("warnings = %v, want one for the empty repo", warnings)
	}
}

func TestResolveRepos_BaseDirErrors(t *testing.T) {
	none := t.TempDir()
	mkdir(t, none, "notes")
	if _, _, err := ResolveRepos(none, true); err == nil || !strings.Contains(err.Error(), "no git repositories") {
		t.Errorf("no repos: error %v", err)
	}

	unusable := t.TempDir()
	newTestRepoAt(t, filepath.Join(unusable, "empty"))
	if _, _, err := ResolveRepos(unusable, true); err == nil || !strings.Contains(err.Error(), "no usable") {
		t.Errorf("only unusable repos: error %v", err)
	}

	if _, _, err := ResolveRepos(filepath.Join(none, "missing"), true); err == nil {
		t.Error("missing base dir: want error")
	}
}

// Linked worktrees have a .git file pointing at the main repository's
// common dir; go-git needs EnableDotGitCommonDir to open them.
func TestLinkedWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git CLI not available")
	}

	r := newTestRepo(t)
	want := r.commit(testCommit{msg: "feat: a", when: day(0), files: map[string]string{"a": "1"}})

	wtPath := filepath.Join(t.TempDir(), "linked")
	cmd := exec.Command("git", "-C", r.dir, "worktree", "add", "-b", "linked", wtPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}

	if err := ValidateRepo(wtPath); err != nil {
		t.Fatalf("ValidateRepo(worktree): %v", err)
	}
	commits, err := NewScanner(wtPath).Scan(day(-1), day(1), nil)
	if err != nil {
		t.Fatalf("Scan(worktree): %v", err)
	}
	if len(commits) != 1 || commits[0].Hash != want.String() {
		t.Errorf("worktree scan = %v, want [%s]", hashes(commits), want)
	}
}
