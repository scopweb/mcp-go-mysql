# Advanced MySQL MCP Server with Intelligent Security

Production-ready MySQL Model Context Protocol (MCP) server in Go with comprehensive database tools, intelligent security system, and SQL injection protection. Features automatic protection for dangerous operations with confirmation keys and modular architecture.

## Table of Contents
- [Security Notice](#-important-security-notice)
- [Features](#-features)
- [Installation](#-installation)
- [Claude Desktop Configuration](#-claude-desktop-configuration)
- [Usage Examples](#-usage-examples)
- [Security Tests](#-security-tests)
- [Project Structure](#-project-structure)
- [Security Configuration](#-security-configuration)
- [Documentation](#-documentation)

## Important Security Notice

**ALWAYS BACKUP YOUR DATABASE BEFORE USING WRITE OPERATIONS**

This server provides powerful database tools that can modify your data. Please:
- **Create backups** before performing any write operations
- **Test operations** on development databases first
- **Use appropriate MySQL user permissions** - create a dedicated MySQL user with only the permissions you need
- **Review SQL statements** carefully before execution
- **Monitor operation logs** for security auditing

### Recommended MySQL User Setup

Create a dedicated MySQL user with minimal required permissions:

```sql
-- Create dedicated user for MCP
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'secure_password';

-- Grant only necessary permissions (adjust as needed)
GRANT SELECT, INSERT, UPDATE, DELETE ON your_database.* TO 'mcp_user'@'%';
GRANT CREATE, DROP, ALTER ON your_database.* TO 'mcp_user'@'%';  -- Only if DDL needed
GRANT SHOW VIEW, CREATE VIEW, DROP VIEW ON your_database.* TO 'mcp_user'@'%';

-- Refresh privileges
FLUSH PRIVILEGES;
```

**Never use root or admin users in production!**

## Features

### Database Tools (10 Available)

| Tool | Description |
|------|-------------|
| `query` | Execute SELECT queries (read-only, security validated) |
| `execute` | Execute INSERT/UPDATE/DELETE with confirmation |
| `tables` | List all tables with metadata |
| `describe` | Describe table/view structure |
| `views` | List all database views |
| `indexes` | Show indexes for a table |
| `explain` | Analyze query execution plans |
| `count` | Count rows with optional WHERE |
| `sample` | Get sample rows (max 100) |
| `database_info` | Show connection and server info |

### Security Features

#### SQL Injection Protection (23+ patterns blocked)
- Classic injection (`' OR '1'='1`)
- UNION-based injection
- Comment injection (`--`, `#`, `/* */`)
- Stacked queries (`;`)
- Time-based blind (`SLEEP`, `BENCHMARK`)
- Hex encoding attacks
- MySQL-specific: `EXTRACTVALUE`, `UPDATEXML`, `LOAD_FILE`

#### Dangerous Operation Blocking
| Operation | Status |
|-----------|--------|
| `DROP DATABASE/SCHEMA` | Blocked |
| `TRUNCATE TABLE` | Blocked |
| `DELETE` without WHERE | Blocked |
| `UPDATE` without WHERE | Blocked |
| `INTO OUTFILE/DUMPFILE` | Blocked |
| `LOAD DATA/LOAD_FILE` | Blocked |

#### Intelligent Risk Assessment
- **Small operations** (≤100 rows) → Execute freely
- **Large operations** (>100 rows) → Require confirmation key
- **DDL operations** (CREATE/DROP/ALTER) → Always require confirmation
- **Database drops** → Completely blocked

## Installation

### 1. Clone and Build

```bash
git clone https://github.com/scopweb/mcp-go-mysql.git
cd mcp-go-mysql
go mod tidy
go build -o mysql-mcp ./cmd
```

### 2. Run Security Tests (Recommended)

```bash
go test -v ./test/security/...
```

### 3. Create Environment File (Optional)

Create `.env` file in the project directory:
```env
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=mcp_user
MYSQL_PASSWORD=secure_password
MYSQL_DATABASE=your_database
LOG_PATH=mysql-mcp.log
ALLOWED_TABLES=users,orders,products  # Optional: whitelist tables
ALLOW_DDL=false                        # Optional: enable DDL operations
```

## Claude Desktop Configuration

### Configuration File Location

| Platform | Configuration File Path |
|----------|------------------------|
| **Windows** | `%APPDATA%\Claude\claude_desktop_config.json` |
| **macOS** | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| **Linux** | `~/.config/Claude/claude_desktop_config.json` |

### Windows Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "C:\\Users\\YourUser\\mcp-go-mysql\\mysql-mcp.exe",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "your_secure_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "C:\\Users\\YourUser\\mcp-go-mysql\\mysql-mcp.log"
      }
    }
  }
}
```

### macOS Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/Users/youruser/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "your_secure_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "/Users/youruser/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

### Linux Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/home/youruser/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "your_secure_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "/home/youruser/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

### Docker/Remote MySQL Configuration

```json
{
  "mcpServers": {
    "mysql-remote": {
      "command": "/path/to/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "db.example.com",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_readonly",
        "MYSQL_PASSWORD": "secure_remote_password",
        "MYSQL_DATABASE": "production_db",
        "ALLOWED_TABLES": "users,orders,products,categories"
      }
    }
  }
}
```

### Environment Variables Reference

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

### Verifying Configuration

After configuring Claude Desktop:

1. **Restart Claude Desktop** completely
2. **Open a new conversation**
3. **Ask Claude**: "What MySQL tools do you have available?"
4. **Test connection**: "List all tables in my database"

If connection fails, check:
- MySQL server is running and accessible
- Credentials are correct
- Firewall allows connections on the MySQL port
- Log file for error messages

## Usage Examples

### Safe Operations (No Confirmation Required)

```sql
-- Query data
SELECT * FROM products WHERE category='electronics' LIMIT 10

-- Small updates (affects ≤100 rows)
UPDATE orders SET status='shipped' WHERE order_id=12345

-- Describe structures
DESCRIBE customers

-- Count rows
SELECT COUNT(*) FROM users WHERE active=1
```

### Protected Operations (Require Confirmation)

```sql
-- Mass updates (requires: confirm_key="PRODUCTION_CONFIRMED_2025")
UPDATE products SET discount=0.1 WHERE category='clearance'

-- DDL operations (always require confirmation)
CREATE VIEW monthly_sales AS
SELECT DATE_FORMAT(date,'%Y-%m') as month, SUM(total)
FROM orders GROUP BY month
```

### Blocked Operations

```sql
-- These are ALWAYS blocked for safety:
DROP DATABASE production           -- Database deletion blocked
DELETE FROM users                  -- DELETE without WHERE blocked
UPDATE users SET role='admin'      -- UPDATE without WHERE blocked
SELECT * INTO OUTFILE '/tmp/data'  -- File write blocked
SELECT LOAD_FILE('/etc/passwd')    -- File read blocked
```

## Security Tests

Run the comprehensive security test suite:

```bash
# Run all security tests
go test -v ./test/security/...

# Run specific test categories
go test -v ./test/security/... -run "SQL"      # SQL injection tests
go test -v ./test/security/... -run "Path"     # Path traversal tests
go test -v ./test/security/... -run "CVE"      # CVE vulnerability tests

# Run with coverage
go test -v -cover ./test/security/...

# Run benchmarks
go test -bench=. ./test/security/...
```

### Test Coverage

| Category | Tests | Status |
|----------|-------|--------|
| SQL Injection | 23 patterns | Pass |
| Path Traversal | 9 patterns | Pass |
| Command Injection | 10 patterns | Pass |
| Dangerous SQL | 9 operations | Pass |
| Client Validation | 22 cases | Pass |

### CWE Coverage

| CWE ID | Description | Protection |
|--------|-------------|------------|
| CWE-89 | SQL Injection | Pattern matching + prepared statements |
| CWE-22 | Path Traversal | URL decode + pattern blocking |
| CWE-78 | Command Injection | Metacharacter blocking |
| CWE-287 | Improper Auth | Environment variables |
| CWE-311 | Missing Encryption | TLS support |
| CWE-522 | Credential Exposure | Masked logging |
| CWE-400 | Resource Exhaustion | Connection pooling |

## Project Structure

```
mcp-go-mysql/
├── cmd/
│   ├── main.go           # Server entry point
│   ├── types.go          # MCP message structures
│   ├── handlers.go       # Message routing
│   ├── tools.go          # Tool implementations
│   └── security.go       # Security helpers for write operations
├── internal/
│   ├── client.go         # Secure MySQL client with security validation
│   ├── mysql.go          # Database operations and query execution
│   └── analysis.go       # Query analysis and optimization tools
├── test/
│   └── security/
│       ├── security_tests.go    # Dependency & code tests
│       ├── cves_test.go         # CVE & injection tests
│       ├── integration_test.go  # Client integration tests
│       └── README.md            # Test documentation
├── docs/
│   ├── ARCHITECTURE.md          # System architecture
│   ├── CLAUDE_DESKTOP.md        # Claude Desktop setup guide
│   └── SECURITY.md              # Security best practices
├── go.mod
├── go.sum
├── CHANGELOG.md
└── README.md
```

## Security Configuration

### Current Settings
- **Safety Key**: `PRODUCTION_CONFIRMED_2025`
- **Row Limit**: `100 rows` (operations affecting more require confirmation)

### Customizing Security

Edit `cmd/main.go`:
```go
const (
    SAFETY_KEY    = "YOUR_CUSTOM_KEY_2025"
    MAX_SAFE_ROWS = 50  // Adjust threshold
)
```

### Table Whitelist

Restrict access to specific tables:
```bash
# In environment or Claude Desktop config
ALLOWED_TABLES=users,orders,products,categories
```

## Troubleshooting

### Connection Issues

```bash
# Test MySQL connection
mysql -h localhost -u mcp_user -p your_database

# Check if server is listening
netstat -an | grep 3306
```

### Log Analysis

```bash
# View recent logs
tail -f mysql-mcp.log

# Search for errors
grep -i error mysql-mcp.log
```

### Common Errors

| Error | Solution |
|-------|----------|
| "Connection refused" | Check MySQL is running and port is correct |
| "Access denied" | Verify username/password and user permissions |
| "Unknown database" | Confirm database exists and user has access |
| "Security validation failed" | Query contains blocked patterns |

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run security tests: `go test -v ./test/security/...`
4. Submit a pull request

## Documentation

For detailed documentation, see the `docs/` directory:

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture and component overview |
| [CLAUDE_DESKTOP.md](docs/CLAUDE_DESKTOP.md) | Complete Claude Desktop setup and integration guide |
| [SECURITY.md](docs/SECURITY.md) | Security best practices and configuration |

### Quick Links

- **New to MCP Go MySQL?** Start with [Claude Desktop Setup](docs/CLAUDE_DESKTOP.md)
- **Understanding the codebase?** Read [Architecture](docs/ARCHITECTURE.md)
- **Security concerns?** Review [Security Best Practices](docs/SECURITY.md)

## License

MIT License - See LICENSE file for details.

---

**Built for production environments with security as the top priority. Always backup your data!**

**Optimized for Claude Desktop** - Seamless integration with Anthropic's Claude Desktop application.
