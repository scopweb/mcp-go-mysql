# MCP Go MySQL - Architecture

This document describes the architecture and design of the MCP Go MySQL server.

## Overview

MCP Go MySQL is a Model Context Protocol (MCP) server that provides secure MySQL database access through Claude Desktop. It implements the MCP specification for tool-based interactions with databases.

```
┌─────────────────┐     MCP Protocol      ┌──────────────────┐
│  Claude Desktop │ ◄──────────────────► │  MCP Go MySQL    │
│                 │     (JSON-RPC 2.0)    │     Server       │
└─────────────────┘                       └────────┬─────────┘
                                                   │
                                          Security │ Validation
                                                   │
                                          ┌────────▼─────────┐
                                          │   MySQL Server   │
                                          │                  │
                                          └──────────────────┘
```

## Components

### 1. Command Layer (`cmd/`)

The command layer handles MCP protocol communication and tool routing.

#### `main.go`
- Entry point for the server
- Reads JSON-RPC messages from stdin
- Writes responses to stdout
- Manages the main message processing loop
- Configures logging and environment variables

#### `types.go`
- Defines MCP message structures
- `MCPMessage`: Main message container (JSON-RPC 2.0)
- `MCPError`: Error response structure
- `ToolResponse`: Tool execution result
- `ContentItem`: Response content wrapper

#### `handlers.go`
- Routes incoming MCP messages to appropriate handlers
- Handles `initialize`, `tools/list`, `tools/call`, and notifications
- Returns structured MCP responses

#### `tools.go`
- Implements all 10 database tools
- Routes tool calls to client methods
- Formats query results for MCP responses

#### `security.go`
- Handles write operation security
- Estimates affected rows for safety checks
- Manages DDL operation confirmations
- Strips SQL comments for analysis

### 2. Internal Layer (`internal/`)

The internal layer provides secure database operations.

#### `client.go`
Core security client with:
- **SecurityConfig**: Safety key, row limits, table whitelist
- **Client**: Main database client with connection pooling
- **Validation Methods**:
  - `ValidateQuery()`: SQL injection detection
  - `ValidateTableAccess()`: Table whitelist enforcement
- **Security Functions**:
  - `IsSafeSQL()`: SQL injection pattern detection
  - `IsSafePath()`: Path traversal protection
  - `IsSafeCommand()`: Command injection protection
- **Query Methods**:
  - `Query()`: SELECT with security validation
  - `QueryPrepared()`: Parameterized queries
  - `Execute()`: INSERT/UPDATE/DELETE with confirmation

#### `mysql.go`
Database operations:
- `getDB()`: Connection management
- `ExecuteQuerySimple()`: Simple SELECT execution
- `DescribeSimple()`: Table structure description
- `ListViewsSimple()`: View listing
- View management functions

#### `analysis.go`
Advanced analysis tools:
- `ExplainQuery()`: Query plan analysis
- `AnalyzeObject()`: Object analysis (tables, views)
- `OptimizeTables()`: Table optimization
- `ShowProcessList()`: Active process listing
- Performance analysis helpers

### 3. Test Layer (`test/security/`)

Comprehensive security testing:

#### `security_tests.go`
- Dependency version checks
- Module integrity verification
- Secret scanning
- Import validation
- Code pattern verification

#### `cves_test.go`
- Known CVE documentation
- SQL injection tests (23+ patterns)
- Path traversal tests (9 patterns)
- Command injection tests (10 patterns)
- Dangerous SQL operation blocking

#### `integration_test.go`
- Client security validation
- Query validation tests
- Security function tests
- Performance benchmarks

## Security Architecture

### Defense in Depth

```
┌─────────────────────────────────────────────────────────────┐
│                    Layer 1: Input Validation                 │
│  - SQL injection pattern matching (23+ patterns)             │
│  - Table name whitelist                                      │
│  - Query type restrictions                                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Layer 2: Operation Control                  │
│  - DDL blocking (configurable)                               │
│  - Dangerous operation detection                             │
│  - Row count estimation and confirmation                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Layer 3: Connection Security                │
│  - Connection pooling with limits                            │
│  - Query timeouts                                            │
│  - TLS support                                               │
└─────────────────────────────────────────────────────────────┘
```

