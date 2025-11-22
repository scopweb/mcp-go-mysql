# Security Tests for MCP Go MySQL

This directory contains comprehensive security tests for the MCP Go MySQL server.

## Test Files

### `security_tests.go`
General security validation tests:
- **TestDependencyVersions**: Checks for outdated dependencies
- **TestGoModuleIntegrity**: Verifies go.mod hasn't been tampered with
- **TestGoSumIntegrity**: Validates dependency checksums
- **TestMainDependencies**: Checks critical dependency versions
- **TestNoPrivateKeyCommitted**: Scans for accidentally committed secrets
- **TestNoDangerousImports**: Checks for unsafe/dangerous imports
- **TestInputValidationExists**: Verifies input validation is implemented
- **TestSecurityConstantsDefined**: Checks security constants
- **TestErrorHandlingExists**: Verifies proper error handling
- **TestNoHardcodedCredentials**: Scans for hardcoded credentials
- **TestContextTimeoutsUsed**: Checks for context timeout usage
- **TestConnectionPoolConfigured**: Verifies connection pool settings

### `cves_test.go`
CVE and vulnerability tests:
- **TestKnownCVEs**: Documents known CVEs for dependencies
- **TestGolangSecurityDatabase**: Provides security scanning guidance
- **TestCommonWeaknessPatterns**: Reviews relevant CWE patterns
- **TestSQLInjectionVulnerability**: Comprehensive SQL injection tests
- **TestPathTraversalVulnerability**: Path traversal attack tests
- **TestCommandInjectionVulnerability**: Command injection tests
- **TestDangerousSQLOperations**: Dangerous SQL operation blocking
- **TestTableWhitelistValidation**: Table access validation tests
- **TestMySQLSpecificInjections**: MySQL-specific attack patterns

## Running Tests

```bash
# Run all security tests
cd test/security
go test -v ./...

# Run specific test
go test -v -run TestSQLInjectionVulnerability

# Run with coverage
go test -v -cover ./...
```

## Security Features Tested

### SQL Injection Prevention
- Classic injection patterns (`' OR '1'='1`)
- Union-based injection
- Comment-based injection (`--`, `#`, `/* */`)
- Stacked queries (`;`)
- Time-based blind injection (`SLEEP`, `BENCHMARK`)
- Information schema enumeration
- Hex encoding attacks
- Function-based obfuscation

### Dangerous Operation Blocking
- DROP DATABASE/SCHEMA
- TRUNCATE TABLE
- DELETE/UPDATE without WHERE
- INTO OUTFILE/DUMPFILE
- LOAD DATA/LOAD_FILE

### Path Traversal Prevention
- Unix-style traversal (`../`)
- Windows-style traversal (`..\`)
- URL-encoded traversal
- Absolute path blocking

### Command Injection Prevention
- Shell metacharacters (`;`, `|`, `&`)
- Command substitution (`` ` ``, `$()`)
- Variable expansion (`${}`)
- Newline injection

## Continuous Security

### Recommended Tools
```bash
# Install Go vulnerability checker
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run vulnerability scan
govulncheck ./...

# Install staticcheck for static analysis
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

### Pre-commit Hooks
Add to `.git/hooks/pre-commit`:
```bash
#!/bin/bash
govulncheck ./...
go test ./test/security/... -v
```

## Security Checklist

- [x] SQL injection prevention with pattern matching
- [x] Prepared statements for parameterized queries
- [x] Table access whitelist support
- [x] DDL operation blocking (configurable)
- [x] Dangerous operation detection
- [x] Connection timeout configuration
- [x] Connection pool limits
- [x] Password masking in logs
- [x] Environment variable configuration
- [x] No hardcoded credentials
- [x] Proper error handling with wrapping
- [x] Context timeouts for all operations

## CWE Coverage

| CWE ID | Description | Status |
|--------|-------------|--------|
| CWE-89 | SQL Injection | ✅ Tested |
| CWE-22 | Path Traversal | ✅ Tested |
| CWE-78 | Command Injection | ✅ Tested |
| CWE-287 | Improper Authentication | ✅ Documented |
| CWE-311 | Missing Encryption | ✅ TLS Support |
| CWE-522 | Credential Protection | ✅ Env Vars |
| CWE-400 | Resource Consumption | ✅ Limits |

## Contributing

When adding new features:
1. Add corresponding security tests
2. Update this README
3. Run full test suite before committing
4. Consider CWE implications
