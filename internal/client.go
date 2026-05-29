package internal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// SecurityConfig holds security-related configuration.
//
// Security model (two layers):
//  1. PRIMARY: MySQL user grants. The dedicated user should have only the
//     minimum privileges needed (typically SELECT/INSERT/UPDATE/DELETE on a
//     specific schema, never FILE, never PROCESS, never CREATE USER/GRANT).
//  2. SECONDARY (this layer): a verb-based statement classifier that blocks
//     statements which a too-permissive MySQL user would otherwise accept
//     (GRANT, CREATE USER, LOAD DATA INFILE, stacked statements, ...),
//     plus a row-count threshold (MaxSafeRows + SafetyKey) to catch
//     accidental UPDATE/DELETE without WHERE.
type SecurityConfig struct {
	SafetyKey      string
	MaxSafeRows    int
	AllowedTables  []string // Whitelist of allowed tables (empty = all allowed)
	BlockDDL       bool     // Block DDL operations (CREATE, DROP, ALTER, TRUNCATE, RENAME)
	RequireConfirm bool     // Require confirmation for large operations
}

// Client represents a secure MySQL/MariaDB database client.
// Carries the active *sql.DB plus the policy/config bundles that
// gate statements before they reach the driver.
type Client struct {
	db             *sql.DB
	config         *DatabaseConfig
	securityConfig *SecurityConfig
	compatConfig   *DBCompatibilityConfig
	timeoutConfig  *TimeoutConfig
	detectedDBType DatabaseType
	connected      bool
}

// DatabaseConfig holds connection configuration for the MySQL/MariaDB driver.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Timeout  time.Duration
	DBType   DatabaseType // Explicit database type (mysql or mariadb)
}

// QueryResult holds the result of a database query (rows + metadata).
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

// Statement classifier — verb-based whitelist.
//
// Why classifier and not regex blacklist:
//   - Blacklists must enumerate every dangerous form (and miss new ones).
//   - Whitelists need only enumerate accepted verbs; everything else is blocked.
//   - Looking at the FIRST verb is unambiguous after StripComments has run,
//     and cannot be confused by the word "DROP" appearing inside a string or
//     a column name.
//
// The categories below are mutually exclusive. Any statement whose leading
// verb does not appear in any category is rejected as "unknown verb".
var (
	// Read-only verbs (always allowed when the user has SELECT grant).
	readOnlyVerbs = []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "USE"}

	// Write verbs (DML) — allowed but subject to MaxSafeRows confirmation.
	writeVerbs = []string{"INSERT", "UPDATE", "DELETE", "REPLACE"}

	// DDL verbs — schema mutation. Allowed only when ALLOW_DDL=true.
	ddlVerbs = []string{"CREATE", "DROP", "ALTER", "TRUNCATE", "RENAME"}

	// CALL verbs (stored procedures) — treated as write by default since
	// procedures may modify data.
	callVerbs = []string{"CALL", "EXEC", "EXECUTE"}

	// Forbidden verbs — privilege management and filesystem access.
	// These are ALWAYS blocked, regardless of ALLOW_DDL or any other flag,
	// because they are never legitimate uses of an MCP database client and
	// they are exactly the operations that abuse a too-permissive MySQL user.
	forbiddenVerbs = []string{
		"GRANT", "REVOKE",     // privilege management
		"SET",                 // SET PASSWORD, SET GLOBAL var, SET ROLE, ...
		"FLUSH",               // FLUSH PRIVILEGES, FLUSH HOSTS, ...
		"RESET",               // RESET MASTER, RESET SLAVE, ...
		"KILL",                // KILL [QUERY|CONNECTION] thread_id
		"SHUTDOWN",            // server shutdown
		"LOAD",                // LOAD DATA INFILE — filesystem read
		"HANDLER",             // direct B-tree access, bypasses many checks
		"INSTALL", "UNINSTALL", // INSTALL PLUGIN — code execution surface
		"LOCK", "UNLOCK",      // table locks
	}

	// Multi-statement separator outside of strings — used to detect stacked
	// queries like "SELECT 1; DROP DATABASE foo".
	stackedStmtSeparator = ';'
)

// firstVerb returns the leading SQL verb of a statement, uppercased.
// Assumes comments and surrounding whitespace have already been stripped.
func firstVerb(query string) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}
	// Take the first whitespace-separated token.
	for i, r := range trimmed {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '(' {
			return strings.ToUpper(trimmed[:i])
		}
	}
	return strings.ToUpper(trimmed)
}