### SQL Injection Protection

Patterns detected and blocked:
- Classic injection (`' OR '1'='1`)
- UNION-based injection
- Comment injection (`--`, `#`, `/* */`)
- Time-based blind injection (`SLEEP`, `BENCHMARK`)
- Information schema enumeration
- Hex encoding attacks
- Function-based obfuscation

### Dangerous Operations Blocked

- `DROP DATABASE/SCHEMA` - Always blocked
- `TRUNCATE TABLE` - Blocked
- `DELETE FROM table` without WHERE - Blocked
- `UPDATE table SET` without WHERE - Blocked
- `INTO OUTFILE/DUMPFILE` - Blocked
- `LOAD DATA/LOAD_FILE` - Blocked

## Data Flow

### Read Operation (SELECT)

```
Claude Desktop → MCP Message → tools/call
                                   │
                                   ▼
                            handleQuery()
                                   │
                                   ▼
                        client.ValidateQuery()
                         (SQL injection check)
                                   │
                                   ▼
                         client.Query(sql)
                                   │
                                   ▼
                          MySQL Server
                                   │
                                   ▼
                      formatQueryResult()
                                   │
                                   ▼
                        MCP Response
```

### Write Operation (INSERT/UPDATE/DELETE)

```
Claude Desktop → MCP Message → tools/call
                                   │
                                   ▼
                          handleExecute()
                                   │
                                   ▼
                       client.ValidateQuery()
                                   │
                                   ▼
                     estimateAffectedRows()
                                   │
              ┌────────────────────┴────────────────────┐
              │                                         │
         ≤100 rows                                  >100 rows
              │                                         │
              ▼                                         ▼
        Execute directly                    Require confirm_key
              │                                         │
              └────────────────────┬────────────────────┘
                                   │
                                   ▼
                          MySQL Server
```

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MYSQL_HOST` | Yes | localhost | MySQL server hostname |
| `MYSQL_PORT` | No | 3306 | MySQL server port |
| `MYSQL_USER` | Yes | - | MySQL username |
| `MYSQL_PASSWORD` | Yes | - | MySQL password |
| `MYSQL_DATABASE` | Yes | - | Default database |
| `LOG_PATH` | No | mysql-mcp.log | Log file path |
| `ALLOWED_TABLES` | No | (all) | Comma-separated whitelist |
| `ALLOW_DDL` | No | false | Enable DDL operations |
| `SAFETY_KEY` | No | PRODUCTION_CONFIRMED_2025 | Confirmation key |
| `MAX_SAFE_ROWS` | No | 100 | Threshold for confirmation |

## Performance Considerations

### Connection Pooling

```go
db.SetMaxOpenConns(10)      // Maximum open connections
db.SetMaxIdleConns(5)       // Maximum idle connections
db.SetConnMaxLifetime(1h)   // Connection lifetime
db.SetConnMaxIdleTime(15m)  // Idle connection timeout
```

### Query Timeouts

All operations use context timeouts (default 30s):
- `QueryContext()` for SELECT operations
- `ExecContext()` for write operations
- `PingContext()` for connection testing

## Extending the Server

### Adding a New Tool

1. Add tool definition in `cmd/tools.go`:
```go
{
    Name:        "new_tool",
    Description: "Tool description",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param": map[string]interface{}{
                "type": "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param"},
    },
}
```

2. Add handler in `callClientMethod()`:
```go
case "new_tool":
    return handleNewTool(client, args)
```

3. Implement handler function:
```go
func handleNewTool(client *mysql.Client, args map[string]interface{}) (string, error) {
    // Implementation
}
```

### Adding Security Patterns

Add patterns to `internal/client.go`:
```go
sqlInjectionPatterns = []string{
    // Existing patterns...
    "(?i)NEW_PATTERN",
}
```

## Testing

### Run All Security Tests

```bash
go test -v ./test/security/...
```

### Run Specific Tests

```bash
go test -v ./test/security/... -run "SQL"      # SQL injection
go test -v ./test/security/... -run "Path"     # Path traversal
go test -v ./test/security/... -run "CVE"      # CVE checks
```

### Benchmarks

```bash
go test -bench=. ./test/security/...
```
