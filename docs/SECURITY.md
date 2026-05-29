# Security Best Practices

How MCP Go MySQL keeps your database safe — and what it deliberately does not do.

## Two-layer model

The server has **two security layers** and only two. Earlier versions had more, but most of them duplicated what the database does better, or protected against threats that don't exist in this deployment. The current model is small, honest, and intentional.

### Layer 1 — MySQL grants (primary)

This is the real boundary. The MCP connects with a dedicated MySQL user; that user's privileges decide what is actually possible. A user without `FILE` cannot read `/etc/passwd` no matter how the SQL is phrased. A user without `CREATE USER` cannot create users. Get this layer right and most of the rest is paranoia.

### Layer 2 — Verb classifier (defence in depth)

Every statement is matched against a whitelist of leading SQL verbs. Privilege management, filesystem access, stacked statements, and unknown verbs are rejected before reaching the driver. This is the layer that protects you when layer 1 is misconfigured (root user, too-broad grants).

## The MySQL user

```sql
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'secure_password';

-- Read-only
GRANT SELECT, SHOW VIEW ON your_database.* TO 'mcp_user'@'%';

-- Read + write (typical)
GRANT SELECT, INSERT, UPDATE, DELETE, SHOW VIEW
  ON your_database.* TO 'mcp_user'@'%';

-- DDL (only if Claude should alter the schema)
GRANT CREATE, ALTER, DROP, INDEX ON your_database.* TO 'mcp_user'@'%';

FLUSH PRIVILEGES;
```

**Never** point the MCP at:

- `root`
- a user with `GRANT OPTION`
- a user with `FILE`
- a user with `CREATE USER`
- a user with `SUPER`
- a user with `*.*` privileges

The classifier blocks the obvious abuses (`GRANT`, `LOAD DATA INFILE`, `CREATE USER`, etc.) but the grants are what stops them at the source.

## What the verb classifier accepts and rejects

The full categorisation lives in `internal/client.go`. Summary:

| Category | Verbs | Behaviour |
|----------|-------|-----------|
| Read-only | `SELECT`, `WITH`, `SHOW`, `DESCRIBE`, `DESC`, `EXPLAIN`, `USE` | Allowed. |
| Write (DML) | `INSERT`, `UPDATE`, `DELETE`, `REPLACE` | Allowed. Row-count gate applies. |
| DDL | `CREATE`, `DROP`, `ALTER`, `TRUNCATE`, `RENAME` | Rejected unless `ALLOW_DDL=true`. |
| Stored procedures | `CALL`, `EXEC`, `EXECUTE` | Allowed. |
| Forbidden | `GRANT`, `REVOKE`, `SET`, `FLUSH`, `RESET`, `KILL`, `SHUTDOWN`, `LOAD`, `HANDLER`, `INSTALL`, `UNINSTALL`, `LOCK`, `UNLOCK` | Always rejected. |
| Unknown | anything else | Rejected. |

Plus two extra checks that run on top of the verb:

- **`INTO OUTFILE` / `INTO DUMPFILE`** — rejected anywhere they appear.
- **Stacked statements** — `SELECT 1; DROP DATABASE foo` is rejected. The detector ignores `;` characters inside string literals or backticked identifiers.

## The row-count gate

A naked `UPDATE users SET x = 1` is *valid SQL*. The classifier passes it. The statement is executed inside an explicit transaction. After execution the MCP checks `RowsAffected()`. If it exceeds `MAX_SAFE_ROWS` and no valid `confirm_key` matching `SAFETY_KEY` was provided, the transaction is rolled back before commit, so the changes never become visible.

This is the actual protection against "I forgot the WHERE" on large UPDATE/DELETE.

```
ALLOWED:  UPDATE users SET active=0 WHERE id=42        (1 row)
ALLOWED:  UPDATE users SET active=0 WHERE created_at < '2024-01-01'  (50 rows)
BLOCKED:  UPDATE users SET active=0                    (10000 rows, no key)
ALLOWED:  UPDATE users SET active=0    -- with confirm_key
```

