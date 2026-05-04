package main

import (
	"fmt"
	"strings"

	mysql "mcp-gp-mysql/internal"
)

// CompactMode controls output verbosity. When true, responses are optimized
// for token-efficient AI consumption (minimal formatting, one-line summaries).
var CompactMode = false

// Aliases for internal types
type QueryResult = mysql.QueryResult
type TableInfo = mysql.TableInfo
type ColumnInfo = mysql.ColumnInfo

// ============================================================================
// Query Result Formatting
// ============================================================================

// formatQueryResultStructured formats database query results for AI consumption
func formatQueryResultStructured(result *QueryResult) string {
	if CompactMode {
		return formatQueryResultCompact(result)
	}
	return formatQueryResultVerbose(result)
}

func formatQueryResultCompact(result *QueryResult) string {
	if result.RowCount == 0 {
		return "0 rows"
	}

	// Single row: show values inline
	if result.RowCount == 1 {
		return formatRowCompact(result.Columns, result.Rows[0])
	}

	// Multiple rows: tabulated compact
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d rows\n", result.RowCount))

	// Column headers
	sb.WriteString(strings.Join(result.Columns, "\t"))
	sb.WriteString("\n")

	// Data rows (limit to 5)
	limit := result.RowCount
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(formatRowCompact(result.Columns, result.Rows[i]))
		sb.WriteString("\n")
	}
	if result.RowCount > 5 {
		sb.WriteString(fmt.Sprintf("... +%d more rows", result.RowCount-5))
	}

	return sb.String()
}

func formatQueryResultVerbose(result *QueryResult) string {
	if result.RowCount == 0 {
		return "Query returned 0 rows."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d rows\n\n", result.RowCount))

	// Column headers
	headerLine := strings.Join(result.Columns, " | ")
	sb.WriteString(headerLine)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", len(headerLine)))
	sb.WriteString("\n")

	// Data rows
	limit := result.RowCount
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(formatRowCompact(result.Columns, result.Rows[i]))
		sb.WriteString("\n")
	}
	if result.RowCount > 20 {
		sb.WriteString(fmt.Sprintf("... +%d more rows", result.RowCount-20))
	}

	return sb.String()
}

func formatRowCompact(columns []string, row map[string]interface{}) string {
	values := make([]string, len(columns))
	for i, col := range columns {
		if v, ok := row[col]; ok && v != nil {
			values[i] = fmt.Sprintf("%v", v)
		} else {
			values[i] = "NULL"
		}
	}
	return strings.Join(values, "\t")
}

// ============================================================================
// Table/View Listing Formatting
// ============================================================================

