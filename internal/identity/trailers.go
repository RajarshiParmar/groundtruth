package identity

import (
	"regexp"
	"strings"
)

// Person is a name and email from a commit trailer
type Person struct {
	Name, Email string
}

var coAuthorLine = regexp.MustCompile(`(?i)^co-authored-by\s*:\s*(.*?)\s*<([^<>]+)>\s*$`)

// A paragraph ends at a line that is empty or only whitespace
var paragraphBreak = regexp.MustCompile(`\n[ \t]*\n`)

// CoAuthors reads Co-authored-by trailers from the message's trailer block:
// its last paragraph, and never the subject line.
func CoAuthors(message string) []Person {
	message = strings.TrimSpace(strings.ReplaceAll(message, "\r\n", "\n"))
	paragraphs := paragraphBreak.Split(message, -1)
	if len(paragraphs) < 2 {
		return nil
	}

	var people []Person
	for _, line := range strings.Split(paragraphs[len(paragraphs)-1], "\n") {
		if p := coAuthorLine.FindStringSubmatch(strings.TrimSpace(line)); p != nil {
			people = append(people, Person{Name: p[1], Email: p[2]})
		}
	}
	return people
}
