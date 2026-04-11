---
title: Security
description: Security controls, audit logging, rate limiting, and error sanitization
---

MCP Go MySQL applies security controls across six components, implemented incrementally through the project's development phases.

## Security Features

| Phase | Component | Status |
|-------|-----------|--------|
| 1 | Security Hardening | **Complete** |
| 2 | Database Compatibility | **Complete** |
| 3.1 | Timeout Management | **Complete** |
| 3.2 | Audit Logging | **Complete** |
| 3.3 | Rate Limiting | **Complete** |
| 3.4 | Error Sanitization | **Complete** |

## Phase 1: Security Hardening

### SQL Injection Protection

Detects and blocks **23+ patterns** of SQL injection:

- Classic injection: `' OR '1'='1`
- UNION-based: `UNION SELECT`
- Comments: `--`, `#`, `/* */`
- Stacked queries: `;`
- Blind injection: `SLEEP()`, `BENCHMARK()`
- Hexadecimal encoding
- MySQL functions: `EXTRACTVALUE`, `UPDATEXML`

### Blocking Dangerous Operations

| Operation | Status |
|-----------|--------|
| `DROP DATABASE` | **Blocked** |
| `TRUNCATE TABLE` | **Blocked** |
| `DELETE` without WHERE | **Blocked** |
| `UPDATE` without WHERE | **Blocked** |
| `INTO OUTFILE` | **Blocked** |
| `LOAD_FILE` | **Blocked** |

### Path Traversal Protection

Prevents unauthorized access to system files:

- `../../../etc/passwd` &rarr; Blocked
- `..\..\windows\system32` &rarr; Blocked
- Unauthorized absolute paths &rarr; Blocked
- URL encoding &rarr; Detected and blocked

### Operation Confirmation

- **Small operations** (≤100 rows): Execute without confirmation
- **Large operations** (>100 rows): Require the `SAFETY_KEY` before proceeding
- **DDL operations**: Always require `SAFETY_KEY`

### Safety Key Protection

The `SAFETY_KEY` environment variable protects destructive operations (DROP, TRUNCATE, DELETE without WHERE).

:::caution[Default Safety Key]
If `SAFETY_KEY` is not configured, the server uses `PRODUCTION_CONFIRMED_2025` as default and logs a warning. For production environments, always set a unique key:
```bash
export SAFETY_KEY=$(openssl rand -hex 16)
```
:::

When executing bulk operations (>100 rows) or destructive statements, the MCP client must provide this key to confirm the operation.

## Phase 3.1: Timeout Management

### Timeout Profiles

| Profile | Timeout | Usage |
|---------|---------|-------|
| Query | 30 seconds | Fast SELECT queries |
| Long Query | 5 minutes | Complex queries |
| Write | 2 minutes | INSERT, UPDATE, DELETE |
| Admin | 10 minutes | DDL operations |
| Connection | 15 seconds | Establish connection |

**Benefits:**

- Prevents indefinitely running queries
- Automatically frees resources
- Improves system stability

## Phase 3.2: Audit Logging

Detailed logging of all operations:

### Logged Information

- Timestamp of the operation
- User who executed the operation
- Operation type (SELECT, INSERT, UPDATE, DELETE, DDL)
- Executed SQL query (sanitized)
- Result (success/error)
- Execution time
- Affected rows

### Event Categories

| Category | Severity |
|----------|----------|
| Query Success | **Info** |
| Write Operation | **Warning** |
| Security Violation | **Critical** |
| Connection Error | **Error** |

:::note
Logs are essential for security audits and troubleshooting. Configure the `LOG_PATH` environment variable to enable audit logging.
:::

## Phase 3.3: Rate Limiting

### Token Bucket Algorithm

Implementation of token bucket algorithm for rate control:

| Operation Type | Limit | Purpose |
|---------------|-------|---------|
| Queries (SELECT) | 1,000/second | Prevent query saturation |
| Writes (INSERT/UPDATE/DELETE) | 100/second | Protect data integrity |
| Admin (DDL) | 10/second | Control structural changes |

### Behavior Under Load

- Requests exceeding the per-type limit are rejected with an error, not queued
- Each operation type (queries, writes, DDL) has an independent bucket
- Overhead is sub-microsecond per operation

## Phase 3.4: Error Sanitization

### Sensitive Information Protection

Errors are automatically sanitized before display:

- IP addresses (IPv4/IPv6)
- System file paths
- Database names
- Hostnames
- Port numbers
- SQL query patterns

### Sanitization Example

:::danger[Original Error (internal)]
```
Error connecting to 192.168.1.100:3306, database 'production_db' at /var/lib/mysql/data
```
:::

:::tip[Sanitized Error (client)]
```
Database connection error. Code: DB_CONN_001
```
:::

### Error Categories

| Category | Code | Example |
|----------|------|---------|
| User Error | USR_* | SQL syntax error |
| System Error | SYS_* | Internal server error |
| Network Error | NET_* | Connection failure |
| Auth Error | AUTH_* | Invalid credentials |
| Timeout Error | TO_* | Operation expired |

## Security Validation

### Implemented Tests

| Category | Tests | Status |
|----------|-------|--------|
| SQL Injection | 23 patterns | Pass |
| Path Traversal | 9 patterns | Pass |
| Command Injection | 10 patterns | Pass |
| Dangerous SQL | 9 operations | Pass |
| Client Validation | 22 cases | Pass |

**Total:** 170 tests, all passing.

## CWE Coverage

| CWE | Description | Protection |
|-----|-------------|------------|
| CWE-89 | SQL Injection | **Protected** |
| CWE-22 | Path Traversal | **Protected** |
| CWE-78 | Command Injection | **Protected** |
| CWE-287 | Improper Authentication | **Protected** |
| CWE-311 | Missing Encryption | **TLS Supported** |
| CWE-522 | Credential Protection | **Protected** |
| CWE-400 | Resource Consumption | **Rate Limiting** |

## Best Practices

1. **Never use the root user** for MCP connections
2. **Create dedicated users** with minimal necessary permissions
3. **Use ALLOWED_TABLES** to restrict access in production
4. **Enable audit logging** and review it periodically
5. **Run govulncheck** regularly to detect vulnerabilities
6. **Keep Go updated** to the latest stable version
7. **Use TLS/SSL** for remote database connections
8. **Adjust rate limiting** according to your use case
9. **Review sanitized errors** in internal logs
10. **Make backups** before important write operations

## Vulnerability Scanning

**Current status:** 0 vulnerabilities detected.

Run manual scan:

```bash
govulncheck ./...
```

**Last updated:** Go 1.24.12 (2026-02-01)
