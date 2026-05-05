package main

import (
	"fmt"
	"strings"

	mysql "mcp-gp-mysql/internal"
)

// Tool definitions for MCP protocol
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// getToolsList returns the list of available tools
func getToolsList() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "query",
			Title:       "Query Database",
			Description: "Execute a SELECT query on the MySQL database. Only SELECT queries are allowed for safety.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sql": map[string]interface{}{
						"type":        "string",
						"description": "The SELECT SQL query to execute",
					},
				},
				"required": []string{"sql"},
			},
		},
		{
			Name:        "execute",
			Title:       "Execute Statement",
			Description: "Execute an INSERT, UPDATE, or DELETE query. Requires confirmation key for large operations (>100 rows affected).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sql": map[string]interface{}{
						"type":        "string",
						"description": "The SQL statement to execute (INSERT, UPDATE, DELETE)",
					},
					"confirm_key": map[string]interface{}{
						"type":        "string",
						"description": "Safety confirmation key for large operations",
					},
				},
				"required": []string{"sql"},
			},
		},
		{
			Name:        "tables",
			Title:       "List Tables",
			Description: "List all tables in the current database with their metadata (type, engine, row count, comments).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "describe",
			Title:       "Describe Table",
			Description: "Describe the structure of a specific table, including columns, types, keys, and constraints.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"table": map[string]interface{}{
						"type":        "string",
						"description": "The name of the table to describe",
					},
				},
				"required": []string{"table"},
			},
		},
		{
			Name:        "views",
			Title:       "List Views",
			Description: "List all views in the current database.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "indexes",
			Title:       "Show Indexes",
			Description: "Show indexes for a specific table.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"table": map[string]interface{}{
						"type":        "string",
						"description": "The name of the table to show indexes for",
					},
				},
				"required": []string{"table"},
			},
		},
		{
			Name:        "explain",
			Title:       "Explain Query",
			Description: "Explain the execution plan for a query.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sql": map[string]interface{}{
						"type":        "string",
						"description": "The SQL query to explain",
					},
				},
				"required": []string{"sql"},
			},
		},
		{
			Name:        "count",
			Title:       "Count Rows",
			Description: "Count rows in a table. For filtered counts, use 'query' with SELECT COUNT(*) ... WHERE.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"table": map[string]interface{}{
						"type":        "string",
						"description": "The table name to count rows from",
					},
				},
				"required": []string{"table"},
			},
		},
		{
			Name:        "sample",
			Title:       "Sample Rows",
			Description: "Get a sample of rows from a table (default 10 rows).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"table": map[string]interface{}{
						"type":        "string",
						"description": "The table name to sample from",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Number of rows to return (default: 10, max: 100)",
					},
				},
				"required": []string{"table"},
			},
		},
		{
			Name:        "database_info",
			Title:       "Database Info",
			Description: "Get information about the current database connection and server.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// callClientMethod routes tool calls to the appropriate client method
func callClientMethod(client *mysql.Client, toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "query":
		return handleQuery(client, args)
	case "execute":
		return handleExecute(client, args)
	case "tables":
		return handleTables(client)
	case "describe":
		return handleDescribe(client, args)
	case "views":
		return handleViews(client)
	case "indexes":
		return handleIndexes(client, args)
	case "explain":
		return handleExplain(client, args)
	case "count":
		return handleCount(client, args)
	case "sample":
		return handleSample(client, args)
	case "database_info":
		return handleDatabaseInfo(client)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// handleQuery executes a SELECT query
func handleQuery(client *mysql.Client, args map[string]interface{}) (string, error) {
	sql, err := getStringArg(args, "sql")
	if err != nil {
		return "", err
	}

	if !isReadOnlyQuery(sql) {
		return "", fmt.Errorf("only SELECT, WITH (CTE), and SHOW queries are allowed. Use 'execute' for modifications")
	}

	result, err := client.Query(sql)
	if err != nil {
		return "", err
	}

	return formatQueryResultStructured(result), nil
}

// handleExecute runs INSERT, UPDATE, DELETE queries
func handleExecute(client *mysql.Client, args map[string]interface{}) (string, error) {
	sql, err := getStringArg(args, "sql")
	if err != nil {
		return "", err
	}

	confirmKey := getOptionalString(args, "confirm_key", "")

	result, err := client.Execute(sql, confirmKey)
	if err != nil {
		return "", err
	}

	return result.Message, nil
}

// handleTables lists all tables
func handleTables(client *mysql.Client) (string, error) {
	tables, err := client.ListTables()
	if err != nil {
		return "", err
	}

	if len(tables) == 0 {
		return "No tables found in the database.", nil
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

	return sb.String(), nil
}

// handleDescribe shows table structure
func handleDescribe(client *mysql.Client, args map[string]interface{}) (string, error) {
	table, err := getStringArg(args, "table")
	if err != nil {
		return "", err
	}

	columns, err := client.DescribeTable(table)
	if err != nil {
		return "", err
	}

	if len(columns) == 0 {
		return fmt.Sprintf("Table '%s' not found or has no columns.", table), nil
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

	return sb.String(), nil
}

// handleViews lists all views
func handleViews(client *mysql.Client) (string, error) {
	result, err := client.Query(`
		SELECT TABLE_NAME as view_name, VIEW_DEFINITION as definition
		FROM INFORMATION_SCHEMA.VIEWS
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME
	`)
	if err != nil {
		return "", err
	}

	if result.RowCount == 0 {
		return "No views found in the database.", nil
	}

	return formatQueryResultStructured(result), nil
}

// handleIndexes shows indexes for a table
func handleIndexes(client *mysql.Client, args map[string]interface{}) (string, error) {
	table, err := getStringArg(args, "table")
	if err != nil {
		return "", err
	}

	// Use prepared statement for safety
	result, err := client.QueryPrepared(`
		SELECT
			INDEX_NAME,
			COLUMN_NAME,
			NON_UNIQUE,
			SEQ_IN_INDEX,
			CARDINALITY
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`, table)
	if err != nil {
		return "", err
	}

	if result.RowCount == 0 {
		return fmt.Sprintf("No indexes found for table '%s'.", table), nil
	}

	return formatQueryResultStructured(result), nil
}

// handleExplain explains query execution plan
func handleExplain(client *mysql.Client, args map[string]interface{}) (string, error) {
	sql, err := getStringArg(args, "sql")
	if err != nil {
		return "", err
	}

	if !isSelectOnly(sql) {
		return "", fmt.Errorf("EXPLAIN only supports SELECT queries")
	}

	result, err := client.Query("EXPLAIN " + sql)
	if err != nil {
		return "", err
	}

	return formatQueryResultStructured(result), nil
}

// handleCount counts rows in a table.
// Filtered counts (with WHERE) intentionally go through the 'query' tool
// instead, so the user-provided WHERE goes through ValidateQuery and the
// stacked-statement detector like any other SELECT.
func handleCount(client *mysql.Client, args map[string]interface{}) (string, error) {
	table, err := getStringArg(args, "table")
	if err != nil {
		return "", err
	}

	safeTable := sanitizeIdentifier(table)
	query := "SELECT COUNT(*) as count FROM " + safeTable

	result, err := client.Query(query)
	if err != nil {
		return "", err
	}

	if result.RowCount > 0 && len(result.Rows) > 0 {
		if count, ok := result.Rows[0]["count"]; ok {
			return fmt.Sprintf("Count: %v rows", count), nil
		}
	}

	return "Count: 0 rows", nil
}

// handleSample gets sample rows from a table
func handleSample(client *mysql.Client, args map[string]interface{}) (string, error) {
	table, err := getStringArg(args, "table")
	if err != nil {
		return "", err
	}

	limit := getIntArgClamped(args, "limit", DefaultLimit, MinLimit, MaxSampleRows)

	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", sanitizeIdentifier(table), limit)
	result, err := client.Query(query)
	if err != nil {
		return "", err
	}

	return formatQueryResultStructured(result), nil
}

// handleDatabaseInfo gets database connection info
func handleDatabaseInfo(client *mysql.Client) (string, error) {
	result, err := client.Query(`
		SELECT
			@@version as version,
			@@version_comment as version_info,
			DATABASE() as current_database,
			USER() as db_user,
			@@hostname as hostname,
			@@port as port
	`)
	if err != nil {
		return "", err
	}

	if result.RowCount == 0 || len(result.Rows) == 0 {
		return "Could not retrieve database information.", nil
	}

	row := result.Rows[0]
	var sb strings.Builder
	sb.WriteString("Database Connection Info:\n\n")
	sb.WriteString(fmt.Sprintf("• Version: %v\n", getMapValue(row, "version")))
	sb.WriteString(fmt.Sprintf("• Info: %v\n", getMapValue(row, "version_info")))
	sb.WriteString(fmt.Sprintf("• Database: %v\n", getMapValue(row, "current_database")))
	sb.WriteString(fmt.Sprintf("• User: %v\n", getMapValue(row, "db_user")))
	sb.WriteString(fmt.Sprintf("• Host: %v\n", getMapValue(row, "hostname")))
	sb.WriteString(fmt.Sprintf("• Port: %v\n", getMapValue(row, "port")))

	return sb.String(), nil
}

// getMapValue safely retrieves a value from a map, returning "N/A" if not found
func getMapValue(m map[string]interface{}, key string) interface{} {
	if v, ok := m[key]; ok && v != nil {
		return v
	}
	return "N/A"
}

// Helper functions

// sanitizeIdentifier ensures a SQL identifier is safe
func sanitizeIdentifier(s string) string {
	// Remove any characters that aren't alphanumeric or underscore
	var result strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result.WriteRune(c)
		}
	}
	return result.String()
}
