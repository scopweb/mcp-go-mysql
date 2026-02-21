# Skill: MySQL MCP Tools — Claude Desktop Usage Guide

## Description

This skill teaches Claude Desktop how to effectively use the 10 MCP database tools provided by the **mcp-go-mysql** server. Follow these patterns and workflows to provide the best database assistance to users.

---

## Available Tools Reference

### Read-Only Tools (query rate: 1000/s)

#### `query`
Execute SELECT, WITH (CTE), and SHOW queries.
```
Parameters: { "sql": "SELECT ..." }
```
- Only SELECT, WITH, and SHOW statements are allowed
- Results return as JSON with columns, rows, and row_count
- Timeout: 30 seconds (standard queries)
- For INSERT/UPDATE/DELETE, tell the user to use `execute` instead

#### `explain`
Analyze query execution plans.
```
Parameters: { "sql": "SELECT ..." }
```
- Only accepts SELECT queries
- Returns MySQL EXPLAIN output
- Use this to help users optimize slow queries

#### `count`
Count rows in a table with optional filtering.
```
Parameters: { "table": "table_name", "where": "status = 'active'" }
```
- `table` is required, `where` is optional
- The WHERE clause is validated for SQL injection
- Returns "Count: N rows"

#### `sample`
Get sample rows from a table.
```
Parameters: { "table": "table_name", "limit": 10 }
```
- Default limit: 10, maximum: 100
- Returns all columns from the table
- Useful for understanding data structure and content

### Metadata Tools (admin rate: 10/s)

#### `tables`
List all tables in the database.
```
Parameters: {} (none required)
```
- Returns: table name, type, engine, approximate row count, comments
- Always start database exploration with this tool

#### `describe`
Show column details for a specific table.
```
Parameters: { "table": "table_name" }
```
- Returns: column name, type, nullable, key, default, extra, comment
- Subject to table whitelist if ALLOWED_TABLES is configured

#### `views`
List all database views.
```
Parameters: {} (none required)
```
- Returns view names and definitions
- "No views found" if database has no views

#### `indexes`
Show indexes for a specific table.
```
Parameters: { "table": "table_name" }
```
- Returns: index name, column, uniqueness, sequence, cardinality
- Uses prepared statements (safe from injection)

#### `database_info`
Show server connection and version information.
```
Parameters: {} (none required)
```
- Returns: MySQL version, database name, user, hostname, port
- Useful for debugging connection issues

### Write Tools (write rate: 100/s)

#### `execute`
Execute INSERT, UPDATE, or DELETE statements.
```
Parameters: { "sql": "INSERT INTO ...", "confirm_key": "PRODUCTION_CONFIRMED_2025" }
```
- Only INSERT, UPDATE, DELETE allowed (not SELECT, not DDL)
- If operation affects >100 rows, the `confirm_key` parameter is required
- DDL operations (CREATE, ALTER, DROP) are blocked by default
- DROP DATABASE/SCHEMA is ALWAYS blocked

---

## Recommended Workflows

### Workflow 1: Database Exploration (most common)

When a user asks about their database or wants to understand its structure:

```
Step 1: tables          → Get overview of all tables
Step 2: describe        → Examine specific tables of interest
Step 3: sample          → See actual data examples
Step 4: count           → Understand data volumes
```

**Example conversation:**
- User: "What's in my database?"
- You: Use `tables` first, then `describe` on interesting tables, then `sample` to show examples

### Workflow 2: Data Analysis

When a user asks a question that requires querying data:

```
Step 1: describe        → Understand column names and types
Step 2: count           → Check data volume before querying
Step 3: query           → Execute the SELECT query
Step 4: explain         → Optimize if query is slow (optional)
```

**Important:** Always check the table structure with `describe` before writing a query. This prevents errors from wrong column names or types.

### Workflow 3: Data Modification

When a user wants to insert, update, or delete data:

```
Step 1: describe        → Verify table structure and constraints
Step 2: count           → Estimate affected rows
Step 3: query           → Preview data that will be affected (SELECT first)
Step 4: execute         → Run the modification
```

**Critical safety rules:**
- ALWAYS preview with a SELECT query before executing modifications
- Tell the user how many rows will be affected
- If >100 rows affected, inform the user that a confirmation key is needed
- The default confirmation key is `PRODUCTION_CONFIRMED_2025` (may be custom)
- NEVER execute UPDATE or DELETE without a WHERE clause

### Workflow 4: Query Optimization

When a user asks about performance or slow queries:

```
Step 1: explain         → Get the execution plan
Step 2: indexes         → Check existing indexes
Step 3: describe        → Review column types and keys
Step 4: query           → Test the optimized query
```

---

## Tool Usage Best Practices

### DO

1. **Start with metadata tools** — Use `tables`, `describe`, and `sample` to understand the schema before writing complex queries
2. **Use `count` before large queries** — Check data volumes to set appropriate expectations
3. **Preview before modifying** — Always use `query` (SELECT) to preview rows before using `execute`
4. **Show results clearly** — Format query results as readable tables or summaries for the user
5. **Use `explain` proactively** — When building complex queries, check the execution plan
6. **Be specific with columns** — Use `SELECT col1, col2` instead of `SELECT *` when the user only needs certain columns
7. **Apply LIMIT on large tables** — Add `LIMIT` to queries when the full result isn't needed

