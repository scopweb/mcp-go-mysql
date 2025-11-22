package internal

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	SafetyKey       string
	MaxSafeRows     int
	AllowedTables   []string // Whitelist of allowed tables (empty = all allowed)
	BlockDDL        bool     // Block DDL operations (CREATE, DROP, ALTER)
	BlockDangerous  bool     // Block dangerous operations (TRUNCATE, etc.)
	RequireConfirm  bool     // Require confirmation for large operations
}

// Client represents a secure MySQL database client
type Client struct {
	db            *sql.DB
	config        *DatabaseConfig
	securityConfig *SecurityConfig
	connected     bool
}

// DatabaseConfig holds connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Timeout  time.Duration
}

// QueryResult holds the result of a database query
type QueryResult struct {
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	RowCount int                      `json:"row_count"`
	Message  string                   `json:"message,omitempty"`
}

// TableInfo holds table metadata
type TableInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Engine  string `json:"engine,omitempty"`
	Rows    int64  `json:"rows,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// ColumnInfo holds column metadata
type ColumnInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	Key        string `json:"key,omitempty"`
	Default    string `json:"default,omitempty"`
	Extra      string `json:"extra,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

// Dangerous SQL patterns for security validation
var (
	// DDL patterns
	ddlPatterns = regexp.MustCompile(`(?i)^\s*(CREATE|DROP|ALTER|TRUNCATE|RENAME)\s+`)

	// Dangerous patterns that should always be blocked
	dangerousPatterns = []string{
		"(?i)DROP\\s+DATABASE",
		"(?i)DROP\\s+SCHEMA",
		"(?i)TRUNCATE\\s+TABLE",
		"(?i)DELETE\\s+FROM\\s+\\w+\\s*$", // DELETE without WHERE
		"(?i)UPDATE\\s+\\w+\\s+SET\\s+.*\\s*$", // UPDATE without WHERE
		"(?i)INTO\\s+OUTFILE",
		"(?i)INTO\\s+DUMPFILE",
		"(?i)LOAD\\s+DATA",
		"(?i)LOAD_FILE\\s*\\(",
	}

	// SQL Injection patterns
	sqlInjectionPatterns = []string{
		"(?i)'\\s*(OR|AND)\\s+'",                    // ' OR ' / ' AND '
		"(?i)\"\\s*(OR|AND)\\s+\"",                  // " OR " / " AND "
		"(?i)'\\s*=\\s*'",                           // '='
		"(?i)\\d+\\s*=\\s*\\d+",                     // 1=1
		"(?i)--\\s*$",                               // SQL comment at end
		"(?i);\\s*--",                               // Statement terminator with comment
		"(?i)/\\*.*\\*/",                            // Block comments
		"'#",                                        // MySQL hash comment after quote
		"(?i)UNION\\s+(ALL\\s+)?SELECT",             // UNION injection
		"(?i)SELECT\\s+.*\\s+FROM\\s+INFORMATION_SCHEMA", // Schema enumeration
		"(?i)SLEEP\\s*\\(",                          // Time-based injection
		"(?i)BENCHMARK\\s*\\(",                      // Time-based injection
		"(?i)WAITFOR\\s+DELAY",                      // Time-based injection (MSSQL style)
		"(?i)0x[0-9a-fA-F]+",                        // Hex encoding
		"(?i)CHAR\\s*\\(\\s*\\d+",                   // CHAR() function abuse
		"(?i)CONCAT\\s*\\(",                         // CONCAT for obfuscation
		"(?i)GROUP_CONCAT\\s*\\(",                   // Data extraction
		"(?i)EXTRACTVALUE\\s*\\(",                   // XML extraction
		"(?i)UPDATEXML\\s*\\(",                      // XML injection
		"(?i)INTO\\s+OUTFILE",                       // File write attack
		"(?i)INTO\\s+DUMPFILE",                      // Binary file write attack
		"(?i)LOAD_FILE",                             // File read attack
	}

	compiledDangerousPatterns   []*regexp.Regexp
	compiledInjectionPatterns   []*regexp.Regexp
)

func init() {
	// Compile dangerous patterns
	for _, pattern := range dangerousPatterns {
		compiledDangerousPatterns = append(compiledDangerousPatterns, regexp.MustCompile(pattern))
	}

	// Compile injection patterns
	for _, pattern := range sqlInjectionPatterns {
		compiledInjectionPatterns = append(compiledInjectionPatterns, regexp.MustCompile(pattern))
	}
}