Defaults: `MAX_SAFE_ROWS=100`, `SAFETY_KEY=PRODUCTION_CONFIRMED_2025` (a warning is logged at startup if you leave the default).

```bash
export SAFETY_KEY=$(openssl rand -hex 16)
export MAX_SAFE_ROWS=100
```

## What the classifier deliberately does **not** do

Each of these was tried and removed. Documenting why so it doesn't come back.

- **No regex for `SLEEP`, `BENCHMARK`, `EXTRACTVALUE`, `UPDATEXML`, `WAITFOR DELAY`.** Time-based blind injection assumes user input is being concatenated into SQL by an application. That is not the threat model here — the LLM writes whole statements. A `SELECT SLEEP(1)` for debugging is fine.
- **No regex for "DELETE/UPDATE without WHERE".** The previous regex (`(?i)UPDATE\s+\w+\s+SET\s+.*\s*$`) was buggy: the greedy `.*` matched the WHERE too. Replaced by the row-count gate, which is reliable and uses real semantics.
- **No error sanitization.** Driver/database errors (`unknown column 'foo'`, `table 'x' doesn't exist`) are exactly what the LLM needs to fix its query. Hiding them produces worse next attempts.
- **No rate limiting.** A local stdio process driven by one human running an LLM cannot saturate a database. The token bucket existed for an imaginary multi-tenant scenario.
- **No `IsSafePath` / `IsSafeCommand` helpers.** The MCP does not touch the filesystem or run shell commands. Validators on code paths that don't exist are dead weight.

## Configuration reference

| Variable | Default | Description |
|----------|---------|-------------|
| `MYSQL_HOST` | `localhost` | Server host. |
| `MYSQL_PORT` | `3306` | Server port. |
| `MYSQL_USER` | (required) | Database user. Not root. |
| `MYSQL_PASSWORD` | (required) | Database password. |
| `MYSQL_DATABASE` | (required) | Default schema. |
| `LOG_PATH` | `mysql-mcp.log` | Confined to cwd, temp, or `/var/log`. |
| `ALLOWED_TABLES` | empty | Comma-separated whitelist applied to `describe`. |
| `ALLOW_DDL` | `false` | `true` lets DDL through the classifier. |
| `SAFETY_KEY` | `PRODUCTION_CONFIRMED_2025` | Required for >`MAX_SAFE_ROWS` writes. |
| `MAX_SAFE_ROWS` | `100` | Threshold for `confirm_key`. |

## Auditing

The MCP logs every operation to `LOG_PATH` (default `mysql-mcp.log`):

- Timestamp
- User and database
- Operation type (SELECT / INSERT / UPDATE / DELETE / DDL)
- The statement
- Result and row count
- Duration
- Source (which tool ran the call)

Log file permissions: `0600` (owner-only on Unix; ACLs on Windows). The path is validated to live within cwd, OS temp dir, or `/var/log` — anything else is silently redirected to the default.

```bash
tail -f mysql-mcp.log
grep -i error mysql-mcp.log
```

## Backups

Always back up before running structural or large data operations:

```bash
mysqldump -u root -p your_database > backup_$(date +%Y%m%d).sql
```

The MCP does not back up automatically.

## Vulnerability scanning

```bash
govulncheck ./...
```

Run periodically. Keep Go and the MySQL driver updated. The classifier itself has no external dependencies, so its surface is bounded by what's in `internal/client.go`.

## Reporting a vulnerability

If you find a security issue, please open an issue on GitHub with the `security` label, or email the maintainer privately. Do **not** post exploitation details publicly until a fix is available.

## Validation

Tests live in `test/security/`:

- `integration_test.go` — every category of the verb classifier (allowed, forbidden, DDL-gated, stacked, unknown). Reject categories list explicit rationale strings.
- `security_tests.go` — `go.mod` / `go.sum` integrity, dependency freshness.

```bash
go test -v ./test/security/...
go test -bench=. ./test/security/...
```

Run before any release, and any time you touch `ValidateQuery` or the verb lists.
