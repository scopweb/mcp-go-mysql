# CLAUDE.md — Project Instructions for Claude Code

## Project

**mcp-go-mysql** — Enterprise MCP server (Go 1.24) for secure MySQL/MariaDB access via Claude Desktop.

## Quick Reference

```bash
# Build
go build -o mysql-mcp ./cmd

# Test all (170+ tests)
go test -v ./...

# Test with coverage
go test -v -cover ./...

# Security tests only
go test -v ./test/security/... ./cmd/security/...

# Vet
go vet ./...

# Vulnerability check
govulncheck ./...
```

## Architecture

```
cmd/           → MCP protocol layer (main, handlers, tools, security)
internal/      → Database & security layer (client, audit, ratelimit, timeout, errors)
test/security/ → Security pattern tests and CVE detection
docs/          → Architecture, deployment, security documentation
```

## Key Conventions

- **Module path**: `mcp-gp-mysql` (imports: `mysql "mcp-gp-mysql/internal"`)
- **Protocol**: MCP 2.0 over JSON-RPC 2.0 via stdin/stdout
- **Log language**: Spanish for internal logs, English for public API docs
- **Error handling**: Tool errors use `ToolResponse.IsError = true`, NOT JSON-RPC protocol errors
- **Security**: All user input validated through `ValidateQuery()` (5 focused patterns: time-based/XML injection), `sanitizeIdentifier()`, and table whitelist
- **Rate limiting**: Token bucket per operation type — query (1000/s), write (100/s), admin (10/s)
- **No external dependencies** beyond `github.com/go-sql-driver/mysql`

## Adding a New MCP Tool

1. Define `ToolDefinition` in `cmd/tools.go` → `getToolsList()`
2. Write handler function `handleXxx()` in `cmd/tools.go`
3. Register in `callClientMethod()` switch with correct rate limit category
4. Write tests following existing patterns in `cmd/`

## Do Not

- Introduce raw SQL concatenation with user input
- Remove or weaken security patterns in `internal/client.go`
- Add external dependencies without justification
- Expose internal errors to clients (use `ErrorSanitizer`)
- Skip rate limit classification for new tools
