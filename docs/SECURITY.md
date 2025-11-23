# Security Best Practices

This document outlines security best practices for deploying and using MCP Go MySQL.

## Overview

MCP Go MySQL implements multiple layers of security to protect against common database attacks. This document explains these protections and how to configure them for your environment.

## Security Features

### 1. SQL Injection Protection

The server detects and blocks 23+ SQL injection patterns:

| Category | Patterns Blocked |
|----------|------------------|
| Classic Injection | `' OR '1'='1`, `' AND '1'='1` |
| Tautology | `1=1`, `'='` |
| UNION-based | `UNION SELECT`, `UNION ALL SELECT` |
| Comment Injection | `--`, `#`, `/* */` |
| Stacked Queries | `;` followed by commands |
| Time-based Blind | `SLEEP()`, `BENCHMARK()`, `WAITFOR DELAY` |
| Schema Enumeration | `INFORMATION_SCHEMA` queries |
| Encoding Attacks | Hex encoding (`0x...`), `CHAR()` |
| Function Abuse | `CONCAT()`, `GROUP_CONCAT()` |
| XML Injection | `EXTRACTVALUE()`, `UPDATEXML()` |
| File Operations | `LOAD_FILE()`, `INTO OUTFILE` |

#### Testing SQL Injection Protection

```bash
# Run SQL injection tests
go test -v ./test/security/... -run "SQLInjection"
```

### 2. Dangerous Operation Blocking

Operations that are always blocked for safety:

| Operation | Reason |
|-----------|--------|
| `DROP DATABASE` | Prevents accidental database deletion |
| `DROP SCHEMA` | Prevents schema destruction |
| `TRUNCATE TABLE` | Prevents mass data deletion |
| `DELETE FROM table` (no WHERE) | Prevents accidental table clearing |
| `UPDATE table SET` (no WHERE) | Prevents mass updates |
| `INTO OUTFILE` | Prevents file system writes |
| `INTO DUMPFILE` | Prevents binary file writes |
| `LOAD DATA INFILE` | Prevents unauthorized file reads |
| `LOAD_FILE()` | Prevents arbitrary file access |

### 3. Path Traversal Protection

Protects against directory traversal attacks:

| Pattern | Description |
|---------|-------------|
| `../` | Unix relative path traversal |
| `..\` | Windows relative path traversal |
| `%2e%2e%2f` | URL-encoded traversal |
| `%252e` | Double URL-encoded |
| `..%c0%af` | Overlong UTF-8 encoding |
| `\\server\share` | UNC paths |
| `/absolute/path` | Absolute Unix paths |
| `C:\path` | Absolute Windows paths |

### 4. Command Injection Protection

Blocks shell metacharacters that could enable command execution:

| Pattern | Description |
|---------|-------------|
| `;` | Command separator |
| `\|` | Pipe operator |
| `&` | Background/chain operator |
| `` ` `` | Command substitution |
| `$()` | Command substitution |
| `${}` | Variable expansion |
| `\n`, `\r` | Newline injection |

## Configuration

### Environment Variables

```bash
# Core connection settings (required)
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=mcp_user
MYSQL_PASSWORD=secure_password
MYSQL_DATABASE=mydb

# Security settings (optional but recommended)
ALLOWED_TABLES=users,orders,products    # Table whitelist
ALLOW_DDL=false                         # Disable DDL operations
SAFETY_KEY=YOUR_CUSTOM_KEY_2025         # Custom confirmation key
MAX_SAFE_ROWS=100                       # Threshold for confirmation
```

### Table Whitelist

Restrict queries to specific tables:

```bash
# Only allow access to these tables
ALLOWED_TABLES=users,orders,products,categories

# All other table access will be blocked
```

### DDL Control

Control Data Definition Language operations:

```bash
# Disable DDL (recommended for production)
ALLOW_DDL=false

# Enable DDL (for development)
ALLOW_DDL=true
```

When DDL is disabled, these operations are blocked:
- `CREATE TABLE/VIEW/INDEX`
- `DROP TABLE/VIEW/INDEX`
- `ALTER TABLE`
- `TRUNCATE TABLE`
- `RENAME TABLE`

### Row Limit Confirmation

Large operations require confirmation:

```bash
# Set threshold (default: 100)
MAX_SAFE_ROWS=50

# Set confirmation key
SAFETY_KEY=MY_PRODUCTION_KEY_2025
```

Operations affecting more than `MAX_SAFE_ROWS` require the confirmation key.

## MySQL User Setup

### Recommended Permissions

Create a dedicated MySQL user with minimal permissions:

```sql
-- Create dedicated user
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'strong_password_here';

-- Grant read permissions (SELECT only)
GRANT SELECT ON mydb.* TO 'mcp_user'@'%';

-- Add write permissions if needed
GRANT INSERT, UPDATE, DELETE ON mydb.* TO 'mcp_user'@'%';

-- Add DDL permissions only if required
GRANT CREATE, DROP, ALTER ON mydb.* TO 'mcp_user'@'%';

-- Apply changes
FLUSH PRIVILEGES;
```

### Read-Only User

For production environments with read-only access:

```sql
CREATE USER 'mcp_readonly'@'%' IDENTIFIED BY 'readonly_password';
GRANT SELECT ON production_db.* TO 'mcp_readonly'@'%';
FLUSH PRIVILEGES;
```

### Table-Specific Permissions

Grant access only to specific tables:

