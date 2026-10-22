package model

import "time"

type Commit struct {
	Hash        string
	AuthorName  string
	AuthorEmail string
	Message     string
	Timestamp   time.Time

	IsMerge     bool
	ParentCount int

	LinesAdded   int
	LinesRemoved int

	FilesChanged []string

	// ExcludedLines and ExcludedFiles are the lines and paths left out by
	// exclude_paths; they are not part of LinesAdded, LinesRemoved or FilesChanged
	ExcludedLines int
	ExcludedFiles []string

	// CoAuthored is set when the user is named only in a Co-authored-by trailer
	CoAuthored bool

	// MergedHashes lists the commits a merge brought in (merge analysis only)
	MergedHashes []string
}

type ClassifiedCommit struct {
	Commit

	Type     string
	Scope    string
	Subject  string // header without "type(scope)!: "; the whole header when unclassified
	Breaking bool
}

type MergeSummary struct {
	Hash        string
	Message     string
	Timestamp   time.Time
	CommitCount int
	Types       map[string]int
}

// ContributionScore is the typed result emitted by ContributionScoreMetric.
// Breakdown maps a commit type (e.g. "feat", "fix") to its accumulated score.
type ContributionScore struct {
	Total       float64
	Breakdown   map[string]float64
	CommitCount int
}
