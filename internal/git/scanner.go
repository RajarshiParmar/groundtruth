package git

type Scanner struct {
	RepoPath string
}

func NewScanner(path string) *Scanner {
	return &Scanner{RepoPath: path}
}
