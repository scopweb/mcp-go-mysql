package main

import (
	"strings"

	mysql "mcp-gp-mysql/internal"
)

// SQL validation helpers - centralized query type detection

// isReadOnlyQuery checks if a query is read-only (SELECT, WITH, SHOW)
func isReadOnlyQuery(sql string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(sql))
	return strings.HasPrefix(normalized, "SELECT") ||
		strings.HasPrefix(normalized, "WITH") ||
		strings.HasPrefix(normalized, "SHOW")
}

// isWriteQuery checks if a query is a write operation (INSERT, UPDATE, DELETE)
func isWriteQuery(sql string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(mysql.StripComments(sql)))
	return strings.HasPrefix(normalized, "INSERT") ||
		strings.HasPrefix(normalized, "UPDATE") ||
		strings.HasPrefix(normalized, "DELETE")
}

// isDDLQuery checks if a query is DDL (CREATE, DROP, ALTER, TRUNCATE)
func isDDLQuery(sql string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(mysql.StripComments(sql)))
	return strings.HasPrefix(normalized, "CREATE") ||
		strings.HasPrefix(normalized, "DROP") ||
		strings.HasPrefix(normalized, "ALTER") ||
		strings.HasPrefix(normalized, "TRUNCATE")
}

// isSelectOnly checks if query is strictly SELECT (for EXPLAIN)
func isSelectOnly(sql string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(sql))
	return strings.HasPrefix(normalized, "SELECT")
}
