---
title: Security
description: How MCP Go MySQL keeps your database safe — and what it deliberately does not do
---

The security model has **two layers**, and only two. Earlier versions added more, but most of those checks duplicated what the database does better, or protected against threats that do not exist in this deployment. The current model is small, honest, and intentional.

## Layer 1 — MySQL grants (primary)

This is the real boundary. The MCP server connects with a dedicated MySQL user, and that user's privileges decide what is actually possible. A user without the `FILE` privilege cannot read `/etc/passwd` no matter how the SQL is phrased. A user without `CREATE USER` cannot create users. A user without `GRANT OPTION` cannot grant anything to anyone.

Get this layer right and most of the rest is paranoia.

:::caution[Never use root]
Do not point the MCP at `root`, or any user with `GRANT OPTION`, `FILE`, `CREATE USER`, or `SUPER`. The classifier (layer 2) is defence-in-depth, not the primary boundary.
:::

### Recommended grants

```sql
-- Dedicated user
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'secure_password';

-- Read-only (most restrictive)
GRANT SELECT, SHOW VIEW ON your_database.* TO 'mcp_user'@'%';

-- Read + write (most common)
GRANT SELECT, INSERT, UPDATE, DELETE, SHOW VIEW
  ON your_database.* TO 'mcp_user'@'%';

-- DDL (only if you actually want Claude to alter the schema)
GRANT CREATE, ALTER, DROP, INDEX ON your_database.* TO 'mcp_user'@'%';

FLUSH PRIVILEGES;
```

## Layer 2 — Verb classifier (defence-in-depth)

Every statement is parsed for its leading SQL verb (after comments are stripped) and matched against a whitelist. Anything not on the whitelist is rejected before it reaches the driver.

This is the layer that protects you when layer 1 is misconfigured — for example, if someone points the MCP at a user with too many privileges by mistake.

### Verb categories

| Category | Verbs | Behaviour |
|----------|-------|-----------|
| **Read-only** | `SELECT`, `WITH`, `SHOW`, `DESCRIBE`, `DESC`, `EXPLAIN`, `USE` | Allowed. |
| **Write (DML)** | `INSERT`, `UPDATE`, `DELETE`, `REPLACE` | Allowed. Statements affecting more than `MAX_SAFE_ROWS` rows require `confirm_key`. |
| **DDL** | `CREATE`, `DROP`, `ALTER`, `TRUNCATE`, `RENAME` | Rejected unless `ALLOW_DDL=true`. |
| **Stored procedures** | `CALL`, `EXEC`, `EXECUTE` | Allowed. |
| **Forbidden** | `GRANT`, `REVOKE`, `SET`, `FLUSH`, `RESET`, `KILL`, `SHUTDOWN`, `LOAD`, `HANDLER`, `INSTALL`, `UNINSTALL`, `LOCK`, `UNLOCK` | **Always rejected**, regardless of any flag. |
| **Unknown** | anything else | Rejected. |

### Why a whitelist, not a blacklist

A blacklist must enumerate every dangerous form (and miss new ones). A whitelist needs only enumerate accepted verbs; everything else is blocked. Looking at the **first verb** is unambiguous — it cannot be confused by the word `DROP` appearing inside a string literal or column name.

### Additional checks on top of the verb

Two clauses can smuggle dangerous behaviour into otherwise-legal verbs, so they get explicit checks:

- **`INTO OUTFILE` / `INTO DUMPFILE`** — a `SELECT ... INTO OUTFILE` writes to the filesystem. These clauses are rejected anywhere they appear.
- **Stacked statements** — a single MCP call must contain only one statement. `SELECT 1; DROP DATABASE foo` is rejected. The detector ignores `;` characters inside string literals or backticked identifiers.

### Row-count gate

A naked `UPDATE users SET x = 1` (no `WHERE`) is *valid SQL*. The classifier passes it. But after the driver executes it, the MCP checks `RowsAffected`. If it exceeds `MAX_SAFE_ROWS` (default 100), the operation is rolled back unless the caller provided a matching `confirm_key`.

This catches the "ups, I forgot the WHERE" case without trying to parse the SQL — which is what the previous version's regex tried to do, and got wrong.

## What the classifier deliberately does **not** do

- It does **not** pattern-match for `SLEEP`, `BENCHMARK`, `EXTRACTVALUE`, etc. These are legitimate SQL functions. The classic "time-based blind injection" threat assumes user input is being concatenated into SQL by an application — that is not what is happening here. The MCP client is the LLM, and it writes whole statements directly. A `SELECT SLEEP(1)` for debugging is fine.
- It does **not** sanitize error messages. The driver/database error message (`unknown column 'foo'`, `table 'x' doesn't exist`) is exactly what the LLM needs to fix its own query. Hiding it just produces worse next attempts.
- It does **not** rate-limit. A local stdio process with one human user driving an LLM cannot saturate a database.

## Configuration

| Variable | Default | What it does |
|----------|---------|--------------|
| `SAFETY_KEY` | `PRODUCTION_CONFIRMED_2025` | Required for writes that affect more than `MAX_SAFE_ROWS` rows. A warning is logged at startup if left at default. |
| `MAX_SAFE_ROWS` | `100` | Threshold above which `confirm_key` is required. |
| `ALLOW_DDL` | `false` | Set to `true` to let DDL through the classifier. |
| `ALLOWED_TABLES` | empty | Comma-separated whitelist applied to the `describe` tool. |

:::tip[Set your own SAFETY_KEY]
The default value is public. For any non-trivial use, override it:

```bash
export SAFETY_KEY=$(openssl rand -hex 16)
```
:::

## Examples

### Allowed

```sql
SELECT * FROM products WHERE category = 'electronics' LIMIT 10
UPDATE orders SET status = 'shipped' WHERE order_id = 12345
WITH t AS (SELECT 1) SELECT * FROM t
SELECT SLEEP(1)              -- legitimate debugging, allowed
```

### Allowed but gated

```sql
-- Affects more than MAX_SAFE_ROWS rows → requires confirm_key
UPDATE products SET discount = 0.1 WHERE category = 'clearance'

-- DELETE without WHERE: also gated by row count, not by syntax
DELETE FROM staging_table
```

### Rejected

```sql
-- Privilege management
GRANT ALL ON *.* TO 'evil'@'%'
CREATE USER 'foo'@'%' IDENTIFIED BY 'bar'
SET PASSWORD FOR 'root'@'localhost' = PASSWORD('x')
FLUSH PRIVILEGES

-- Filesystem
LOAD DATA INFILE '/etc/passwd' INTO TABLE x
SELECT * FROM users INTO OUTFILE '/tmp/data'

-- Stacked
SELECT 1; DROP DATABASE foo

-- DDL (unless ALLOW_DDL=true)
DROP TABLE users
ALTER TABLE users ADD COLUMN x INT

-- Unknown verb
FOOBAR users
```

## Validation

Tests live in `test/security/integration_test.go`. They cover every category of the classifier: allowed verbs (legitimate SQL must pass), forbidden verbs (privilege/filesystem/server), DDL gating, stacked-statement detection, comment-prefixed queries, and unknown verbs.

```bash
go test -v ./test/security/...
go test -bench=. ./test/security/...
```

The `test/security/security_tests.go` file additionally checks `go.mod` / `go.sum` integrity and dependency freshness.

## Vulnerability scanning

```bash
govulncheck ./...
```

Run periodically. Keep Go and the MySQL driver updated.