// formatTablesList formats table list for AI consumption
func formatTablesList(tables []TableInfo) string {
	if len(tables) == 0 {
		return "No tables found in the database."
	}

	if CompactMode {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%d tables:\n", len(tables)))
		for _, t := range tables {
			sb.WriteString(fmt.Sprintf("• %s [%s, ~%d rows]\n", t.Name, t.Type, t.Rows))
		}
		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d tables:\n\n", len(tables)))

	for _, t := range tables {
		sb.WriteString(fmt.Sprintf("• %s\n", t.Name))
		sb.WriteString(fmt.Sprintf("  Type: %s, Engine: %s, Rows: ~%d\n", t.Type, t.Engine, t.Rows))
		if t.Comment != "" {
			sb.WriteString(fmt.Sprintf("  Comment: %s\n", t.Comment))
		}
	}

	return sb.String()
}

// formatViewsResult formats view list result
func formatViewsResult(result *QueryResult) string {
	if CompactMode {
		return formatQueryResultCompact(result)
	}
	return formatQueryResultVerbose(result)
}

// formatIndexesResult formats index list result
func formatIndexesResult(result *QueryResult) string {
	if CompactMode {
		return formatQueryResultCompact(result)
	}
	return formatQueryResultVerbose(result)
}

// formatExplainResult formats EXPLAIN output
func formatExplainResult(result *QueryResult) string {
	if CompactMode {
		return formatQueryResultCompact(result)
	}
	return formatQueryResultVerbose(result)
}

// ============================================================================
// Table Structure Formatting
// ============================================================================

// formatDescribeTable formats table structure for AI consumption
func formatDescribeTable(table string, columns []ColumnInfo) string {
	if len(columns) == 0 {
		return fmt.Sprintf("Table '%s' not found or has no columns.", table)
	}

	if CompactMode {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%s: %d columns\n", table, len(columns)))
		for _, col := range columns {
			nullStr := ""
			if col.Nullable {
				nullStr = " NULL"
			}
			defaultStr := ""
			if col.Default != "" {
				defaultStr = fmt.Sprintf(" DEFAULT '%s'", col.Default)
			}
			keyStr := ""
			if col.Key != "" {
				keyStr = fmt.Sprintf(" %s", col.Key)
			}
			sb.WriteString(fmt.Sprintf("• %s %s%s%s%s\n", col.Name, col.Type, nullStr, defaultStr, keyStr))
		}
		return sb.String()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Structure of table '%s':\n\n", table))

	for _, col := range columns {
		sb.WriteString(fmt.Sprintf("• %s (%s)\n", col.Name, col.Type))
		if col.Key != "" {
			sb.WriteString(fmt.Sprintf("  Key: %s\n", col.Key))
		}
		if col.Nullable {
			sb.WriteString("  Nullable: Yes\n")
		}
		if col.Default != "" {
			sb.WriteString(fmt.Sprintf("  Default: %s\n", col.Default))
		}
		if col.Extra != "" {
			sb.WriteString(fmt.Sprintf("  Extra: %s\n", col.Extra))
		}
	}

	return sb.String()
}

// ============================================================================
// Count Formatting
// ============================================================================

// formatCountResult formats count result
func formatCountResult(count int64) string {
	return fmt.Sprintf("Count: %d rows", count)
}

// formatCountWithWhere formats count with WHERE clause
func formatCountWithWhere(table string, count int64, where string) string {
	if CompactMode {
		return fmt.Sprintf("%d rows in %s", count, table)
	}
	return fmt.Sprintf("Count: %d rows in %s (WHERE: %s)", count, table, where)
}

// ============================================================================
// Execute Result Formatting
// ============================================================================

// formatExecuteResult formats execute (INSERT/UPDATE/DELETE) result
func formatExecuteResult(affected int64) string {
	return fmt.Sprintf("Rows affected: %d", affected)
}

// formatExecuteResultWithConfirm formats execute result with confirmation
func formatExecuteResultWithConfirm(affected int64, confirmed bool) string {
	status := "confirmed"
	if !confirmed {
		status = "requires confirmation"
	}
	return fmt.Sprintf("Rows affected: %d (%s)", affected, status)
}

// ============================================================================
// Database Info Formatting
// ============================================================================

// formatDatabaseInfo formats database connection information
func formatDatabaseInfo(result *QueryResult) string {
	if result.RowCount == 0 || len(result.Rows) == 0 {
		return "Could not retrieve database information."
	}

	row := result.Rows[0]

	if CompactMode {
		version := getMapValue(row, "version")
		db := getMapValue(row, "current_database")
		user := getMapValue(row, "db_user")
		return fmt.Sprintf("DB: %s | User: %s | Version: %v", db, user, version)
	}

	var sb strings.Builder
	sb.WriteString("Database Connection Info:\n\n")
	sb.WriteString(fmt.Sprintf("• Version: %v\n", getMapValue(row, "version")))
	sb.WriteString(fmt.Sprintf("• Info: %v\n", getMapValue(row, "version_info")))
	sb.WriteString(fmt.Sprintf("• Database: %v\n", getMapValue(row, "current_database")))
	sb.WriteString(fmt.Sprintf("• User: %v\n", getMapValue(row, "db_user")))
	sb.WriteString(fmt.Sprintf("• Host: %v\n", getMapValue(row, "hostname")))
	sb.WriteString(fmt.Sprintf("• Port: %v\n", getMapValue(row, "port")))

	return sb.String()
}

// ============================================================================
// Error Formatting
// ============================================================================

// formatSecurityError formats security validation errors
func formatSecurityError(operation string, details string) string {
	return fmt.Sprintf("Security validation failed for %s: %s", operation, details)
}

// formatRateLimitError formats rate limit errors
func formatRateLimitError(opType string) string {
	return fmt.Sprintf("Rate limit exceeded for %s operations. Please try again later.", opType)
}

// ============================================================================
// Batch Results Formatting
// ============================================================================

// formatBatchCount formats multiple count results
func formatBatchCount(results map[string]int64) string {
	if CompactMode {
		var parts []string
		for table, count := range results {
			parts = append(parts, fmt.Sprintf("%s=%d", table, count))
		}
		return strings.Join(parts, " | ")
	}

	var sb strings.Builder
	sb.WriteString("Row counts:\n")
	for table, count := range results {
		sb.WriteString(fmt.Sprintf("• %s: %d\n", table, count))
	}
	return sb.String()
}