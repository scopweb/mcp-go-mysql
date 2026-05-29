# MCP Go MySQL — Architecture

How the server is wired internally. For *what* it does see the README; for *how it stays safe* see `SECURITY.md`.

## Overview

```
┌─────────────────┐     MCP / JSON-RPC 2.0     ┌──────────────────┐
│  Claude Desktop │ ◄───────────────────────► │  MCP Go MySQL    │
│   (or any MCP   │     stdin / stdout         │     Server       │
│      client)    │                            │                  │
└─────────────────┘                            └────────┬─────────┘
                                                        │
                                              ValidateQuery()
                                              (verb classifier)
                                                        │
                                              ┌─────────▼────────┐
                                              │  MySQL / MariaDB │
                                              │  (real boundary  │
                                              │   = user grants) │
                                              └──────────────────┘
```

The server is a single Go binary that speaks JSON-RPC 2.0 over stdin/stdout. Every tool call goes through the same path: parse → dispatch to a handler in `cmd/tools.go` → handler calls a method on `internal.Client` → that method calls `ValidateQuery` → the SQL goes to the driver → the result is formatted and returned.

## Components

### Command layer (`cmd/`)

The MCP protocol surface. Knows nothing about MySQL specifically.

- **`main.go`** — entry point. Reads stdin line by line, decodes JSON-RPC, dispatches to `handleMessage`, encodes the response back to stdout. Loads `.env`, opens the log file (with path confined to cwd / temp / `/var/log`), creates the `internal.Client`.
- **`types.go`** — `MCPMessage`, `MCPError`, `ToolResponse`, `ContentItem`. JSON-RPC 2.0 wire format.
- **`handlers.go`** — handles `initialize`, `tools/list`, `tools/call`, and `notifications/initialized`. The `initialize` response carries the `instructions` string the LLM sees on connect.
- **`tools.go`** — the ten tool definitions plus their handlers (`handleQuery`, `handleExecute`, `handleTables`, `handleDescribe`, `handleViews`, `handleIndexes`, `handleExplain`, `handleCount`, `handleSample`, `handleDatabaseInfo`). `callClientMethod` routes a tool name to its handler. No rate limiting — handlers go straight to the client.
- **`format.go`** — formats `QueryResult`, table lists, and table descriptions for AI consumption (compact mode).
- **`security.go`** — removed. The duplicate `stripSQLComments` was unified into `internal.StripComments`, which is now the single source of truth used by both `ValidateQuery` and the helpers in `sqlcheck.go`.
- **`sqlcheck.go`** — `isReadOnlyQuery`, `isWriteQuery`, `isDDLQuery`, `isSelectOnly`. Used by handlers to gate which tool can run what (`query` accepts only read-only; `explain` accepts only SELECT; `execute` accepts only write).
- **`params.go`** — argument-extraction helpers (`getStringArg`, `getOptionalString`, `getIntArgClamped`).

### Internal layer (`internal/`)

The database client and the policy that gates statements before they reach the driver.

- **`client.go`** — the heart. Defines:
  - `SecurityConfig` — `SafetyKey`, `MaxSafeRows`, `AllowedTables`, `BlockDDL`, `RequireConfirm`. No more `BlockDangerous` flag (was a no-op, always `true`).
  - `Client` — holds `*sql.DB`, the configs, and the detected database type. No `rateLimiter`, no `errorSanitizer` fields.
  - The verb-classifier data: `readOnlyVerbs`, `writeVerbs`, `ddlVerbs`, `callVerbs`, `forbiddenVerbs`.
  - The classifier helpers: `firstVerb`, `hasStackedStatements`, `containsAny`, `containsVerb`.
  - `stripComments` — removes `--`, `#`, `/* ... */` comments and normalises whitespace before the verb is read.
  - `ValidateQuery` — the seven-step gate (see `SECURITY.md`).
  - The query path: `Query`, `QueryPrepared`, `Execute`, `ListTables`, `DescribeTable`, etc.
- **`audit.go`** — removed during cleanup. The sophisticated audit event system was never integrated into `Query`/`Execute` paths and only existed in test code for features that were later removed. Keeping it would have been unnecessary bloat.
- **`timeout.go`** — `TimeoutConfig` with per-profile timeouts (query 30s, long-query 5m, write 60s, admin 15s, connection 5s). `ValidateQuery` doesn't use it; the query methods do.
- **`db_compat.go`** — detects MySQL vs MariaDB at connect time and returns version-specific compatibility flags.

### Test layer (`test/security/`)

- **`integration_test.go`** — exercises every category of the verb classifier: allowed verbs (legitimate SQL must pass), forbidden verbs (privilege/filesystem/server), DDL gating, stacked-statement detection, comment-prefixed queries, unknown verbs. Includes `BenchmarkValidateQuery`.
- **`security_tests.go`** — `go.mod`/`go.sum` integrity, dependency freshness check via `go list -u`. Useful for catching outdated drivers.

## Security architecture

