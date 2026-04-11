---
title: Available Tools
description: Reference for the 10 database tools provided by MCP Go MySQL
---

MCP Go MySQL exposes 10 tools for database interaction.

## Read Tools

### 1. query - Execute SELECT Queries

**Purpose:** Perform read queries (SELECT) on the database.

**Usage:** "Show me the 10 most recent users"

**Security:** Automatic validation against SQL injection. SELECT queries only.

```sql
SELECT * FROM users ORDER BY created_at DESC LIMIT 10
```

### 2. tables - List Tables

**Purpose:** Get a list of all tables with metadata.

**Usage:** "What tables are in the database?"

**Information:** Name, storage engine, row count, size.

### 3. describe - Describe Structure

**Purpose:** View the detailed structure of a table or view.

**Usage:** "Describe the users table"

**Information:** Columns, data types, keys, indexes, constraints.

### 4. views - List Views

**Purpose:** Show all database views.

**Usage:** "List available views"

**Information:** View name and SQL definition.

### 5. indexes - View Indexes

**Purpose:** Show indexes for a specific table.

**Usage:** "What indexes does the orders table have?"

**Information:** Index name, columns, type, uniqueness.

### 6. count - Count Rows

**Purpose:** Count records with optional conditions.

**Usage:** "Count active users"

```sql
SELECT COUNT(*) FROM users WHERE active = 1
```

### 7. sample - Get Sample Data

**Purpose:** Get sample rows (maximum 100).

**Usage:** "Give me 5 product examples"

**Limit:** Maximum 100 rows for security.

## Write Tools

### 8. execute - Execute INSERT/UPDATE/DELETE

**Purpose:** Execute write operations with confirmation.

**Usage:** "Update order 123 status to 'shipped'"

**Protection:**

- Small operations (≤100 rows): Executed directly
- Large operations (>100 rows): Require confirmation key
- DELETE/UPDATE without WHERE: Automatically blocked

:::caution
Requires confirmation for bulk operations affecting more than 100 rows.
:::

## Analysis Tools

### 9. explain - Analyze Execution Plan

**Purpose:** Analyze how MySQL will execute a query.

**Usage:** "Explain this query: SELECT * FROM orders WHERE user_id = 123"

**Information:** Index usage, join type, rows examined, cost.

```sql
EXPLAIN SELECT * FROM orders WHERE user_id = 123
```

### 10. database_info - Server Information

**Purpose:** Get connection and server information.

**Usage:** "What version of MySQL am I using?"

**Information:**

- MySQL/MariaDB version
- Current database
- Host and port
- Connected user
- Charset and collation

## Usage Examples with Claude

| User says | Claude uses |
|-----------|------------|
| "How many orders do we have today?" | `count` with date condition |
| "Show me the structure of the products table" | `describe` |
| "Update user ID 42 email to new@email.com" | `execute` (small operation, no confirmation) |
| "This query is slow, why?" | `explain` to analyze the plan |

## Blocked Operations

For security, these operations are **always blocked**:

| Operation | Status |
|-----------|--------|
| `DROP DATABASE` / `DROP SCHEMA` | Blocked |
| `TRUNCATE TABLE` | Blocked |
| `DELETE FROM table` (without WHERE) | Blocked |
| `UPDATE table SET` (without WHERE) | Blocked |
| `INTO OUTFILE` / `DUMPFILE` | Blocked |
| `LOAD_FILE` / `LOAD DATA` | Blocked |
