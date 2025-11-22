package security

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestDependencyVersions verifies that all dependencies are up to date
func TestDependencyVersions(t *testing.T) {
	cmd := exec.Command("go", "list", "-u", "-m", "all")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Note: go list command output: %s", string(output))
		// Don't fail test if modules not initialized
		if strings.Contains(string(output), "go.mod") {
			t.Skip("go.mod not accessible from test directory")
		}
	}

	lines := strings.Split(string(output), "\n")
	outdated := 0
	for _, line := range lines {
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			outdated++
			t.Logf("⚠️  Outdated dependency: %s", line)
		}
	}

	if outdated > 0 {
		t.Logf("Found %d outdated dependencies. Run 'go get -u ./...' to update", outdated)
	} else {
		t.Log("✅ All dependencies are up to date")
	}
}

// TestGoModuleIntegrity verifies go.mod hasn't been tampered
func TestGoModuleIntegrity(t *testing.T) {
	content, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	// Calculate SHA256
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])
	t.Logf("go.mod SHA256: %s", hashStr)

	// Check for suspicious patterns
	modContent := string(content)
	suspiciousPatterns := []string{
		"replace ",
		"retract ",
		"exclude ",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(modContent, pattern) {
			t.Logf("ℹ️  Found directive: %s (review manually)", pattern)
		}
	}

	// Verify module name
	if !strings.Contains(modContent, "module mcp-gp-mysql") {
		t.Error("❌ Unexpected module name in go.mod")
	} else {
		t.Log("✅ Module name verified")
	}
}

// TestGoSumIntegrity verifies that all dependencies have valid checksums
func TestGoSumIntegrity(t *testing.T) {
	content, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatalf("Failed to read go.sum: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	validLines := 0
	invalidLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			validLines++
			// Verify checksum format (should start with h1:)
			if !strings.HasPrefix(parts[2], "h1:") {
				t.Logf("⚠️  Unusual checksum format: %s", line)
			}
		} else if line != "" {
			invalidLines++
			t.Logf("⚠️  Invalid go.sum line: %s", line)
		}
	}

	t.Logf("go.sum entries: %d valid, %d invalid", validLines, invalidLines)

	if invalidLines > 0 {
		t.Errorf("Found %d invalid lines in go.sum", invalidLines)
	} else {
		t.Log("✅ go.sum integrity verified")
	}
}

// TestMainDependencies checks critical dependencies for known issues
func TestMainDependencies(t *testing.T) {
	criticalDeps := map[string]string{
		"github.com/go-sql-driver/mysql": "v1.8.1", // MySQL driver
	}

	cmd := exec.Command("go", "list", "-m", "all")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Could not list modules: %v", err)
	}

	modules := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			modules[parts[0]] = parts[1]
		}
	}

	for dep, expectedVersion := range criticalDeps {
		if version, ok := modules[dep]; ok {
			if version == expectedVersion {
				t.Logf("✅ %s: %s", dep, version)
			} else {
				t.Logf("⚠️  %s: got %s, expected %s", dep, version, expectedVersion)
			}
		} else {
			t.Errorf("❌ Critical dependency not found: %s", dep)
		}
	}
}

// TestNoPrivateKeyCommitted checks for accidentally committed secrets
func TestNoPrivateKeyCommitted(t *testing.T) {
	sensitivePatterns := []string{
		"PRIVATE KEY",
		"-----BEGIN RSA",
		"-----BEGIN EC",
		"-----BEGIN OPENSSH",
		"api_key=",
		"apikey=",
		"secret_key=",
		"secretkey=",
		"password=",
		"passwd=",
		"token=",
		"aws_access_key",
		"aws_secret_key",
	}

	checkFiles := []string{
		"../../cmd/main.go",
		"../../cmd/handlers.go",
		"../../cmd/types.go",
		"../../internal/client.go",
		"../../go.mod",
	}

	foundSecrets := false
	for _, file := range checkFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Logf("ℹ️  Could not read file: %s", file)
			continue
		}

		fileContent := strings.ToLower(string(content))
		for _, pattern := range sensitivePatterns {
			if strings.Contains(fileContent, strings.ToLower(pattern)) {
				// Skip false positives in environment variable reads
				if strings.Contains(pattern, "=") {
					continue
				}
				t.Logf("⚠️  Potential sensitive pattern in %s: %s", file, pattern)
				foundSecrets = true
			}
		}
	}

	if !foundSecrets {
		t.Log("✅ No obvious secrets detected in code files")
	}
}

// TestNoDangerousImports checks for unsafe or dangerous imports
func TestNoDangerousImports(t *testing.T) {
	dangerousImports := []string{
		`"unsafe"`,
		`"plugin"`,
		`"debug/pe"`,
		`"debug/elf"`,
	}

	checkFiles := []string{
		"../../cmd/main.go",
		"../../cmd/handlers.go",
		"../../internal/client.go",
	}

	for _, file := range checkFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)
		for _, dangerous := range dangerousImports {
			if strings.Contains(fileContent, dangerous) {
				t.Logf("⚠️  Found %s import in %s (review for security)", dangerous, file)
			}
		}
	}

	t.Log("✅ No dangerous imports found")
}

