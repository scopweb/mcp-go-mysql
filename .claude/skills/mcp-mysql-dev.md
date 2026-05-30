# Skill: MCP Go MySQL Development

## Description

Professional development skill for the **mcp-go-mysql** project — an MCP (Model Context Protocol) server written in Go (1.26.3+) for secure MySQL/MariaDB database access through Claude Desktop and other MCP clients.

Use this skill when working on any aspect of this project: adding MCP tools, modifying the verb classifier, writing tests, build configuration, or documentation.

**Related:** See `claude-desktop-usage.md` for effective use of the 10 MCP tools as a database assistant.

---

## Project Architecture

### Layers

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **MCP Protocol** | `cmd/` | JSON-RPC 2.0 over stdio, tool definitions, handlers, routing |
| **Security Tests** | `cmd/security/` | Dependency vulns, module integrity, classifier behavior tests |
| **Database & Policy** | `internal/` | MySQL client, `ValidateQuery` (verb classifier), timeouts, DSN handling, DB compatibility |
| **Documentation** | `docs/`, `README.md` | Architecture, security model, Claude Desktop setup |

**Current internal packages (after 3.0 cleanup):** only `client.go`, `db_compat.go`, `timeout.go`.

### Key Files

- `cmd/main.go` — Server entry point, stdio JSON-RPC loop, env config loading
- `cmd/types.go` — MCP message types (`MCPMessage`, `ToolResponse`, etc.)
- `cmd/handlers.go` — Message routing (`initialize`, `tools/list`, `tools/call`)
- `cmd/tools.go` — 10 MCP tool definitions + handlers + `callClientMethod` switch
- `cmd/sqlcheck.go` — Shared SQL comment stripping (`StripComments`)
- `cmd/format.go` — Structured AI-optimized response formatting
- `internal/client.go` — Core `Client` with `ValidateQuery`, `Query`, `Execute` (transactional safety gate), connection
- `internal/db_compat.go` — MySQL 8.x / MariaDB auto-detection and version-specific behavior
- `internal/timeout.go` — Per-operation timeout profiles (query/write/admin/connection)
- `cmd/security/*.go` — Security test suite (govulncheck-style, integrity, classifier edge cases)

**Removed (3.0.0+):** `internal/ratelimit.go`, `internal/error_sanitizer.go`, `internal/audit.go`, `cmd/security.go` (old), old `test/security/` tree.

---

## Current MCP Tools (10)

| Tool | Category | Description |
|------|----------|-------------|
| `query` | Read | Execute SELECT/WITH/SHOW (read-only) |
| `execute` | Write | INSERT/UPDATE/DELETE/REPLACE inside explicit transaction. Row count gate: > `MAX_SAFE_ROWS` requires valid `confirm_key` or rollback |
| `tables` | Admin | List tables + metadata (engine, rows, comments) |
| `describe` | Admin | Table structure (columns, types, keys, constraints) |
| `views` | Admin | List views |
| `indexes` | Admin | Indexes for a table |
| `explain` | Read | EXPLAIN / execution plan |
| `count` | Read | COUNT(*) for a table (filtered counts → use `query`) |
| `sample` | Read | Sample N rows (max 100) from a table |
| `database_info` | Admin | Connection + server info (version, user, variables) |

All tools that accept SQL route through `client.Query`/`client.Execute` → `ValidateQuery`.

---

## Adding a New MCP Tool

1. Add `ToolDefinition` entry in `cmd/tools.go` → `getToolsList()`.
2. Implement `handleXxx(client *mysql.Client, args map[string]interface{}) (string, error)`.
3. Wire it in the `callClientMethod()` switch in `cmd/tools.go`.
4. If the tool accepts or builds SQL, **always** go through `client.Query` / `client.Execute` (never raw `sql.DB` or string concat).
5. Table/column identifiers must use `sanitizeIdentifier()` or prepared paths.
6. Add behavioral tests (especially for new verbs or safety cases) in `cmd/security/`.

**Never** bypass `ValidateQuery` for new SQL surfaces.

---

## Security Model (Current — Two Layers Only)

**Layer 1 (Primary):** MySQL grants on the dedicated runtime user. This is the real boundary. Never run as root / GRANT OPTION / FILE / CREATE USER / SUPER.

**Layer 2 (Defence-in-depth):** Verb classifier in `internal/client.go:ValidateQuery()`.

- Whitelist of leading verbs after comment stripping (`StripComments`).
- **Always rejected (forbiddenVerbs):** GRANT, REVOKE, SET, FLUSH, RESET, KILL, SHUTDOWN, LOAD, HANDLER, INSTALL, UNINSTALL, LOCK, UNLOCK + `INTO OUTFILE`/`DUMPFILE`.
- **DDL** (CREATE/DROP/ALTER/TRUNCATE/RENAME): rejected unless `ALLOW_DDL=true`.
- **Stacked statements:** detected and rejected.
- **Unknown verbs:** rejected.
- **Write safety gate:** `Execute` wraps DML in explicit tx + `MAX_SAFE_ROWS` check. Bad `confirm_key` → explicit Rollback before any commit.

