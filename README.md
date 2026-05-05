# mcp-go-mysql

A Model Context Protocol (MCP) server for MySQL and MariaDB, written in Go.
Lets Claude Desktop (or any MCP client) explore and modify a database through
a small, well-defined set of tools.

## Security model

Two layers — and only two. Any extra "regex of dangerous patterns" was removed
because it traded clarity and false positives for protection that the database
already provides better.

**Layer 1 — MySQL grants (primary).** This is the real boundary. The MCP
connects with a dedicated user; that user's privileges decide what is actually
possible. A user without `FILE` cannot read `/etc/passwd` no matter how the
SQL is phrased. A user without `CREATE USER` cannot create users. Get this
layer right and most of the rest is paranoia.

**Layer 2 — Verb classifier (this code).** A defence-in-depth layer for the
case where layer 1 is misconfigured (root user, too-broad grants, …). Every
statement is classified by its leading SQL verb after comments are stripped.
The classifier:

- **Always rejects** privilege management and filesystem access:
  `GRANT`, `REVOKE`, `SET`, `FLUSH`, `RESET`, `KILL`, `SHUTDOWN`, `LOAD`,
  `HANDLER`, `INSTALL`, `UNINSTALL`, `LOCK`, `UNLOCK`. Also rejects
  `INTO OUTFILE` / `INTO DUMPFILE` clauses inside otherwise-legal SELECTs.
- **Rejects DDL** (`CREATE`, `DROP`, `ALTER`, `TRUNCATE`, `RENAME`) unless
  `ALLOW_DDL=true`.
- **Rejects stacked statements** (`SELECT 1; DROP DATABASE foo`).
- **Rejects unknown verbs** — it's a whitelist, not a blacklist.
- **Allows** `SELECT/WITH/SHOW/DESCRIBE/EXPLAIN/USE` and
  `INSERT/UPDATE/DELETE/REPLACE/CALL`.

Plus: an `UPDATE` or `DELETE` that ends up affecting more than `MAX_SAFE_ROWS`
rows requires `confirm_key` to commit. This catches the "ups, I forgot the
WHERE" case without trying to parse the SQL.

What the classifier deliberately does **not** do:

- It does not pattern-match for `SLEEP`, `BENCHMARK`, `EXTRACTVALUE`, etc.
  These are legitimate SQL functions; the classic "time-based blind injection"
  threat model assumes user input is being concatenated into SQL by an
  application — that's not what's happening here. The MCP client is the LLM,
  and it writes whole statements directly.
- It does not try to detect missing `WHERE` clauses with regex (those checks
  had bugs and false positives). The row-count gate covers the same use case
  more reliably.

## Recommended MySQL user

```sql
-- Create a dedicated user
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'secure_password';

-- Minimum privileges for read-only usage
GRANT SELECT, SHOW VIEW ON your_database.* TO 'mcp_user'@'%';

-- Read + write (most common)
GRANT SELECT, INSERT, UPDATE, DELETE, SHOW VIEW
  ON your_database.* TO 'mcp_user'@'%';

-- DDL (only if you actually want Claude to alter the schema)
GRANT CREATE, ALTER, DROP, INDEX ON your_database.* TO 'mcp_user'@'%';

FLUSH PRIVILEGES;
```

Do **not** use `root` or any user with `GRANT OPTION` / `FILE` /
`CREATE USER` / `SUPER`. The classifier blocks the obvious abuses but the
grants are what stops them at the source.

## Tools

| Tool            | What it does                                                           |
|-----------------|------------------------------------------------------------------------|
| `query`         | Run a SELECT/WITH/SHOW. Read-only.                                     |
| `execute`       | Run INSERT/UPDATE/DELETE. Asks for `confirm_key` past `MAX_SAFE_ROWS`. |
| `tables`        | List tables with metadata.                                             |
| `describe`      | Show columns, types, keys for one table.                               |
| `views`         | List views.                                                            |
| `indexes`       | Show indexes for a table.                                              |
| `explain`       | EXPLAIN a SELECT.                                                      |
| `count`         | `SELECT COUNT(*)` on a table. For filtered counts, use `query`.        |
| `sample`        | First N rows of a table (default 10, max 100).                         |
| `database_info` | Server version, current user, host, port, database.                    |

## Install

```bash
git clone https://github.com/scopweb/mcp-go-mysql.git
cd mcp-go-mysql
go mod tidy
go build -o mysql-mcp ./cmd
go test ./...
```

## Configuration

`.env` in the project directory, or environment variables in the Claude
Desktop config.

