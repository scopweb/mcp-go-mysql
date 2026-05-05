---
title: Available Tools
description: Reference for the 10 database tools provided by MCP Go MySQL
---

MCP Go MySQL exposes 10 tools. Read tools cover everything you need to inspect a schema and pull data; the single `execute` tool covers writes; `explain` and `database_info` cover analysis and metadata.

## Read Tools

### query — Execute SELECT/WITH/SHOW Queries

**Purpose:** Run any read-only statement.

**Accepted verbs:** `SELECT`, `WITH` (CTEs), `SHOW`, `DESCRIBE`, `EXPLAIN`, `USE`.

**Usage:** "Show me the 10 most recent users."

```sql
SELECT * FROM users ORDER BY created_at DESC LIMIT 10
```

**Filtered counts also go here:**

```sql
SELECT COUNT(*) FROM users WHERE active = 1
```

The `count` tool handles only unfiltered counts; anything with a `WHERE` belongs in `query` so it goes through the verb classifier and stacked-statement detector.

### tables — List Tables

**Purpose:** Get every table in the current schema with metadata.

**Returns:** name, type, storage engine, approximate row count, comment.

**Usage:** "What tables are in the database?"

### describe — Describe Structure

**Purpose:** Show the columns of a table or view.

**Returns:** column name, type, nullability, key, default, extra, comment.

**Usage:** "Describe the users table."

If `ALLOWED_TABLES` is set, this tool will refuse tables outside the whitelist.

### views — List Views

**Purpose:** List all views in the current schema with their definitions.

**Usage:** "List available views."

### indexes — Show Indexes

**Purpose:** Show all indexes for a given table.

**Returns:** index name, column, sequence, uniqueness, cardinality.

**Usage:** "What indexes does the orders table have?"

Internally uses a prepared statement, so the table name cannot smuggle SQL.

### count — Count Rows

**Purpose:** Unfiltered row count of a single table.

**Usage:** "How many rows does the users table have?"

```sql
SELECT COUNT(*) FROM users
```

For filtered counts, use `query` with `SELECT COUNT(*) FROM table WHERE ...`. This is intentional: it routes the user-supplied `WHERE` through the same validation as any other SELECT.

### sample — Get Sample Rows

**Purpose:** First N rows of a table (default 10, max 100).

**Usage:** "Give me 5 example products."

## Write Tool

### execute — Run INSERT/UPDATE/DELETE/REPLACE

**Purpose:** Single tool for all data modifications.

**Usage:** "Update order 123 status to 'shipped'."

```sql
UPDATE orders SET status = 'shipped' WHERE order_id = 123
```

**Row-count gate:**

- Operations affecting **≤ `MAX_SAFE_ROWS`** rows (default 100): execute directly.
- Operations affecting **more** rows: rolled back unless you pass `confirm_key` matching `SAFETY_KEY`.

This catches the "ups, I forgot the `WHERE`" case. The classifier itself does **not** reject `UPDATE`/`DELETE` without `WHERE` — that decision is made on the actual row count, not on the syntax.

:::caution[The MCP only confirms after counting]
A `DELETE FROM huge_table` is sent to the database; the rows are matched (not committed); if the count exceeds `MAX_SAFE_ROWS` the operation fails. There is no "dry run" — the database does the counting. Make sure your user has rollback-capable storage (InnoDB).
:::

## Analysis Tools

### explain — Execution Plan

**Purpose:** Show how MySQL/MariaDB will execute a SELECT.

**Usage:** "Why is this query slow?"

```sql
EXPLAIN SELECT * FROM orders WHERE user_id = 123
```

**Returns:** join type, possible keys, key used, rows examined, extra info.

`explain` only accepts SELECT statements.

### database_info — Server Metadata

**Purpose:** Connection and server information.

**Usage:** "What MySQL version am I connected to?"

**Returns:** server version, version comment, current database, current user, hostname, port.

## Usage Examples with Claude

| User says | Claude uses |
|-----------|-------------|
| "How many orders do we have today?" | `query` with `SELECT COUNT(*) ... WHERE date = CURDATE()` |
| "Show me the structure of the products table" | `describe` |
| "Update user 42 email to new@email.com" | `execute` (single row, no confirmation) |
| "Set all clearance products to 10% off" | `execute` — if it touches >100 rows, asks for `confirm_key` |
| "This query is slow, why?" | `explain` |
| "What database am I connected to?" | `database_info` |

## What Gets Rejected

The verb classifier runs on every statement before it reaches the driver. See the [Security page](/security/overview/) for the full categorisation. Quick summary:

| Category | Examples | Why |
|----------|----------|-----|
| **Forbidden verbs** | `GRANT`, `REVOKE`, `SET`, `FLUSH`, `RESET`, `KILL`, `SHUTDOWN`, `LOAD`, `HANDLER`, `INSTALL`, `LOCK` | Privilege management, filesystem access, server control. Always rejected. |
| **Filesystem clauses** | `... INTO OUTFILE '/tmp/x'`, `... INTO DUMPFILE '/tmp/x'` | Filesystem write. Always rejected. |
| **Stacked statements** | `SELECT 1; DROP DATABASE foo` | Multiple statements in one call. Rejected. |
| **DDL** | `CREATE`, `DROP`, `ALTER`, `TRUNCATE`, `RENAME` | Rejected unless `ALLOW_DDL=true`. |
| **Unknown verb** | `FOOBAR users` | Whitelist-only. Rejected. |

What is **not** in this list (intentionally): `SELECT SLEEP(1)`, `SELECT BENCHMARK(...)`, `SELECT EXTRACTVALUE(...)`. These are legitimate SQL functions and the classifier no longer special-cases them.
