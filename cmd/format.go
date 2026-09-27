package main

import (
	"fmt"
	"strings"

	mysql "mcp-gp-mysql/internal"
)

// QueryResult is the row set returned by the database client.
type QueryResult = mysql.QueryResult

// formatQueryResult renders a query result for the MCP client.
// At most 20 rows are included; further rows are summarized so a large
// result does not flood the model context.
func formatQueryResult(result *QueryResult) string {
	if result == nil || result.RowCount == 0 {
		return "Query returned 0 rows."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d rows\n\n", result.RowCount))

	headerLine := strings.Join(result.Columns, " | ")
	sb.WriteString(headerLine)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("-", len(headerLine)))
	sb.WriteString("\n")

	limit := result.RowCount
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		sb.WriteString(formatRow(result.Columns, result.Rows[i]))
		sb.WriteString("\n")
	}
	if result.RowCount > 20 {
		sb.WriteString(fmt.Sprintf("... +%d more rows", result.RowCount-20))
	}

	return sb.String()
}

func formatRow(columns []string, row map[string]interface{}) string {
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