// hasStackedStatements reports whether the query contains a ';' separator
// outside of single- or double-quoted strings (after a non-empty statement).
// A single trailing ';' is allowed.
func hasStackedStatements(query string) bool {
	inSingle, inDouble, inBacktick := false, false, false
	var prev rune
	seenContent := false
	seenSepAfterContent := false
	for _, r := range query {
		switch {
		case r == '\'' && prev != '\\' && !inDouble && !inBacktick:
			inSingle = !inSingle
		case r == '"' && prev != '\\' && !inSingle && !inBacktick:
			inDouble = !inDouble
		case r == '`' && !inSingle && !inDouble:
			inBacktick = !inBacktick
		case r == stackedStmtSeparator && !inSingle && !inDouble && !inBacktick:
			if seenSepAfterContent {
				return true // second ';' found
			}
			if seenContent {
				seenSepAfterContent = true
			}
		default:
			if seenSepAfterContent && r != ' ' && r != '\t' && r != '\n' && r != '\r' {
				// Non-whitespace after a ';' means a second statement is starting.
				return true
			}
			if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
				seenContent = true
			}
		}
		prev = r
	}
	return false
}

// containsAny reports whether s contains any of the given substrings (case-insensitive).
func containsAny(s string, needles []string) bool {
	upper := strings.ToUpper(s)
	for _, n := range needles {
		if strings.Contains(upper, n) {
			return true
		}
	}
	return false
}

// containsVerb reports whether the given verb appears in the slice.
func containsVerb(verb string, list []string) bool {
	for _, v := range list {
		if v == verb {
			return true
		}
	}
	return false
}

// NewClient creates a new MySQL client with security defaults
func NewClient() *Client {
	// Get database type from environment (default: MariaDB)
	dbType := GetDBTypeFromEnv()
	compatConfig := GetDBCompatibilityConfig(string(dbType))

	config := &DatabaseConfig{
		Host:     getEnvOrDefault("MYSQL_HOST", "localhost"),
		Port:     getEnvOrDefault("MYSQL_PORT", "3306"),
		User:     getEnvOrDefault("MYSQL_USER", ""),
		Password: getEnvOrDefault("MYSQL_PASSWORD", ""),
		Database: getEnvOrDefault("MYSQL_DATABASE", ""),
		Timeout:  30 * time.Second,
		DBType:   dbType,
	}

	// Security configuration with warning for default key
	safetyKey := getEnvOrDefault("SAFETY_KEY", "PRODUCTION_CONFIRMED_2025")
	if os.Getenv("SAFETY_KEY") == "" {
		log.Printf("WARNING: Using default SAFETY_KEY. Set SAFETY_KEY env var for production!")
	}

	securityConfig := &SecurityConfig{
		SafetyKey:      safetyKey,
		MaxSafeRows:    getEnvIntOrDefault("MAX_SAFE_ROWS", 100),
		AllowedTables:  parseAllowedTables(os.Getenv("ALLOWED_TABLES")),
		BlockDDL:       os.Getenv("ALLOW_DDL") != "true",
		RequireConfirm: true,
	}

	timeoutConfig := NewTimeoutConfig()

	client := &Client{
		config:         config,
		securityConfig: securityConfig,
		compatConfig:   compatConfig,
		timeoutConfig:  timeoutConfig,
		connected:      false,
	}

	// Log database type information
	log.Printf("Using database: %s (EOL: %s, Support: %s)",
		compatConfig.DisplayName, compatConfig.EOLDate, compatConfig.SupportDuration)

	return client
}