```
┌──────────────────────────────────────────────────────────────┐
│ Layer 1: MySQL grants (PRIMARY — outside this codebase)      │
│   - SELECT/INSERT/UPDATE/DELETE on a specific schema         │
│   - never FILE, never PROCESS, never CREATE USER, never SUPER│
│   - this is the real boundary                                │
└──────────────────────────────────────────────────────────────┘
                              ▲
                              │ everything that gets past
                              │ Layer 2 still has to satisfy
                              │ MySQL's grants
                              │
┌──────────────────────────────────────────────────────────────┐
│ Layer 2: Verb classifier (defence in depth — ValidateQuery)  │
│                                                              │
│  1. Reject empty                                             │
│  2. Reject stacked (";" outside strings)                     │
│  3. Strip comments → extract first verb                      │
│  4. Reject forbidden verbs (GRANT/SET/FLUSH/LOAD/...)        │
│  5. Reject DDL unless ALLOW_DDL=true                         │
│  6. Reject INTO OUTFILE / INTO DUMPFILE                      │
│  7. Reject unknown verbs (whitelist)                         │
│  8. Allow read-only / write / call verbs                     │
└──────────────────────────────────────────────────────────────┘
                              ▲
                              │
┌──────────────────────────────────────────────────────────────┐
│ Layer 3 (post-execution): Row-count gate                     │
│   - Execute() checks RowsAffected after the driver runs      │
│   - if > MaxSafeRows AND no confirm_key: return error        │
│   - covers UPDATE/DELETE without WHERE without parsing SQL   │
└──────────────────────────────────────────────────────────────┘
```

What was removed compared to the previous version:

- **Regex-based "23+ patterns" injection list** — replaced by the verb whitelist.
- **Regex-based "dangerous operations"** — replaced by the verb whitelist + filesystem-clause checks.
- **Token-bucket rate limiter** — irrelevant for a single-user stdio process.
- **Error sanitizer** — counterproductive; the LLM needs real error text to self-correct.
- **`IsSafePath` / `IsSafeCommand`** — for code paths that don't exist in this MCP.

See `CHANGELOG.md` 3.0.0 for the full rationale.

## Data flow

### Read (SELECT)

```
Claude Desktop
   │ tools/call name=query, arguments={sql: "SELECT ..."}
   ▼
cmd/handlers.go: handleToolCall
   │
   ▼
cmd/tools.go: handleQuery
   │ isReadOnlyQuery(sql) → true
   ▼
internal.Client.Query(sql)
   │ ValidateQuery(sql)        ← classifier runs here
   │   ├─ stacked? no
   │   ├─ first verb = SELECT  ← in readOnlyVerbs
   │   └─ INTO OUTFILE? no
   │ → driver.QueryContext(ctx, sql)
   ▼
MySQL/MariaDB
   │
   ▼
processRows → QueryResult
   │
   ▼
cmd/format.go: formatQueryResultStructured
   │
   ▼
ToolResponse{Content:[{Type:"text", Text:"..."}]}
   │
   ▼
Claude Desktop
```

### Write (INSERT/UPDATE/DELETE)

```
Claude Desktop
   │ tools/call name=execute, arguments={sql:"UPDATE ...", confirm_key:"..."}
   ▼
cmd/tools.go: handleExecute
   │
   ▼
internal.Client.Execute(sql, confirmKey)
   │ ValidateQuery(sql)         ← classifier runs here
   │ → BeginTx + ExecContext inside the transaction
   │ → result.RowsAffected()
   │
   │ if rowsAffected > MaxSafeRows and no valid confirmKey:
   │     Rollback()               ← row-count gate: changes never committed
   │ else:
   │     Commit()
   ▼
QueryResult{RowCount, Message:"Query executed successfully. Rows affected: N"}
```

The row-count gate is enforced inside an explicit transaction (`BeginTx` + conditional `Commit`/`Rollback`). If the number of affected rows exceeds `MaxSafeRows` and no valid `confirm_key` is supplied, `Rollback()` is called before the changes are committed. This makes the safety guarantee real for InnoDB (and any transactional engine). Non-transactional storage engines are inherently outside this protection.

## Configuration reference

See `SECURITY.md` for the full table. Quick summary:

- Connection: `MYSQL_HOST`, `MYSQL_PORT`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE`.
- Behaviour: `ALLOW_DDL`, `SAFETY_KEY`, `MAX_SAFE_ROWS`, `ALLOWED_TABLES`.
- Operations: `LOG_PATH`.

## Performance

- **Connection pool**: `MaxOpenConns=10`, `MaxIdleConns=5`, `ConnMaxLifetime=1h`, `ConnMaxIdleTime=15m`.
- **Timeouts**: per-operation profiles in `timeout.go`.
- **Classifier overhead**: a few string operations and one `containsVerb` lookup per call. The `BenchmarkValidateQuery` test reports the cost; in practice it's well below network latency.

## Extending the server

### Adding a new tool

1. Add a `ToolDefinition` to `getToolsList()` in `cmd/tools.go`.
2. Write a `handleXxx(client, args)` function in the same file.
3. Add the case to the `switch` in `callClientMethod`.
4. If the tool builds SQL from caller input, route the final string through `client.Query` / `client.Execute` so it goes through `ValidateQuery`. For identifiers (table/column names), use `sanitizeIdentifier()` and `?` placeholders where the driver supports them.
5. Add classifier-level tests to `test/security/integration_test.go` if the new tool widens the SQL surface.

### Changing the verb classifier

The verb lists live in `internal/client.go` near the top:

```go
readOnlyVerbs = []string{...}
writeVerbs    = []string{...}
ddlVerbs      = []string{...}
callVerbs     = []string{...}
forbiddenVerbs = []string{...}
```

Move a verb between lists or add new ones, then update `test/security/integration_test.go` so the suite reflects the new policy. Don't add regex-based "dangerous patterns" — the whitelist plus the row-count gate plus filesystem-clause check is the design.

## Tests

```bash
go test ./...                      # all
go test -v ./test/security/...     # classifier
go test -bench=. ./test/security/...
go vet ./...
govulncheck ./...
```
