# CLAUDE.md — Project Instructions for Claude Code

## Project

**mcp-go-mysql** — MCP server (Go 1.26.3+) for MySQL/MariaDB access via Claude Desktop.

## Quick Reference

```bash
# Build
go build -o mysql-mcp ./cmd

# Test
go test -v ./...

# Test with coverage
go test -v -cover ./...

# Verb-classifier / security tests only
go test -v ./cmd/security/...

# Vet
go vet ./...

# Vulnerability check
govulncheck ./...
```

## Architecture

```
cmd/           MCP protocol layer (stdin/stdout JSON-RPC, tool dispatch)
cmd/security/  Verb-classifier tests + dependency-integrity tests
internal/      Database client and policy (connection, ValidateQuery, timeouts)
docs/          Architecture and security notes
```

## Key Conventions

- **Module path**: `mcp-gp-mysql` (note the typo — kept for backward compat). Imports: `mysql "mcp-gp-mysql/internal"`.
- **Protocol**: MCP 2.0 over JSON-RPC 2.0 via stdin/stdout.
- **Log language**: Spanish for internal logs, English for public-facing strings (errors, README, instructions sent to the LLM).
- **Tool errors**: returned as `ToolResponse{IsError: true}`, **not** as JSON-RPC protocol errors. Protocol errors are reserved for transport/parsing failures.
- **Errors are passed verbatim** to the caller (no sanitizer). Driver/database messages are useful for the LLM to self-correct (typos in column names, type mismatches, …).
- **Security model**: see README. Two layers — MySQL grants (primary) + verb classifier (`ValidateQuery` in `internal/client.go`). Do **not** add regex-based "dangerous pattern" lists.
- **No external dependencies** beyond `github.com/go-sql-driver/mysql`.

## Adding a New MCP Tool

1. Define `ToolDefinition` in `cmd/tools.go` → `getToolsList()`.
2. Write handler `handleXxx()` in `cmd/tools.go`.
3. Register it in the `callClientMethod()` switch.
4. If the tool builds SQL from user input, route the final statement through `client.Query` / `client.Execute` so it goes through `ValidateQuery`. Identifiers (table/column names) must go through `sanitizeIdentifier()`.
5. If the tool introduces new SQL surfaces or verbs, add tests for the behavior in `cmd/security/` (the main location for classifier and security-related tests).

## Do Not

- Bring back regex-based "SQL injection patterns" or "dangerous patterns". The verb classifier is the design; lists of bad strings are not.
- Concatenate user input into SQL without going through `ValidateQuery` (or `QueryPrepared` with placeholders for identifiers).
- Add a rate limiter, error sanitizer, path-traversal validator, or command-injection validator. They were removed deliberately — they protected against threats the MCP doesn't have, and added bugs and false positives.
- Sanitize errors before returning them to the LLM. The LLM needs the real message to fix its own query.
- Add external dependencies without justification.
- Use `root` (or any user with `GRANT OPTION` / `FILE` / `CREATE USER` / `SUPER`) as the runtime MySQL user. The classifier is defence-in-depth, not the primary boundary.