// Connect establishes a secure connection to the MySQL/MariaDB database
func (c *Client) Connect() error {
	if c.connected && c.db != nil {
		return nil
	}

	// Use database-specific DSN generation
	dsn := GetDSNByType(c.config.DBType,
		c.config.User,
		c.config.Password,
		c.config.Host,
		c.config.Port,
		c.config.Database)

	// Add timeout parameters
	if !strings.Contains(dsn, "?") {
		dsn += "?"
	} else {
		dsn += "&"
	}
	dsn += fmt.Sprintf("timeout=%s&readTimeout=%s&writeTimeout=%s",
		c.config.Timeout, c.config.Timeout, c.config.Timeout)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool for security and performance
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute)

	// Test connection with timeout configuration
	ctx, cancel := c.timeoutConfig.TimeoutContext(context.Background(), ProfileConnection)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Detect actual database type
	detectedType, version, err := DetectDatabaseType(db)
	if err == nil {
		c.detectedDBType = detectedType
		log.Printf("Connected to: %s", version)
		// Update compatibility config if detected type differs from configured
		if detectedType != c.config.DBType {
			log.Printf("Database type mismatch: detected=%s, configured=%s", detectedType, c.config.DBType)
			c.compatConfig = GetDBCompatibilityConfig(string(detectedType))
		}
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

// ValidateQuery performs security validation on a SQL statement using the
// verb-based classifier described on SecurityConfig.
//
// Validation order (each step short-circuits):
//  1. Reject empty queries.
//  2. Reject stacked statements ("SELECT 1; DROP DATABASE x").
//  3. Strip comments and identify the leading verb.
//  4. Reject forbidden verbs (GRANT, SET, FLUSH, LOAD, ...) regardless of any flag.
//  5. Reject DDL verbs unless BlockDDL is false.
//  6. Reject INTO OUTFILE / INTO DUMPFILE clauses inside otherwise-legal SELECTs.
//  7. Reject unknown verbs (whitelist by default).
//
// Note: this function does NOT enforce row-count thresholds. The threshold
// (MaxSafeRows + SafetyKey) is enforced inside Execute() using an explicit
// transaction so that large unconfirmed writes are rolled back and never committed.
func (c *Client) ValidateQuery(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("empty query")
	}

	// Stacked-statement detection runs on the raw query (before stripping
	// comments) so a comment cannot mask an extra ';'.
	if hasStackedStatements(query) {
		return fmt.Errorf("multiple statements are not allowed in a single call")
	}

	stripped := StripComments(query)
	verb := firstVerb(stripped)

	if verb == "" {
		return fmt.Errorf("empty query after comment stripping")
	}

	if containsVerb(verb, forbiddenVerbs) {
		return fmt.Errorf("statement %q is not allowed (privilege management or filesystem access)", verb)
	}

	if containsVerb(verb, ddlVerbs) {
		if c.securityConfig.BlockDDL {
			return fmt.Errorf("DDL operations are blocked. Set ALLOW_DDL=true to enable")
		}
		return nil
	}

	// SELECT/INSERT can still smuggle filesystem access through INTO OUTFILE /
	// INTO DUMPFILE. These are MySQL-specific clauses, not separate verbs, so
	// the classifier alone does not catch them.
	if containsAny(stripped, []string{"INTO OUTFILE", "INTO DUMPFILE"}) {
		return fmt.Errorf("INTO OUTFILE / INTO DUMPFILE clauses are not allowed")
	}

	if containsVerb(verb, readOnlyVerbs) || containsVerb(verb, writeVerbs) || containsVerb(verb, callVerbs) {
		return nil
	}

	return fmt.Errorf("statement starts with unknown verb %q; only SELECT/WITH/SHOW/DESCRIBE/EXPLAIN/USE and INSERT/UPDATE/DELETE/REPLACE are accepted", verb)
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

	// Use timeout configuration for query operations
	ctx, cancel := c.timeoutConfig.TimeoutContext(context.Background(), ProfileQuery)
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

	// Use timeout configuration for query operations
	ctx, cancel := c.timeoutConfig.TimeoutContext(context.Background(), ProfileQuery)
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

// Execute runs a non-SELECT query with security validation.
//
// Large writes (more than MaxSafeRows rows affected) require a valid confirmKey.
// The operation is executed inside an explicit transaction. If the row threshold
// is exceeded and no valid confirmKey is provided, the transaction is rolled back
// so the changes are never committed.
func (c *Client) Execute(query string, confirmKey string) (*QueryResult, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	// Security validation
	if err := c.ValidateQuery(query); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Use timeout configuration for write operations
	ctx, cancel := c.timeoutConfig.TimeoutContext(context.Background(), ProfileWrite)
	defer cancel()

	// Execute inside an explicit transaction so we can roll back large
	// unconfirmed writes before they become visible. This is the actual
	// implementation of the MAX_SAFE_ROWS safety gate.
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	result, err := tx.ExecContext(ctx, query)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	affected, _ := result.RowsAffected()

	// Row-count safety gate: if the operation touched more rows than allowed,
	// require the safety key. Without it we roll back so no changes persist.
	if c.securityConfig.RequireConfirm && affected > int64(c.securityConfig.MaxSafeRows) {
		if confirmKey != c.securityConfig.SafetyKey {
			tx.Rollback()
			return nil, fmt.Errorf(
				"operation affects %d rows (>%d). Provide safety key to confirm. Changes have been rolled back",
				affected, c.securityConfig.MaxSafeRows,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
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

func getEnvIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
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

// StripComments removes SQL comments from a query so the verb classifier
// (and other helpers) can see the real leading keyword.
//
// Handles three comment styles:
//   - line:  -- ... \n
//   - line:  # ... \n          (MySQL extension)
//   - block: /* ... */
//
// This is a textual scrub, not a full SQL parser; it does not attempt to
// preserve comments that appear inside string literals (it removes them
// anyway, which is harmless for our use because we never execute the
// stripped form — we only inspect its leading verb).
//
// This is the single implementation used both by ValidateQuery and by
// the cmd/sqlcheck helpers.
func StripComments(s string) string {
	// Line comments first.
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if idx := strings.Index(l, "--"); idx >= 0 {
			l = l[:idx]
		}
		if idx := strings.Index(l, "#"); idx >= 0 {
			l = l[:idx]
		}
		lines[i] = l
	}
	out := strings.Join(lines, "\n")

	// Block comments.
	for {
		start := strings.Index(out, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(out[start+2:], "*/")
		if end < 0 {
			break
		}
		out = out[:start] + out[start+2+end+2:]
	}

	// Normalize whitespace so the verb extractor sees a clean prefix.
	out = strings.ReplaceAll(out, "\t", " ")
	out = strings.ReplaceAll(out, "\r", " ")
	return strings.Join(strings.Fields(out), " ")
}
