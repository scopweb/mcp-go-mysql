# Skill: MCP Go MySQL Development

## Description

Professional development skill for the **mcp-go-mysql** project — an enterprise-grade MCP (Model Context Protocol) server written in Go for secure MySQL/MariaDB database access through Claude Desktop.

Use this skill when working on any aspect of this project: adding MCP tools, modifying security layers, writing tests, building, or deploying.

---

## Project Architecture

### Layers

| Layer | Location | Responsibility |
|-------|----------|----------------|
| **MCP Protocol** | `cmd/` | JSON-RPC 2.0 message handling, tool definitions, tool routing |
| **Database & Security** | `internal/` | Secure MySQL client, query validation, rate limiting, audit, timeouts, error sanitization |
| **Security Tests** | `test/security/`, `cmd/security/` | CVE detection, injection pattern tests, code security analysis |
| **Documentation** | `docs/` | Architecture, Claude Desktop config, security practices |

### Key Files

- `cmd/main.go` — Server entry point, stdin/stdout JSON-RPC loop
- `cmd/types.go` — MCP message types (`MCPMessage`, `MCPError`, `ToolResponse`)
- `cmd/handlers.go` — Message routing (`initialize`, `tools/list`, `tools/call`)
- `cmd/tools.go` — 10 MCP tool definitions and handlers
- `cmd/security.go` — Write operation safety, DDL confirmation, row estimation
- `internal/client.go` — Secure MySQL client with SQL injection protection (23+ patterns)
- `internal/mysql.go` — Database CRUD operations
- `internal/audit.go` — Structured JSON audit logging with event builder
- `internal/ratelimit.go` — Token bucket rate limiter (query/write/admin buckets)
- `internal/timeout.go` — Context-based timeout management per operation type
- `internal/error_sanitizer.go` — Error masking to prevent information leakage
- `internal/db_compat.go` — MySQL 8.x / MariaDB 11.8 LTS auto-detection

---

## MCP Tool Development

### Current Tools (10)

| Tool | Type | Handler | Description |
|------|------|---------|-------------|
| `query` | query | `handleQuery` | Execute SELECT/WITH/SHOW queries |
| `execute` | write | `handleExecute` | Execute INSERT/UPDATE/DELETE with safety |
| `tables` | admin | `handleTables` | List tables with metadata |
| `describe` | admin | `handleDescribe` | Table structure and columns |
| `views` | admin | `handleViews` | List database views |
| `indexes` | admin | `handleIndexes` | Show table indexes |
| `explain` | query | `handleExplain` | Query execution plan |
| `count` | query | `handleCount` | Count rows with optional WHERE |
| `sample` | query | `handleSample` | Sample rows (max 100) |
| `database_info` | admin | `handleDatabaseInfo` | Server and connection info |

### Adding a New MCP Tool

Follow these exact steps to add a new tool:

#### Step 1: Define the tool in `cmd/tools.go`

Add the `ToolDefinition` entry to `getToolsList()`:

```go
{
    Name:        "new_tool",
    Description: "Clear description of what the tool does.",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param_name": map[string]interface{}{
                "type":        "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param_name"},
    },
},
```

#### Step 2: Add the handler function in `cmd/tools.go`

```go
func handleNewTool(client *mysql.Client, args map[string]interface{}) (string, error) {
    param, ok := args["param_name"].(string)
    if !ok || param == "" {
        return "", fmt.Errorf("missing or invalid 'param_name' parameter")
    }

    // Use client methods with security validation
    result, err := client.Query(/* ... */)
    if err != nil {
        return "", err
    }

    return formatQueryResult(result)
}
```

#### Step 3: Register in the router (`cmd/tools.go`)

Add to `callClientMethod()` switch:

```go
case "new_tool":
    return handleNewTool(client, args)
```

And classify the operation type for rate limiting:

```go
// In the opType switch:
case "new_tool":
    opType = "query" // or "write" or "admin"
```

#### Step 4: Write tests

Create test cases in the appropriate test file in `cmd/` following existing patterns.

---

## Security Requirements

Every code change MUST respect the multi-layer security architecture:

### Layer 1: Input Validation
- All SQL goes through `ValidateQuery()` which checks 23+ injection patterns
- Table names validated via `ValidateTableAccess()` and `isValidIdentifier()`
- Identifiers sanitized through `sanitizeIdentifier()` — allows only `[a-zA-Z0-9_]`

### Layer 2: Operation Control
- DDL blocked by default (`BlockDDL: true`, controlled by `ALLOW_DDL` env var)
- Dangerous operations always blocked: `DROP DATABASE`, `TRUNCATE`, `DELETE` without WHERE, etc.
- Write operations require confirmation key for >100 rows affected
- Rate limiting: query=1000/s, write=100/s, admin=10/s

### Layer 3: Connection Security
- Connection pool: max 10 open, 5 idle, 1h lifetime, 15min idle timeout
- Context-based timeouts per operation type (query, write, admin, connection)
- Error sanitization removes IPs, hostnames, ports, paths, credentials from errors