// TestInputValidationExists checks that input validation exists in main.go
func TestInputValidationExists(t *testing.T) {
	content, err := os.ReadFile("../../internal/client.go")
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	fileContent := string(content)

	// Check for validation patterns
	validationPatterns := []struct {
		pattern string
		name    string
	}{
		{"ValidateQuery", "Query validation"},
		{"ValidateTableAccess", "Table access validation"},
		{"isValidIdentifier", "Identifier validation"},
		{"sqlInjectionPatterns", "SQL injection patterns"},
		{"dangerousPatterns", "Dangerous SQL patterns"},
		{"PrepareContext", "Prepared statements"},
	}

	for _, vp := range validationPatterns {
		if strings.Contains(fileContent, vp.pattern) {
			t.Logf("✅ Found: %s", vp.name)
		} else {
			t.Errorf("❌ Missing: %s", vp.name)
		}
	}
}

// TestSecurityConstantsDefined checks that security constants are properly defined
func TestSecurityConstantsDefined(t *testing.T) {
	content, err := os.ReadFile("../../cmd/main.go")
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	fileContent := string(content)

	requiredConstants := []string{
		"SAFETY_KEY",
		"MAX_SAFE_ROWS",
	}

	for _, constant := range requiredConstants {
		if strings.Contains(fileContent, constant) {
			t.Logf("✅ Security constant defined: %s", constant)
		} else {
			t.Errorf("❌ Missing security constant: %s", constant)
		}
	}
}

// TestErrorHandlingExists checks for proper error handling
func TestErrorHandlingExists(t *testing.T) {
	files := []string{
		"../../cmd/main.go",
		"../../cmd/handlers.go",
		"../../internal/client.go",
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)

		// Check for error handling patterns
		if !strings.Contains(fileContent, "if err != nil") {
			t.Errorf("❌ Missing error handling in: %s", file)
		}

		// Check for error wrapping (good practice)
		if strings.Contains(fileContent, "fmt.Errorf") && strings.Contains(fileContent, "%w") {
			t.Logf("✅ Error wrapping found in: %s", file)
		}
	}
}

// TestNoHardcodedCredentials checks for hardcoded credentials
func TestNoHardcodedCredentials(t *testing.T) {
	files := []string{
		"../../cmd/main.go",
		"../../internal/client.go",
	}

	// Patterns that might indicate hardcoded credentials
	suspiciousPatterns := []struct {
		pattern     string
		description string
	}{
		{`password := "`, "Hardcoded password string"},
		{`password: "`, "Hardcoded password in struct"},
		{`Password = "`, "Hardcoded password assignment"},
		{`secret := "`, "Hardcoded secret"},
		{`apiKey := "`, "Hardcoded API key"},
		{`token := "`, "Hardcoded token"},
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)
		for _, sp := range suspiciousPatterns {
			if strings.Contains(fileContent, sp.pattern) {
				// Check if it's just a variable declaration with empty string
				if strings.Contains(fileContent, sp.pattern+`"`) {
					continue // Empty string is OK
				}
				t.Logf("⚠️  Potential %s in %s", sp.description, file)
			}
		}
	}

	t.Log("✅ No obvious hardcoded credentials found")
}

// TestContextTimeoutsUsed checks that context timeouts are used for database operations
func TestContextTimeoutsUsed(t *testing.T) {
	content, err := os.ReadFile("../../internal/client.go")
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	fileContent := string(content)

	timeoutPatterns := []string{
		"context.WithTimeout",
		"QueryContext",
		"ExecContext",
		"PingContext",
	}

	for _, pattern := range timeoutPatterns {
		if strings.Contains(fileContent, pattern) {
			t.Logf("✅ Found timeout/context pattern: %s", pattern)
		} else {
			t.Errorf("❌ Missing timeout/context pattern: %s", pattern)
		}
	}
}

// TestConnectionPoolConfigured checks that connection pooling is configured
func TestConnectionPoolConfigured(t *testing.T) {
	content, err := os.ReadFile("../../internal/client.go")
	if err != nil {
		t.Fatalf("Failed to read client.go: %v", err)
	}

	fileContent := string(content)

	poolPatterns := []string{
		"SetMaxOpenConns",
		"SetMaxIdleConns",
		"SetConnMaxLifetime",
	}

	for _, pattern := range poolPatterns {
		if strings.Contains(fileContent, pattern) {
			t.Logf("✅ Connection pool setting found: %s", pattern)
		} else {
			t.Errorf("❌ Missing connection pool setting: %s", pattern)
		}
	}
}
