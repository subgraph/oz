package parser

import "strings"

// LineType represents the different types of lines available in a policy file
type LineType int

const (
	unknownLine LineType = iota
	ruleLine
	commentLine
	assignmentLine
	defaultAssignmentLine
	emptyLine
)

func isComment(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "#")
}

func isRule(s string) bool {
	return len(strings.SplitN(s, ":", 2)) == 2
}

func isEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func isDefaultAssignment(s string) bool {
	result := strings.SplitN(s, "=", 2)
	if len(result) == 2 {
		c := strings.TrimSpace(result[0])
		switch c {
		case "DEFAULT_POSITIVE", "DEFAULT_NEGATIVE", "DEFAULT_POLICY":
			return true
		}
	}
	return false
}

func isAssignment(s string) bool {
	return len(strings.SplitN(s, "=", 2)) == 2
}

func lineType(s string) LineType {
	if isComment(s) {
		return commentLine
	}

	if isRule(s) {
		return ruleLine
	}

	if isDefaultAssignment(s) {
		return defaultAssignmentLine
	}

	if isAssignment(s) {
		return assignmentLine
	}

	if isEmpty(s) {
		return emptyLine
	}

	return unknownLine
}
