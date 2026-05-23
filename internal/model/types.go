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
}

type ClassifiedCommit struct {
	Commit

	Type     string
	Scope    string
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
