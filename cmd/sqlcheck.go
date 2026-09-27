package main

import (
	"strings"

	mysql "mcp-gp-mysql/internal"
)

// leadingVerb returns the first SQL token after comments are stripped.
// Tool gates use this so a leading comment cannot hide the real verb.
// The classifier in internal remains the security boundary.
func leadingVerb(sql string) string {
	normalized := strings.ToUpper(strings.TrimSpace(mysql.StripComments(sql)))
	if normalized == "" {
		return ""
	}
	for i, r := range normalized {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '(' {
			return normalized[:i]
		}
	}
	return normalized
}

// isReadOnlyQuery reports whether sql is SELECT, WITH, or SHOW.
// DESCRIBE, EXPLAIN, and USE are allowed by the classifier but not by the
// query tool: describe and explain are separate tools.
func isReadOnlyQuery(sql string) bool {
	switch leadingVerb(sql) {
	case "SELECT", "WITH", "SHOW":
		return true
	default:
		return false
	}
}

// isSelectOnly reports whether sql is a SELECT. The explain tool only
// accepts SELECT plans.
func isSelectOnly(sql string) bool {
	return leadingVerb(sql) == "SELECT"
}
