# Security Test Suite - MCP Go MSSQL

Complete security testing framework for the MCP Go MSSQL server.

## Overview

This test suite provides comprehensive security scanning for the Go-based MSSQL MCP server, including:

- **Dependency Vulnerability Scanning** - Check for known CVEs in dependencies
- **Code Security Analysis** - Detect unsafe patterns and potential vulnerabilities
- **Module Integrity Verification** - Ensure go.mod and go.sum haven't been tampered
- **SQL Injection Detection** - Validate protection against SQL injection attacks
- **Static Analysis** - Code quality and security issues (requires gosec)
- **Race Condition Detection** - Identify data races with `-race` flag
- **Coverage Analysis** - Security test coverage metrics

## Test Files

### 1. `security_tests.go`
Core security unit tests for the MCP Go MSSQL project.

**Tests included:**
- `TestDependencyVersions` - Verify all dependencies are current
- `TestGoModuleIntegrity` - Check go.mod for suspicious patterns
- `TestGoSumIntegrity` - Validate go.sum file structure
- `TestMainDependencies` - Track critical dependencies (go-mssqldb, crypto, etc.)
- `TestNoPrivateKeyCommitted` - Detect accidentally committed secrets
- `TestNoDangerousImports` - Check for unsafe/syscall imports
- `TestInputValidation` - Verify SQL and input validation patterns
- `TestErrorHandling` - Check error handling coverage
- `TestLogSanitization` - Ensure logs don't leak sensitive data
- `TestGoVersion` - Verify Go version compatibility
- `TestCommunitySecurityAdvisories` - Check for known vulnerable packages

### 2. `cves_test.go`
Known CVE detection and security pattern analysis.

**Tests included:**
- `TestKnownCVEs` - Check for known vulnerabilities in dependencies
- `TestGolangSecurityDatabase` - Go's official vulnerability database info
- `TestCommonWeaknessPatterns` - CWE patterns relevant to database apps
- `TestSQLInjectionVulnerability` - SQL injection attack detection (CWE-89)
- `TestPathTraversalVulnerability` - CWE-22 path traversal detection
- `TestCommandInjectionVulnerability` - CWE-78 command injection detection
- `TestRACEVulnerabilities` - Race condition patterns
- `TestMemorySafetyVulnerabilities` - Memory safety assessment
- `TestCryptographyVulnerabilities` - Crypto algorithm review
- `TestDependencySupplyChainRisk` - Supply chain risk assessment
- `TestSoftwareCompositionAnalysis` - SCA tool recommendations
- `TestRegexVulnerabilities` - ReDoS (Regular Expression DoS) detection
- `TestSecurityConfigurationBaseline` - Establish baseline
- `TestSecurityHeadersAndDefenses` - Defense mechanism verification
- `TestFuzzingRecommendations` - Fuzzing guidance
- `TestSecurityAuditLog` - Audit documentation

## Running Tests

### Quick Start

```bash
# Navigate to project root
cd /path/to/mcp-go-mssql

# Run all security tests
go test ./test/security -v

# Run with race detection
go test ./test/security -race -v

# Run with coverage
go test ./test/security -coverprofile=coverage.out

# Run specific test
go test ./test/security -run TestSQLInjectionVulnerability -v

# Run benchmarks
go test ./test/security -bench=. -benchmem
```

### From Test Directory

```bash
cd test/security

# Run all tests
go test -v

# Run with verbose output
go test -v -count=1
```

## Security Analysis Results

### Threat Model

MCP Go MSSQL is a database connectivity service with these primary attack surfaces:

1. **SQL Injection (CWE-89)** - Malicious SQL queries through the MCP interface
2. **Authentication Bypass (CWE-287)** - Unauthorized database access
3. **Connection String Exposure** - Credential leakage in logs or errors
4. **Race Conditions** - Concurrent database access issues
5. **Dependency Vulnerabilities** - Third-party package exploits

### Current Status

