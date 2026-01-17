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