| Variable          | Required | Default                       | Notes                                     |
|-------------------|----------|-------------------------------|-------------------------------------------|
| `MYSQL_HOST`      | yes      | `localhost`                   |                                           |
| `MYSQL_PORT`      | no       | `3306`                        |                                           |
| `MYSQL_USER`      | yes      | —                             | The dedicated user, not root.             |
| `MYSQL_PASSWORD`  | yes      | —                             |                                           |
| `MYSQL_DATABASE`  | yes      | —                             | Default schema.                           |
| `LOG_PATH`        | no       | `mysql-mcp.log`               | Confined to cwd, temp, or `/var/log`.     |
| `ALLOWED_TABLES`  | no       | empty (= all tables allowed)  | Comma-separated whitelist for `describe`. |
| `ALLOW_DDL`       | no       | `false`                       | `true` lets DDL through the classifier.   |
| `SAFETY_KEY`      | no       | `PRODUCTION_CONFIRMED_2025`   | Required for >`MAX_SAFE_ROWS` writes.     |
| `MAX_SAFE_ROWS`   | no       | `100`                         |                                           |

A warning is logged at startup if `SAFETY_KEY` is left at its default —
change it for any non-trivial use.

## Claude Desktop

Configuration file:

| OS      | Path                                                            |
|---------|-----------------------------------------------------------------|
| Windows | `%APPDATA%\Claude\claude_desktop_config.json`                   |
| macOS   | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux   | `~/.config/Claude/claude_desktop_config.json`                   |

```json
{
  "mcpServers": {
    "mysql": {
      "command": "C:\\path\\to\\mysql-mcp.exe",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "secure_password",
        "MYSQL_DATABASE": "your_database",
        "SAFETY_KEY": "your-own-key",
        "MAX_SAFE_ROWS": "100"
      }
    }
  }
}
```

Restart Claude Desktop. In a new chat: "What MySQL tools are available?"
should list ten tools.

## Examples

```sql
-- Allowed
SELECT * FROM products WHERE category = 'electronics' LIMIT 10
UPDATE orders SET status = 'shipped' WHERE order_id = 12345
WITH t AS (SELECT 1) SELECT * FROM t

-- Allowed, but >MAX_SAFE_ROWS rows → asks for confirm_key
UPDATE products SET discount = 0.1 WHERE category = 'clearance'

-- Rejected: privilege management
GRANT ALL ON *.* TO 'evil'@'%'
CREATE USER 'foo'@'%' IDENTIFIED BY 'bar'
SET PASSWORD FOR 'root'@'localhost' = PASSWORD('x')
FLUSH PRIVILEGES

-- Rejected: filesystem
LOAD DATA INFILE '/etc/passwd' INTO TABLE x
SELECT * FROM users INTO OUTFILE '/tmp/data'

-- Rejected: stacked
SELECT 1; DROP DATABASE foo

-- Rejected unless ALLOW_DDL=true
DROP TABLE users
ALTER TABLE users ADD COLUMN x INT
```

## Tests

```bash
go test ./...                       # everything
go test -v ./test/security/...      # the verb classifier
go test -bench=. ./test/security/...
```

`test/security/security_tests.go` checks dependency hashes and `go.mod`
integrity. `test/security/integration_test.go` exercises every category of
the verb classifier (allowed, forbidden, DDL-gated, stacked, unknown).

## Project layout

```
cmd/                     MCP protocol layer (stdin/stdout JSON-RPC)
  main.go                Entry point + .env loader + log path validation
  handlers.go            initialize / tools/list / tools/call routing
  tools.go               Tool definitions and dispatch
  format.go              AI-optimized result formatting
  security.go            stripSQLComments helper
  sqlcheck.go            isReadOnlyQuery / isWriteQuery / isDDLQuery
internal/                Database client + policy
  client.go              Connection, classifier, ValidateQuery, helpers
  audit.go               Audit event types (loggers pluggable)
  timeout.go             Per-operation timeout profiles
  db_compat.go           MySQL vs MariaDB detection and tuning
test/security/           Classifier tests + dependency-integrity tests
docs/                    Architecture and security notes
```

## Troubleshooting

| Symptom                                | Likely cause                                          |
|----------------------------------------|-------------------------------------------------------|
| `connection refused`                   | MySQL is not running on the configured host/port.     |
| `access denied`                        | Wrong credentials or grants don't cover the schema.   |
| `statement "X" is not allowed`         | A forbidden verb (GRANT, SET, LOAD, …). Use grants.   |
| `DDL operations are blocked`           | Set `ALLOW_DDL=true` if you really want this.         |
| `multiple statements are not allowed`  | Split your call into separate `query`/`execute` runs. |
| `operation affects N rows (>M)`        | Pass `confirm_key` matching `SAFETY_KEY`.             |

Logs go to `LOG_PATH` (default `mysql-mcp.log` in cwd). `tail -f` it while
debugging.

## License

MIT — see LICENSE.
