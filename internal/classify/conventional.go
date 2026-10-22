package classify

import (
	"regexp"
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/model"
)

var conventionalRegex = regexp.MustCompile(
	`^(?P<type>[a-zA-Z]+)(\((?P<scope>[^)]+)\))?(?P<breaking>!)?:\s+(?P<message>.+)`,
)

func NewConventionalClassifier() Classifier {
	return &conventionalClassifier{}
}

type conventionalClassifier struct{}

func (c *conventionalClassifier) Classify(commit model.Commit) model.ClassifiedCommit {
	result := model.ClassifiedCommit{
		Commit: commit,
		Type:   "unknown",
	}

	lines := strings.Split(commit.Message, "\n")
	header := strings.TrimSpace(lines[0])
	result.Subject = header

	matches := conventionalRegex.FindStringSubmatch(header)
	if matches == nil {
		return result
	}

	for i, name := range conventionalRegex.SubexpNames() {
		switch name {
		case "type":
			result.Type = strings.ToLower(matches[i])
		case "scope":
			result.Scope = strings.ToLower(matches[i])
		case "breaking":
			result.Breaking = matches[i] != ""
		case "message":
			result.Subject = matches[i]
		}
	}

	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "BREAKING CHANGE") || strings.HasPrefix(line, "BREAKING-CHANGE") {
			result.Breaking = true
		}
	}

	return result
}
