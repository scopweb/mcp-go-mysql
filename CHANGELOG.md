# Changelog

All notable changes to MCP Go MySQL will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.0] - 2025-11-22

### Added

#### Security Module (`internal/client.go`)
- **SQL Injection Protection**: 23+ detection patterns including:
  - Classic injection (`' OR '1'='1`, `' AND '1'='1`)
  - UNION-based injection
  - Comment injection (`--`, `#`, `/* */`)
  - Stacked queries (`;`)
  - Time-based blind injection (`SLEEP`, `BENCHMARK`)
  - Information schema enumeration
  - Hex encoding attacks (`0x...`)
  - Function-based obfuscation (`CHAR`, `CONCAT`, `GROUP_CONCAT`)
  - MySQL XML injection (`EXTRACTVALUE`, `UPDATEXML`)
  - File operations (`INTO OUTFILE`, `LOAD_FILE`, `LOAD DATA`)

- **Dangerous Operation Blocking**:
  - `DROP DATABASE/SCHEMA` - Always blocked
  - `TRUNCATE TABLE` - Blocked
  - `DELETE FROM table` without WHERE - Blocked
  - `UPDATE table SET` without WHERE - Blocked
  - File write operations (`INTO OUTFILE/DUMPFILE`) - Blocked
  - File read operations (`LOAD_FILE`, `LOAD DATA`) - Blocked

- **Path Traversal Protection**:
  - Unix-style traversal (`../`)
  - Windows-style traversal (`..\`)
  - URL-encoded traversal (`%2e%2e%2f`)
  - Double URL-encoded traversal (`%252e`)
  - Overlong UTF-8 encoding
  - UNC/network paths (`\\server\share`)

- **Command Injection Protection**:
  - Shell metacharacters (`;`, `|`, `&`)
  - Command substitution (`` ` ``, `$()`, `${}`)
  - Newline/carriage return injection

- **Prepared Statements Support**:
  - `QueryPrepared()` method for parameterized queries
  - Automatic parameter binding
  - Safe from SQL injection by design

- **Table Access Whitelist**:
  - Environment variable `ALLOWED_TABLES` for table whitelist
  - Validation on all table access operations

- **Connection Security**:
  - Connection pooling with configurable limits
  - Context timeouts on all operations
  - Automatic connection recovery

#### Security Tests (`test/security/`)
- `security_tests.go`: Dependency and code security validation
  - TestDependencyVersions
  - TestGoModuleIntegrity
  - TestGoSumIntegrity
  - TestMainDependencies
  - TestNoPrivateKeyCommitted
  - TestNoDangerousImports
  - TestInputValidationExists
  - TestSecurityConstantsDefined
  - TestErrorHandlingExists
  - TestNoHardcodedCredentials
  - TestContextTimeoutsUsed
  - TestConnectionPoolConfigured

- `cves_test.go`: CVE and vulnerability testing
  - TestKnownCVEs (4 CVEs documented)
  - TestGolangSecurityDatabase
  - TestCommonWeaknessPatterns (7 CWEs)
  - TestSQLInjectionVulnerability (23 test cases)
  - TestPathTraversalVulnerability (9 test cases)
  - TestCommandInjectionVulnerability (10 test cases)
  - TestDangerousSQLOperations (9 operations)
  - TestTableWhitelistValidation
  - TestMySQLSpecificInjections

- `integration_test.go`: Client integration tests
  - TestClientSecurityValidation (22 test cases)
  - TestClientSecurityConfig
  - TestEmptyQueryValidation
  - TestIdentifierValidation
  - TestSecurityFunctions
  - TestPathSecurityFunctions
  - TestCommandSecurityFunctions
  - BenchmarkSQLValidation
  - BenchmarkInjectionDetection

#### Tools Implementation (`cmd/tools.go`)
- `query` - Execute SELECT queries with security validation
- `execute` - Execute INSERT/UPDATE/DELETE with confirmation
- `tables` - List all tables with metadata
- `describe` - Show table structure
- `views` - List all views
- `indexes` - Show table indexes
- `explain` - Query execution plan analysis
- `count` - Count rows with optional WHERE
- `sample` - Get sample rows (max 100)
- `database_info` - Connection and server information

### Security
- CWE-89: SQL Injection - **Protected**
- CWE-22: Path Traversal - **Protected**
- CWE-78: Command Injection - **Protected**
- CWE-287: Improper Authentication - **Environment variables**
- CWE-311: Missing Encryption - **TLS support**
- CWE-522: Credential Protection - **Masked in logs**
- CWE-400: Resource Consumption - **Pool limits & timeouts**

### Documentation
- Added `test/security/README.md` with comprehensive test documentation
- Security checklist and CWE coverage table
- Recommended security scanning tools

## [1.3.0] - 2025-11-21

### Added
- Initial MCP protocol implementation
- JSON-RPC 2.0 message handling
- Basic MySQL connection management
- Environment variable configuration
- Logging with password masking
- Safety key and row limit constants

### Infrastructure
- MCP message types and structures
- Handler routing for tools/list and tools/call
- Error handling with proper MCP error codes

---

## Migration Notes

### From 1.3.0 to 1.4.0
1. No breaking changes - existing configurations work
2. New optional environment variables:
   - `ALLOWED_TABLES`: Comma-separated list of allowed tables
   - `ALLOW_DDL`: Set to "true" to enable DDL operations
3. Run security tests: `go test -v ./test/security/...`

---

## Security Advisories

### Dependencies
- `github.com/go-sql-driver/mysql`: v1.8.1 (CVE-2024-21096 fixed)
- Run `govulncheck ./...` regularly for vulnerability scanning