**What it deliberately does NOT do (by design):**
- No regex "dangerous pattern" lists (removed in 3.0.0 — false positives + incomplete).
- No rate limiting (local stdio process, single human user).
- No error sanitization (LLM needs real driver messages to self-correct typos, column names, etc.).
- No audit logging (never wired to the hot path; removed).

See `README.md` and `docs/SECURITY.md` for the full honest model.

---

## Build, Test, Vet, Security

```bash
# Build
go build -o mysql-mcp ./cmd

# All tests
go test -v ./...

# Coverage
go test -v -cover ./...

# Security / classifier / integrity tests
go test -v ./cmd/security/...

# Vet (no warnings allowed)
go vet ./...

# Vulnerability check (stdlib + deps)
govulncheck ./...
```

**Current Go requirement:** `go 1.26.3+` (see go.mod). This pulls in the fix for GO-2026-4971 (Windows net.Dial panic on NUL in address).

---

## Code Conventions & Language

- **Imports:** `mysql "mcp-gp-mysql/internal"` (module name has the historical typo).
- **Logs:** Spanish for internal `log.Printf` (e.g. "Conectado a...", "Manejando método...").
- **Public strings** (errors, tool descriptions, initialize instructions): English.
- **Tool errors:** Always return `ToolResponse{IsError: true, Content: [...]}` with the **verbatim** driver/database error. Never wrap or sanitize.
- **Protocol:** Strict MCP 2.0 / JSON-RPC 2.0 on stdin/stdout. No extra output on stderr.
- **SQL safety:** All user-facing SQL paths go through `ValidateQuery` + (for writes) the transactional row-count gate. Identifiers → `sanitizeIdentifier()`.

**Do Not (project rule):**
- Re-introduce rate limiters, error sanitizers, audit, or regex dangerous-pattern lists.
- Concatenate user input into SQL without going through the validated client methods.
- Use `root` or privileged MySQL users at runtime.
- Sanitize errors returned to the LLM.

---

## Environment Variables (Runtime)

| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `MYSQL_HOST` | Yes | localhost | |
| `MYSQL_PORT` | No | 3306 | |
| `MYSQL_USER` | Yes | | Dedicated low-privilege user |
| `MYSQL_PASSWORD` | Yes | | |
| `MYSQL_DATABASE` | Yes | | Default DB |
| `LOG_PATH` | No | (none) | If set, detailed logs go here |
| `ALLOWED_TABLES` | No | (all) | Comma-separated whitelist (applied in describe) |
| `ALLOW_DDL` | No | false | Set to "true" to allow CREATE/DROP/ALTER etc. |
| `SAFETY_KEY` | No | PRODUCTION_CONFIRMED_2025 | For >MAX_SAFE_ROWS writes |
| `MAX_SAFE_ROWS` | No | 100 | Threshold for execute confirmation gate |

**Recommendation:** Set a custom `SAFETY_KEY` in production and keep `MAX_SAFE_ROWS` low.

---

## Common Tasks

### Adding a new verb or changing classifier behavior
- Edit the verb slices in `internal/client.go` (`readOnlyVerbs`, `writeVerbs`, `ddlVerbs`, `callVerbs`, `forbiddenVerbs`).
- Update `ValidateQuery` logic and the stacked-statement / `INTO OUTFILE` detectors if needed.
- Add test cases in `cmd/security/security_tests.go` or `advanced_tests.go`.
- Update godoc, tool descriptions, and `initialize` instructions string in `cmd/handlers.go`.

### Database compatibility work
- Update `internal/db_compat.go`.
- Add/adjust tests in `cmd/db_compatibility_test.go`.
- Verify with real MySQL 8.x and MariaDB 11.x instances.

### Changing timeout profiles
- Edit `internal/timeout.go` (ProfileQuery / ProfileWrite / ProfileAdmin / ProfileConnection).
- The profiles are consumed in `client.go` via `TimeoutContext`.

### Releasing / changelog
- Follow Keep a Changelog format in `CHANGELOG.md`.
- Document removals, behavior changes to the safety gate, and any classifier modifications.
- After release, consider whether the `.claude/skills/mcp-mysql-dev.md` needs a corresponding refresh.

---

## References (Authoritative)

- `CLAUDE.md` — Project instructions (this is the source of truth for agents).
- `README.md` + `docs/SECURITY.md` + `docs/ARCHITECTURE.md` — User-facing security model.
- `CHANGELOG.md` — History of the 3.0.0 cleanup (removal of rate limiting, sanitizers, audit, old regex approach).
- `docs/TEST_SAFETY_GATE.md` — How to manually verify the transactional rollback behavior of the MAX_SAFE_ROWS gate.
- `cmd/security/README.md` — Notes on the security test suite.

---

**Status note (post-cleanup):** As of the current session, dead files (`internal/ratelimit.go`, `internal/error_sanitizer.go`) have been removed, `go.mod` requires 1.26.3+, and documentation references have been synchronized. The design is intentionally minimal: MySQL grants + verb classifier + transactional write gate.