### Security Checklist for Code Changes
- [ ] No raw SQL concatenation with user input — use `QueryPrepared()` or `sanitizeIdentifier()`
- [ ] New parameters validated before use
- [ ] Error messages don't leak internal details (use `ErrorSanitizer`)
- [ ] Rate limit category assigned correctly in `callClientMethod()`
- [ ] No `INFORMATION_SCHEMA` queries in user-facing tools without prepared statements
- [ ] Table access checked via whitelist if `ALLOWED_TABLES` is configured

---

## Build & Test Commands

### Build
```bash
# Compile the binary
go build -o mysql-mcp ./cmd

# Tidy dependencies
go mod tidy

# Check for vulnerabilities
govulncheck ./...
```

### Test
```bash
# Run all tests (170+)
go test -v ./...

# Run with coverage
go test -v -cover ./...

# Run specific test categories
go test -v ./cmd/... -run "RateLimit"
go test -v ./cmd/... -run "Timeout"
go test -v ./cmd/... -run "Audit"
go test -v ./cmd/... -run "ErrorSanitizer"
go test -v ./cmd/... -run "Compatibility"

# Security tests
go test -v ./test/security/...
go test -v ./cmd/security/...

# Benchmarks
go test -bench=. ./test/security/...

# Race condition detection
go test -race ./...
```

### Vet & Lint
```bash
go vet ./...
```

---

## Code Conventions

### Language
- Code comments and log messages use Spanish in existing code (`log.Printf("Manejando método: %s", ...)`)
- Public API documentation uses English
- Keep consistency with existing patterns

### Go Patterns Used
- **Fluent builder pattern**: `AuditEventBuilder` (`NewAuditEvent().WithOperation().WithUser().Build()`)
- **Interface-based abstractions**: `AuditLogger` interface with `NoOpAuditLogger` and `InMemoryAuditLogger`
- **Token bucket algorithm**: `TokenBucket` struct with `AcquireToken()` / `AcquireTokenWithWait()`
- **Context-based timeouts**: `TimeoutConfig` with operation-type profiles
- **Error sanitization**: `ErrorSanitizer` wrapping errors for client safety
- **Compiled regex patterns**: `init()` pre-compiles all security patterns for performance

### Module Path
The Go module is `mcp-gp-mysql` (note: not `mcp-go-mysql`). Internal imports use:
```go
import mysql "mcp-gp-mysql/internal"
```

### MCP Protocol Compliance
- JSON-RPC 2.0 with `jsonrpc: "2.0"` on all messages
- Tool execution errors use `isError: true` in `ToolResponse`, NOT JSON-RPC protocol errors
- Protocol version echoed from client for Claude Desktop compatibility
- Notifications (e.g., `notifications/initialized`) return `nil` (no response)

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MYSQL_HOST` | Yes | `localhost` | MySQL server hostname |
| `MYSQL_PORT` | No | `3306` | MySQL server port |
| `MYSQL_USER` | Yes | — | MySQL username |
| `MYSQL_PASSWORD` | Yes | — | MySQL password |
| `MYSQL_DATABASE` | Yes | — | Default database |
| `LOG_PATH` | No | `mysql-mcp.log` | Log file path |
| `ALLOWED_TABLES` | No | — | Comma-separated whitelist |
| `ALLOW_DDL` | No | `false` | Enable DDL operations |
| `SAFETY_KEY` | No | `PRODUCTION_CONFIRMED_2025` | Confirmation key |
| `MAX_SAFE_ROWS` | No | `100` | Row threshold for confirmation |
| `DB_TYPE` | No | `mariadb` | Database type (`mysql` or `mariadb`) |

---

## Common Development Tasks

### Adding a new security pattern

1. Add the regex to `sqlInjectionPatterns` in `internal/client.go`
2. Add test case to `test/security/cves_test.go` under the appropriate CWE category
3. Run `go test -v ./test/security/... ./cmd/security/...`

### Adding a new audit event type

1. Add `EventType` constant in `internal/audit.go`
2. Add method to `AuditLogger` interface
3. Implement in `NoOpAuditLogger` and `InMemoryAuditLogger`
4. Add test cases in `cmd/audit_test.go`

### Adding a new rate limit category

1. Add field to `RateLimitConfig` in `internal/ratelimit.go`
2. Create new `TokenBucket` in `NewRateLimiter()`
3. Add `Allow<Category>()` method
4. Register in `callClientMethod()` switch in `cmd/tools.go`
5. Add tests in `cmd/ratelimit_test.go`

### Adding database compatibility

1. Update `internal/db_compat.go` with new version detection
2. Add compatibility config for the new version
3. Test with `go test -v ./cmd/... -run "Compatibility"`

---

## Deployment

### Claude Desktop Configuration

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "user",
        "MYSQL_PASSWORD": "password",
        "MYSQL_DATABASE": "mydb"
      }
    }
  }
}
```

### Cross-compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o mysql-mcp-linux ./cmd

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o mysql-mcp-macos ./cmd

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o mysql-mcp-macos-arm ./cmd

# Windows
GOOS=windows GOARCH=amd64 go build -o mysql-mcp.exe ./cmd
```