// NewClient creates a new MySQL client with security defaults
func NewClient() *Client {
	config := &DatabaseConfig{
		Host:     getEnvOrDefault("MYSQL_HOST", "localhost"),
		Port:     getEnvOrDefault("MYSQL_PORT", "3306"),
		User:     getEnvOrDefault("MYSQL_USER", ""),
		Password: getEnvOrDefault("MYSQL_PASSWORD", ""),
		Database: getEnvOrDefault("MYSQL_DATABASE", ""),
		Timeout:  30 * time.Second,
	}

	securityConfig := &SecurityConfig{
		SafetyKey:      getEnvOrDefault("SAFETY_KEY", "PRODUCTION_CONFIRMED_2025"),
		MaxSafeRows:    100,
		AllowedTables:  parseAllowedTables(os.Getenv("ALLOWED_TABLES")),
		BlockDDL:       os.Getenv("ALLOW_DDL") != "true",
		BlockDangerous: true,
		RequireConfirm: true,
	}

	return &Client{
		config:         config,
		securityConfig: securityConfig,
		connected:      false,
	}
}

// Connect establishes a secure connection to the MySQL database
func (c *Client) Connect() error {
	if c.connected && c.db != nil {
		return nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=%s&readTimeout=%s&writeTimeout=%s",
		c.config.User,
		c.config.Password,
		c.config.Host,
		c.config.Port,
		c.config.Database,
		c.config.Timeout,
		c.config.Timeout,
		c.config.Timeout,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool for security and performance
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	c.db = db
	c.connected = true
	return nil
}

// Close closes the database connection
func (c *Client) Close() error {
	if c.db != nil {
		c.connected = false
		return c.db.Close()
	}
	return nil
}

// ValidateQuery performs security validation on a SQL query
func (c *Client) ValidateQuery(query string) error {
	query = strings.TrimSpace(query)

	if query == "" {
		return fmt.Errorf("empty query")
	}

	// Check for SQL injection patterns
	for _, pattern := range compiledInjectionPatterns {
		if pattern.MatchString(query) {
			return fmt.Errorf("potential SQL injection detected: query contains suspicious pattern")
		}
	}

	// Check for dangerous patterns
	for _, pattern := range compiledDangerousPatterns {
		if pattern.MatchString(query) {
			return fmt.Errorf("dangerous SQL operation detected and blocked")
		}
	}

	// Check for DDL if blocked
	if c.securityConfig.BlockDDL && ddlPatterns.MatchString(query) {
		return fmt.Errorf("DDL operations are blocked. Set ALLOW_DDL=true to enable")
	}

	return nil
}

// ValidateTableAccess checks if access to a table is allowed
func (c *Client) ValidateTableAccess(tableName string) error {
	if len(c.securityConfig.AllowedTables) == 0 {
		return nil // No whitelist, all tables allowed
	}

	tableName = strings.ToLower(strings.TrimSpace(tableName))
	for _, allowed := range c.securityConfig.AllowedTables {
		if strings.ToLower(allowed) == tableName {
			return nil
		}
	}

	return fmt.Errorf("access to table '%s' is not allowed", tableName)
}

// Query executes a SELECT query with security validation
func (c *Client) Query(query string) (*QueryResult, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	// Security validation
	if err := c.ValidateQuery(query); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	return c.processRows(rows)
}

// QueryPrepared executes a parameterized query (safe from SQL injection)
func (c *Client) QueryPrepared(query string, args ...interface{}) (*QueryResult, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	// Use prepared statement for safety
	stmt, err := c.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	return c.processRows(rows)
}

// Execute runs a non-SELECT query with security validation
func (c *Client) Execute(query string, confirmKey string) (*QueryResult, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	// Security validation
	if err := c.ValidateQuery(query); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	result, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	affected, _ := result.RowsAffected()

	// Check if confirmation is needed for large operations
	if c.securityConfig.RequireConfirm && affected > int64(c.securityConfig.MaxSafeRows) {
		if confirmKey != c.securityConfig.SafetyKey {
			return nil, fmt.Errorf("operation affects %d rows (>%d). Provide safety key to confirm", affected, c.securityConfig.MaxSafeRows)
		}
	}

	return &QueryResult{
		RowCount: int(affected),
		Message:  fmt.Sprintf("Query executed successfully. Rows affected: %d", affected),
	}, nil
}

// ListTablesSimple returns a list of table names
func (c *Client) ListTablesSimple() ([]string, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	query := "SHOW TABLES"
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// ListTables returns detailed table information
func (c *Client) ListTables() ([]TableInfo, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	query := `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			IFNULL(ENGINE, '') as ENGINE,
			IFNULL(TABLE_ROWS, 0) as TABLE_ROWS,
			IFNULL(TABLE_COMMENT, '') as TABLE_COMMENT
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.Type, &t.Engine, &t.Rows, &t.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}

	return tables, rows.Err()
}

// DescribeTable returns column information for a table
func (c *Client) DescribeTable(tableName string) ([]ColumnInfo, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	// Validate table access
	if err := c.ValidateTableAccess(tableName); err != nil {
		return nil, err
	}

	// Validate table name to prevent injection
	if !isValidIdentifier(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	query := `
		SELECT
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE = 'YES' as NULLABLE,
			IFNULL(COLUMN_KEY, '') as COLUMN_KEY,
			IFNULL(COLUMN_DEFAULT, '') as COLUMN_DEFAULT,
			IFNULL(EXTRA, '') as EXTRA,
			IFNULL(COLUMN_COMMENT, '') as COLUMN_COMMENT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`

	rows, err := c.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &col.Key, &col.Default, &col.Extra, &col.Comment); err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	return columns, rows.Err()
}

// processRows converts database rows to QueryResult
func (c *Client) processRows(rows *sql.Rows) (*QueryResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for JSON serialization
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result.Rows = append(result.Rows, row)
	}

	result.RowCount = len(result.Rows)
	return result, rows.Err()
}

// Helper functions

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseAllowedTables(tables string) []string {
	if tables == "" {
		return nil
	}
	parts := strings.Split(tables, ",")
	var result []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}

// isValidIdentifier checks if a string is a valid SQL identifier
func isValidIdentifier(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	// Only allow alphanumeric and underscore, must start with letter or underscore
	validIdentifier := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	return validIdentifier.MatchString(s)
}

// IsSafeSQL checks if a string is safe from SQL injection (for testing)
func IsSafeSQL(input string) bool {
	for _, pattern := range compiledInjectionPatterns {
		if pattern.MatchString(input) {
			return false
		}
	}
	return true
}

// IsSafePath checks if a path is safe from path traversal (for testing)
func IsSafePath(path string) bool {
	// URL decode the path first to catch encoded attacks
	decodedPath := urlDecode(path)
	pathLower := strings.ToLower(decodedPath)
	originalLower := strings.ToLower(path)

	dangerous := []string{
		"../", "..\\",                    // Basic traversal
		"..%2f", "..%5c",                 // URL encoded
		"%2e%2e%2f", "%2e%2e%5c",         // Fully URL encoded
		"%252e", "%255c", "%252f",        // Double URL encoded
		"//", "\\\\",                     // UNC/network paths
		"..%c0%af", "..%c1%9c",           // Overlong UTF-8 encoding
	}

	// Check both original and decoded versions
	for _, pattern := range dangerous {
		if strings.Contains(pathLower, pattern) || strings.Contains(originalLower, pattern) {
			return false
		}
	}

	// Check for absolute paths (after decoding)
	if strings.HasPrefix(decodedPath, "/") || strings.HasPrefix(path, "/") {
		return false
	}
	if len(decodedPath) > 1 && decodedPath[1] == ':' {
		return false
	}
	if len(path) > 1 && path[1] == ':' {
		return false
	}

	return true
}

// urlDecode performs basic URL decoding
func urlDecode(s string) string {
	result := s
	// Common URL encoded characters
	replacements := map[string]string{
		"%2e": ".", "%2E": ".",
		"%2f": "/", "%2F": "/",
		"%5c": "\\", "%5C": "\\",
		"%3a": ":", "%3A": ":",
		"%25": "%", // Decode % itself for double encoding
	}
	for encoded, decoded := range replacements {
		result = strings.ReplaceAll(result, encoded, decoded)
	}
	return result
}

// IsSafeCommand checks if command input is safe from injection
func IsSafeCommand(input string) bool {
	dangerous := []string{";", "|", "&", "`", "$(", "${", "\n", "\r"}

	for _, pattern := range dangerous {
		if strings.Contains(input, pattern) {
			return false
		}
	}
	return true
}
