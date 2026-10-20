package git

import (
	"reflect"
	"sort"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/model"
	"github.com/go-git/go-git/v5/plumbing"
)

func scan(t *testing.T, r *testRepo, analyze bool, emails map[string]bool) []model.Commit {
	t.Helper()
	s := NewScanner(r.dir)
	s.AnalyzeMerges = analyze
	commits, err := s.Scan(day(-1), day(100), emails)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(s.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", s.Warnings)
	}
	return commits
}

func hashes(commits []model.Commit) []string {
	out := make([]string, len(commits))
	for i, c := range commits {
		out[i] = c.Hash
	}
	return out
}

func byHash(t *testing.T, commits []model.Commit, h plumbing.Hash) model.Commit {
	t.Helper()
	for _, c := range commits {
		if c.Hash == h.String() {
			return c
		}
	}
	t.Fatalf("commit %s not in scan", h.String()[:8])
	return model.Commit{}
}

// sameSet compares hash lists ignoring order
func sameSet(got []string, want ...plumbing.Hash) bool {
	w := make([]string, len(want))
	for i, h := range want {
		w[i] = h.String()
	}
	g := append([]string(nil), got...)
	sort.Strings(g)
	sort.Strings(w)
	return reflect.DeepEqual(g, w)
}

func TestScan_TimeRangeAndOrder(t *testing.T) {
	r := newTestRepo(t)
	c0 := r.commit(testCommit{msg: "feat: zero", when: day(0), files: map[string]string{"a": "0"}})
	c1 := r.commit(testCommit{msg: "feat: one", when: day(1), files: map[string]string{"a": "1"}})
	c2 := r.commit(testCommit{msg: "feat: two", when: day(2), files: map[string]string{"a": "2"}})

	all := scan(t, r, false, nil)
	if want := []string{c0.String(), c1.String(), c2.String()}; !reflect.DeepEqual(hashes(all), want) {
		t.Errorf("order = %v, want oldest → newest %v", hashes(all), want)
	}

	// from is inclusive, to is exclusive
	got, err := NewScanner(r.dir).Scan(day(1), day(2), nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if want := []string{c1.String()}; !reflect.DeepEqual(hashes(got), want) {
		t.Errorf("[day1, day2) = %v, want %v", hashes(got), want)
	}

	// Commits outside the range are skipped, not used to stop the walk early
	got, err = NewScanner(r.dir).Scan(day(0), day(1), nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if want := []string{c0.String()}; !reflect.DeepEqual(hashes(got), want) {
		t.Errorf("[day0, day1) = %v, want %v", hashes(got), want)
	}
}

func TestScan_AuthorFilter(t *testing.T) {
	r := newTestRepo(t)
	mine := r.commit(testCommit{msg: "feat: mine", when: day(0), files: map[string]string{"a": "1"}})
	r.commit(testCommit{msg: "feat: theirs", email: "other@example.com", when: day(1), files: map[string]string{"a": "2"}})

	got := scan(t, r, false, map[string]bool{testEmail: true})
	if want := []string{mine.String()}; !reflect.DeepEqual(hashes(got), want) {
		t.Errorf("filtered = %v, want %v", hashes(got), want)
	}

	if got := scan(t, r, false, nil); len(got) != 2 {
		t.Errorf("no filter: %d commits, want 2", len(got))
	}
}

func TestScan_DiffStats(t *testing.T) {
	r := newTestRepo(t)
	add := r.commit(testCommit{msg: "feat: add", when: day(0), files: map[string]string{"a.txt": "1\n2\n3\n"}})
	edit := r.commit(testCommit{msg: "fix: edit", when: day(1), files: map[string]string{
		"a.txt":     "1\nX\n3\n",
		"dir/b.txt": "b\n",
	}})
	del := r.commit(testCommit{msg: "chore: remove", when: day(2), files: map[string]string{"dir/b.txt": deleted}})

	commits := scan(t, r, false, nil)

	cases := []struct {
		name           string
		hash           plumbing.Hash
		added, removed int
		files          []string
	}{
		{"root commit", add, 3, 0, []string{"a.txt"}},
		{"edit and add", edit, 2, 1, []string{"a.txt", "dir/b.txt"}},
		{"delete", del, 0, 1, []string{"dir/b.txt"}},
	}
	for _, tc := range cases {
		c := byHash(t, commits, tc.hash)
		files := append([]string(nil), c.FilesChanged...)
		sort.Strings(files)
		if c.LinesAdded != tc.added || c.LinesRemoved != tc.removed || !reflect.DeepEqual(files, tc.files) {
			t.Errorf("%s: +%d -%d %v, want +%d -%d %v",
				tc.name, c.LinesAdded, c.LinesRemoved, files, tc.added, tc.removed, tc.files)
		}
	}
}

func TestScan_MainlineMerges(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "feat: base", when: day(0), files: map[string]string{"a": "0"}})

	r.checkout("feature", true)
	b := r.commit(testCommit{msg: "feat: b", when: day(1), files: map[string]string{"b": "1"}})
	c := r.commit(testCommit{msg: "fix: c", when: day(2), files: map[string]string{"c": "1"}})

	r.checkout("main", false)
	r.commit(testCommit{msg: "docs: d", when: day(3), files: map[string]string{"d": "1"}})
	m1 := r.merge("feature", day(4))

	r.checkout("feature", false)
	e := r.commit(testCommit{msg: "feat: e", when: day(5), files: map[string]string{"e": "1"}})

	r.checkout("main", false)
	m2 := r.merge("feature", day(6))

	commits := scan(t, r, true, nil)

	merge1 := byHash(t, commits, m1)
	if !merge1.IsMerge || merge1.ParentCount != 2 {
		t.Errorf("m1: IsMerge=%v ParentCount=%d, want true 2", merge1.IsMerge, merge1.ParentCount)
	}
	if merge1.LinesAdded != 0 || len(merge1.FilesChanged) != 0 {
		t.Errorf("m1: merges should carry no diff stats, got +%d %v", merge1.LinesAdded, merge1.FilesChanged)
	}
	if !sameSet(merge1.MergedHashes, b, c) {
		t.Errorf("m1 merged %v, want b and c", merge1.MergedHashes)
	}

	// A second merge of the same branch only brings in what is new
	if merge2 := byHash(t, commits, m2); !sameSet(merge2.MergedHashes, e) {
		t.Errorf("m2 merged %v, want only e", merge2.MergedHashes)
	}

	for _, cm := range commits {
		if !cm.IsMerge && cm.MergedHashes != nil {
			t.Errorf("%s: non-merge has MergedHashes %v", cm.Hash[:8], cm.MergedHashes)
		}
	}

	// Without merge analysis nothing is attributed
	for _, cm := range scan(t, r, false, nil) {
		if cm.MergedHashes != nil {
			t.Errorf("%s: MergedHashes set without analysis", cm.Hash[:8])
		}
	}
}