| Category | Status | Notes |
|----------|--------|-------|
| Unit Tests | PASS | All security tests passing |
| Module Integrity | OK | go.mod/go.sum verified |
| Dependencies | OK | go-mssqldb, x/crypto, x/text, testify |
| Unsafe Code | OK | No unsafe imports |
| Secrets | OK | No hardcoded credentials |
| SQL Injection Protection | OK | Parameterized queries used |
| Error Handling | OK | Consistent error returns |
| Input Validation | OK | Query validation implemented |
| TLS Encryption | OK | Mandatory for production |

## Key Vulnerabilities Tested

### CWE-89: SQL Injection
Tests detect:
- `1' OR '1'='1` - Classic injection
- `UNION SELECT * FROM users--` - Union-based
- `admin'--` - Comment injection
- `1; DROP TABLE users--` - Stacked queries
- `1' AND SLEEP(5)--` - Time-based blind injection

### CWE-22: Path Traversal
Tests detect:
- `../../../etc/passwd`
- `..\..\windows\system32`
- `/etc/passwd` (absolute paths)
- URL-encoded variations: `%2e%2e/`

### CWE-78: OS Command Injection
Tests detect:
- Shell metacharacters: `;` `|` `&` `` ` ``
- Command substitution: `$(...)`
- Pipe chains: `file.txt | cat /etc/passwd`

## Go Security Features

This codebase leverages:
- Memory safety (automatic)
- Type safety (compile-time)
- Bounds checking (automatic)
- Race detection flag (`-race`)
- Fuzzing support (`-fuzz`)
- Go vulnerability database (Go 1.21+)

## Continuous Integration

For CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Security Tests
  run: |
    go test ./test/security -v -race -coverprofile=coverage.out

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    files: coverage.out
```

## Installing Security Tools

Optional tools for enhanced analysis:

```bash
# Static analysis
go install github.com/securego/gosec/v2/cmd/gosec@latest

# CVE detection
go install github.com/sonatype-nexus-oss/nancy@latest

# License compliance
go install github.com/google/go-licenses@latest

# SBOM generation
go install github.com/anchore/syft/cmd/syft@latest

# Run gosec on the project
gosec ./...

# Run nancy for CVE detection
go list -json -m all | nancy sleuth
```

## Security Best Practices

1. **Always run tests before deploying:**
   ```bash
   go test ./test/security -v -race
   ```

2. **Keep dependencies updated:**
   ```bash
   go get -u ./...
   go mod tidy
   ```

3. **Use race detector during development:**
   ```bash
   go test -race ./...
   ```

4. **Check for new vulnerabilities:**
   ```bash
   go list -m all | nancy sleuth
   # Or with Go 1.21+
   go vuln ./...
   ```

5. **Never commit credentials:**
   - Use `.env` files (already in `.gitignore`)
   - Use environment variables
   - Review code before committing

## Troubleshooting

### "Module verification failed"
```bash
go mod tidy
go mod verify
```

### "Unknown test package"
Ensure you're in the project root directory:
```bash
cd /path/to/mcp-go-mssql
go test ./test/security -v
```

### "Command 'gosec' not found"
Install gosec for static analysis:
```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

### Tests timeout
Increase timeout:
```bash
go test ./test/security -timeout 10m
```

## Security Test Metrics

Baseline metrics:

- **Tests:** 25+ security-focused tests
- **Coverage:** Run with `go test -cover ./test/security`
- **Critical Issues:** 0
- **High Issues:** 0
- **Key Dependencies:** go-mssqldb, x/crypto, x/text, testify
- **Review Frequency:** Monthly recommended

## License

Same as parent project (see LICENSE file)

## Security Reporting

For security issues:
1. **DO NOT** create public GitHub issues
2. Use GitHub's private security advisory feature
3. Email maintainers directly if available

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP SQL Injection Prevention](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
- [Go Security Best Practices](https://golang.org/doc/security)
- [CWE/SANS Top 25](https://cwe.mitre.org/top25/)
- [CVE Database](https://cve.mitre.org/)
- [Go Vulnerability Database](https://vuln.go.dev/)
- [Microsoft SQL Server Security](https://docs.microsoft.com/en-us/sql/relational-databases/security/)
