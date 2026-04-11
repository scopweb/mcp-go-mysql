package main

import "strings"

// stripSQLComments removes SQL comments (-- and /* */) to avoid false positives in query analysis
func stripSQLComments(s string) string {
	// Remove line comments
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if idx := strings.Index(l, "--"); idx >= 0 {
			lines[i] = l[:idx]
		}
		if idx := strings.Index(l, "#"); idx >= 0 { // MySQL style
			lines[i] = l[:idx]
		}
	}
	s = strings.Join(lines, "\n")
	// Remove block comments
	for {
		start := strings.Index(s, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(s[start+2:], "*/")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+2+end+2:]
	}
	// Normalize whitespace
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