func TestScan_MergeInsideSideBranch(t *testing.T) {
	r := newTestRepo(t)
	r.commit(testCommit{msg: "feat: base", when: day(0), files: map[string]string{"a": "0"}})

	r.checkout("feature", true)
	b := r.commit(testCommit{msg: "feat: b", when: day(1), files: map[string]string{"b": "1"}})

	r.checkout("sub", true)
	s1 := r.commit(testCommit{msg: "feat: s1", when: day(2), files: map[string]string{"s": "1"}})
	s2 := r.commit(testCommit{msg: "feat: s2", when: day(3), files: map[string]string{"s": "2"}})

	r.checkout("feature", false)
	f := r.commit(testCommit{msg: "fix: f", when: day(4), files: map[string]string{"f": "1"}})
	inner := r.merge("sub", day(5))

	r.checkout("main", false)
	outer := r.merge("feature", day(6))

	commits := scan(t, r, true, nil)

	// inner is not on HEAD's first-parent chain, so it uses the bounded walk
	if got := byHash(t, commits, inner).MergedHashes; !sameSet(got, s1, s2) {
		t.Errorf("inner merged %v, want s1 and s2", got)
	}
	if got := byHash(t, commits, outer).MergedHashes; !sameSet(got, b, s1, s2, f, inner) {
		t.Errorf("outer merged %v, want b, s1, s2, f and inner", got)
	}
}

func TestScan_Errors(t *testing.T) {
	if _, err := NewScanner(t.TempDir()).Scan(day(0), day(1), nil); err == nil {
		t.Error("plain directory: want error")
	}

	// A freshly initialised repository has no HEAD commit
	if _, err := NewScanner(newTestRepo(t).dir).Scan(day(0), day(1), nil); err == nil {
		t.Error("empty repository: want error")
	}
}

func TestScanRepos_KeepsPathOrder(t *testing.T) {
	a := newTestRepo(t)
	a.commit(testCommit{msg: "feat: a", when: day(0), files: map[string]string{"a": "1"}})
	b := newTestRepo(t)
	b.commit(testCommit{msg: "feat: b1", when: day(0), files: map[string]string{"b": "1"}})
	b.commit(testCommit{msg: "feat: b2", when: day(1), files: map[string]string{"b": "2"}})
	missing := t.TempDir()

	paths := []string{b.dir, missing, a.dir}
	scans := ScanRepos(paths, day(-1), day(100), nil, true)

	if len(scans) != len(paths) {
		t.Fatalf("got %d results, want %d", len(scans), len(paths))
	}
	for i, s := range scans {
		if s.Path != paths[i] {
			t.Errorf("result %d is %s, want %s", i, s.Path, paths[i])
		}
	}
	if len(scans[0].Commits) != 2 || scans[0].Err != nil {
		t.Errorf("b: %d commits, err %v; want 2, nil", len(scans[0].Commits), scans[0].Err)
	}
	if scans[1].Err == nil {
		t.Error("missing repo: want error")
	}
	if len(scans[2].Commits) != 1 || scans[2].Err != nil {
		t.Errorf("a: %d commits, err %v; want 1, nil", len(scans[2].Commits), scans[2].Err)
	}
}