### DON'T

1. **Don't guess column names** — Always `describe` first to verify exact names and types
2. **Don't use `query` for writes** — It only allows SELECT; use `execute` for modifications
3. **Don't execute DDL through `execute`** — DDL is blocked unless ALLOW_DDL=true
4. **Don't ignore the confirmation system** — Operations on >100 rows need the safety key
5. **Don't run UPDATE/DELETE without WHERE** — These are blocked by the security layer
6. **Don't query INFORMATION_SCHEMA directly** — Use the metadata tools (`tables`, `describe`, `indexes`, `views`) instead; direct INFORMATION_SCHEMA queries are blocked by security validation
7. **Don't use SQL comments in queries** — Comments (`--`, `#`, `/* */`) are detected as potential injection and will be blocked
8. **Don't use UNION SELECT** — Blocked by SQL injection protection patterns
9. **Don't use SLEEP(), BENCHMARK(), or timing functions** — Blocked as time-based injection patterns
10. **Don't concatenate user-provided strings directly into SQL** — Use proper value quoting

---

## Handling Errors

### "Security validation failed"
The query matched a SQL injection pattern. Rephrase the query:
- Avoid `' OR '1'='1'` patterns
- Avoid `UNION SELECT`
- Don't use `--` or `#` comments
- Don't use `SLEEP()`, `BENCHMARK()`, `CHAR()`, `CONCAT()`
- Don't reference `INFORMATION_SCHEMA` directly — use metadata tools

### "Rate limit exceeded"
Too many operations in a short period. Wait and retry:
- Query operations: up to 1,000/second
- Write operations: up to 100/second
- Admin operations: up to 10/second
- Wait a moment before retrying

### "Access to table 'X' is not allowed"
The table whitelist (ALLOWED_TABLES) doesn't include this table. Inform the user which tables are accessible using the `tables` tool.

### "DDL operations are blocked"
CREATE/ALTER/DROP/TRUNCATE are disabled. Inform the user that the server is configured with ALLOW_DDL=false.

### "Operation affects N rows. Provide safety key"
A write operation affects more than 100 rows. Tell the user:
- How many rows will be affected
- That they need to provide the confirmation key
- Use `confirm_key` parameter with the safety key

### "Only SELECT queries allowed"
User tried a write query through the `query` tool. Redirect them to use `execute` instead.

---

## SQL Syntax Considerations

### Safe Query Patterns
```sql
-- Simple select (safe)
SELECT id, name, email FROM users WHERE status = 'active'

-- Aggregation (safe)
SELECT department, COUNT(*) as total FROM employees GROUP BY department

-- JOINs (safe)
SELECT o.id, u.name FROM orders o JOIN users u ON o.user_id = u.id

-- Subqueries (safe)
SELECT * FROM products WHERE price > (SELECT AVG(price) FROM products)

-- CTEs (safe — use WITH)
WITH active_users AS (SELECT * FROM users WHERE status = 'active')
SELECT * FROM active_users WHERE created_at > '2025-01-01'

-- SHOW commands (safe)
SHOW VARIABLES LIKE 'max_connections'
```

### Patterns That Will Be Blocked
```sql
-- UNION injection (blocked)
SELECT * FROM users UNION SELECT * FROM passwords

-- Comments (blocked)
SELECT * FROM users -- this is a comment

-- INFORMATION_SCHEMA (blocked — use metadata tools)
SELECT * FROM INFORMATION_SCHEMA.TABLES

-- Timing functions (blocked)
SELECT SLEEP(5)
SELECT BENCHMARK(1000000, SHA1('test'))

-- File operations (blocked)
SELECT LOAD_FILE('/etc/passwd')
SELECT * INTO OUTFILE '/tmp/data.csv' FROM users

-- DELETE/UPDATE without WHERE (blocked)
DELETE FROM users
UPDATE users SET status = 'inactive'
```

---

## Response Formatting Guidelines

### For table listings
Present tables as a structured list with relevant metadata:
- Table name, engine, approximate row count
- Group by purpose if recognizable

### For query results
- Small results (< 10 rows): Show as a formatted table
- Medium results (10-50 rows): Show a summary and key patterns
- Large results (50+ rows): Summarize findings and show representative examples

### For table descriptions
Present columns grouped by purpose:
- Primary keys first
- Foreign keys next
- Data columns grouped logically
- Timestamps last

### For counts
Provide context:
- "There are 1,234 users total, of which 987 are active"
- Compare with related tables when relevant

---

## Multi-Database Environments

If the server is configured with multiple database connections (e.g., `mysql-dev` and `mysql-prod`), tools are prefixed by server name. Always clarify with the user which database they intend to operate on before running queries, especially write operations.

---

## Timeouts

Be aware of operation timeouts:
- **Queries**: 30 seconds
- **Long queries**: 5 minutes (complex aggregations)
- **Write operations**: 60 seconds
- **Admin operations**: 15 seconds
- **Connection**: 5 seconds

If a query is timing out, suggest:
1. Adding indexes (check with `indexes` tool first)
2. Adding WHERE clauses to narrow results
3. Using LIMIT to reduce result size
4. Breaking complex queries into smaller steps