```sql
CREATE USER 'mcp_limited'@'%' IDENTIFIED BY 'limited_password';
GRANT SELECT ON mydb.public_data TO 'mcp_limited'@'%';
GRANT SELECT ON mydb.reports TO 'mcp_limited'@'%';
GRANT SELECT ON mydb.statistics TO 'mcp_limited'@'%';
FLUSH PRIVILEGES;
```

## CWE Coverage

The server provides protection against these Common Weakness Enumerations:

| CWE ID | Name | Protection |
|--------|------|------------|
| CWE-89 | SQL Injection | Pattern matching, prepared statements |
| CWE-22 | Path Traversal | URL decode + pattern blocking |
| CWE-78 | Command Injection | Metacharacter blocking |
| CWE-287 | Improper Authentication | Environment variable credentials |
| CWE-311 | Missing Encryption | TLS support |
| CWE-522 | Credential Exposure | Masked logging |
| CWE-400 | Resource Exhaustion | Connection pooling, timeouts |

## Security Testing

### Run All Security Tests

```bash
go test -v ./test/security/...
```

### Run Specific Test Categories

```bash
# SQL injection tests
go test -v ./test/security/... -run "SQL"

# Path traversal tests
go test -v ./test/security/... -run "Path"

# Command injection tests
go test -v ./test/security/... -run "Command"

# CVE checks
go test -v ./test/security/... -run "CVE"

# Dangerous operations
go test -v ./test/security/... -run "Dangerous"
```

### Vulnerability Scanning

Use Go's built-in vulnerability checker:

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run vulnerability scan
govulncheck ./...
```

### Static Analysis

```bash
# Install staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest

# Run static analysis
staticcheck ./...
```

## Logging and Auditing

### Log Configuration

```bash
# Set log file path
LOG_PATH=/var/log/mysql-mcp.log

# Or in Claude Desktop config
{
  "env": {
    "LOG_PATH": "/path/to/mysql-mcp.log"
  }
}
```

### What Gets Logged

- All incoming MCP messages
- Tool execution attempts
- Security validation failures
- Database errors
- Connection status

### What Gets Masked

- Passwords in connection strings
- Credentials in error messages
- Sensitive data in query results (configurable)

### Log Analysis

```bash
# Monitor for security events
tail -f mysql-mcp.log | grep -i "security\|blocked\|injection\|error"

# Count blocked attempts
grep -c "blocked" mysql-mcp.log

# View recent errors
grep -i error mysql-mcp.log | tail -20
```

## Deployment Recommendations

### Development Environment

```json
{
  "env": {
    "MYSQL_HOST": "localhost",
    "MYSQL_USER": "dev_user",
    "MYSQL_PASSWORD": "dev_password",
    "MYSQL_DATABASE": "development",
    "ALLOW_DDL": "true"
  }
}
```

### Production Environment

```json
{
  "env": {
    "MYSQL_HOST": "prod-db.internal",
    "MYSQL_USER": "mcp_readonly",
    "MYSQL_PASSWORD": "SECURE_PRODUCTION_PASSWORD",
    "MYSQL_DATABASE": "production",
    "ALLOWED_TABLES": "users,orders,products",
    "ALLOW_DDL": "false",
    "SAFETY_KEY": "UNIQUE_PRODUCTION_KEY_2025",
    "MAX_SAFE_ROWS": "50"
  }
}
```

### Network Security

1. **Use Internal Networks**: Deploy MySQL on private networks
2. **Firewall Rules**: Restrict MySQL port (3306) access
3. **TLS Encryption**: Enable TLS for MySQL connections
4. **VPN Access**: Require VPN for remote database access

## Incident Response

### If SQL Injection is Suspected

1. Check logs for blocked patterns:
   ```bash
   grep -i "injection\|blocked\|security" mysql-mcp.log
   ```

2. Review recent queries:
   ```bash
   grep -i "query\|sql" mysql-mcp.log | tail -50
   ```

3. Temporarily restrict access:
   ```bash
   # Add strict table whitelist
   ALLOWED_TABLES=safe_table_only
   ```

### If Credentials are Compromised

1. Immediately change MySQL password
2. Revoke existing user sessions:
   ```sql
   KILL CONNECTION_ID;
   ```
3. Create new user with new credentials
4. Update Claude Desktop configuration
5. Review audit logs for unauthorized access

## Security Checklist

### Before Deployment

- [ ] Created dedicated MySQL user with minimal permissions
- [ ] Configured table whitelist for production
- [ ] Disabled DDL operations for production
- [ ] Set custom safety key
- [ ] Configured appropriate row limit
- [ ] Verified TLS is enabled (if remote)
- [ ] Set up log file with appropriate permissions
- [ ] Ran security tests successfully

### Regular Maintenance

- [ ] Run `govulncheck` monthly
- [ ] Update dependencies quarterly
- [ ] Review logs for security events weekly
- [ ] Rotate MySQL passwords quarterly
- [ ] Review and update table whitelist as needed
- [ ] Test security features after updates

## Reporting Security Issues

If you discover a security vulnerability:

1. **Do not** open a public issue
2. Email security concerns to the maintainers
3. Include detailed reproduction steps
4. Allow 90 days for response before public disclosure

## References

- [OWASP SQL Injection Prevention](https://owasp.org/www-community/attacks/SQL_Injection)
- [CWE Database](https://cwe.mitre.org/)
- [Go Security Best Practices](https://golang.org/doc/security)
- [MySQL Security Guide](https://dev.mysql.com/doc/refman/8.0/en/security.html)
