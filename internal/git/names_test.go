package git

import (
	"reflect"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

func TestRepoNames(t *testing.T) {
	cases := []struct {
		base    string
		paths   []string
		baseDir bool
		want    []string
	}{
		{"/w", []string{"/w/services/api", "/w/web"}, true, []string{"services/api", "web"}},
		{"/w", []string{"/w"}, true, []string{"w"}}, // the base folder is itself the repo
		{"/x/proj", []string{"/x/proj"}, false, []string{"proj"}},
		{".", []string{"./a", "b/c"}, true, []string{"a", "b/c"}},
	}
	for _, tc := range cases {
		if got := RepoNames(tc.base, tc.paths, tc.baseDir); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("RepoNames(%q, %q, %v) = %q, want %q", tc.base, tc.paths, tc.baseDir, got, tc.want)
		}
	}
}

func TestNameCommits(t *testing.T) {
	scans := []RepoScan{
		{Path: "/w/a", Commits: []model.Commit{{Hash: "1"}, {Hash: "2"}}},
		{Path: "/w/b", Commits: []model.Commit{{Hash: "3"}}},
	}
	NameCommits(scans, []string{"a", "b"})
	for _, s := range scans {
		if s.Name != s.Path[len("/w/"):] {
			t.Errorf("%s: Name = %q", s.Path, s.Name)
		}
		for _, c := range s.Commits {
			if c.Repo != s.Name {
				t.Errorf("%s: commit %s Repo = %q, want %q", s.Path, c.Hash, c.Repo, s.Name)
			}
		}
	}
}

func TestMergeScans_FirstRepoKeepsDuplicate(t *testing.T) {
	scans := []RepoScan{
		{Path: "/w/a", Commits: []model.Commit{{Hash: "x"}}},
		{Path: "/w/b", Commits: []model.Commit{{Hash: "x"}}},
	}
	NameCommits(scans, []string{"a", "b"})
	commits, _, dups, err := MergeScans(scans)
	if err != nil || len(commits) != 1 || dups != 1 || commits[0].Repo != "a" {
		t.Errorf("MergeScans = %+v, dups %d, err %v; want one commit credited to a", commits, dups, err)
	}
}
