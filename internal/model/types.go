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
