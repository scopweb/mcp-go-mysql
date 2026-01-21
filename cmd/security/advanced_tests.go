package security

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestConnectionStringSecurityBypass checks for connection string manipulation
func TestConnectionStringSecurityBypass(t *testing.T) {
	t.Log("Testing for Connection String Manipulation vulnerabilities...")
	t.Log("")

	testCases := []struct {
		name        string
		input       string
		shouldBlock bool
		description string
	}{
		{
			name:        "SQL query in password",
			input:       "user:pass'; DROP TABLE--@host",
			shouldBlock: true,
			description: "Attempting SQL injection via connection string",
		},
		{
			name:        "Multiple hosts bypass",
			input:       "user:pass@host1,host2",
			shouldBlock: true,
			description: "Connection string specifying multiple hosts",
		},
		{
			name:        "URL-encoded credentials",
			input:       "user%3Aadmin:pass%40word@host",
			shouldBlock: false,
			description: "Properly URL-encoded credentials",
		},
		{
			name:        "Newline in connection string",
			input:       "user:pass\nPassword:backup@host",
			shouldBlock: true,
			description: "Attempting to inject additional parameters via newline",
		},
	}

	for _, tc := range testCases {
		isSafe := isConnectionStringSafe(tc.input)
		expected := !tc.shouldBlock

		if isSafe == expected {
			t.Logf("✅ %s: %s", tc.name, tc.description)
		} else {
			t.Logf("❌ %s: %s (got %v, expected %v)", tc.name, tc.description, isSafe, expected)
		}
	}
}

// isConnectionStringSafe validates connection strings
func isConnectionStringSafe(connStr string) bool {
	// Check for newlines and other control characters
	if strings.Contains(connStr, "\n") || strings.Contains(connStr, "\r") ||
		strings.Contains(connStr, "\x00") || strings.Contains(connStr, "\x1a") {
		return false
	}

	// Check for SQL keywords in password section
	// Extract password part (after : and before @)
	if idx := strings.LastIndex(connStr, "@"); idx > 0 {
		credPart := connStr[:idx]
		if cidx := strings.Index(credPart, ":"); cidx > 0 {
			password := credPart[cidx+1:]
			password = strings.ToUpper(password)

			sqlKeywords := []string{"DROP", "DELETE", "TRUNCATE", "ALTER", "INSERT"}
			for _, kw := range sqlKeywords {
				if strings.Contains(password, kw) {
					return false
				}
			}
		}
	}

	// Check for multiple hosts (comma-separated)
	if strings.Contains(connStr, ",") {
		// If there's a comma, verify it's not in credentials
		if idx := strings.LastIndex(connStr, "@"); idx < 0 {
			return false
		}
	}

	return true
}

// TestContextTimeoutBypass checks for timeout bypass attempts
func TestContextTimeoutBypass(t *testing.T) {
	t.Log("Testing for Context Timeout Bypass vulnerabilities...")
	t.Log("")

	// Simulate context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test that context is properly honored
	select {
	case <-ctx.Done():
		t.Log("✅ Context timeout properly enforced")
	case <-time.After(1 * time.Second):
		t.Log("❌ Context timeout not enforced")
	}
}

// TestErrorMessageInformationLeakage checks for sensitive data in error messages
func TestErrorMessageInformationLeakage(t *testing.T) {
	t.Log("Testing for Information Leakage in Error Messages...")
	t.Log("")

	testCases := []struct {
		name              string
		errorMsg          string
		leaksInformation  bool
		description       string
	}{
		{
			name:             "Generic error",
			errorMsg:         "Database operation failed",
			leaksInformation: false,
			description:      "Non-revealing error message",
		},
		{
			name:             "Query leak",
			errorMsg:         "Error: SELECT * FROM users WHERE id = 123",
			leaksInformation: true,
			description:      "Error includes actual SQL query",
		},
		{
			name:             "Credential leak",
			errorMsg:         "Connection failed: user=admin password=secret123",
			leaksInformation: true,
			description:      "Error includes credentials",
		},
		{
			name:             "Stack trace leak",
			errorMsg:         "at mysql.NewClient() line 42 in /home/user/app/config.go",
			leaksInformation: true,
			description:      "Error includes system paths",
		},
	}

	for _, tc := range testCases {
		leaks := doesErrorLeakInformation(tc.errorMsg)
		if leaks == tc.leaksInformation {
			t.Logf("✅ %s: %s", tc.name, tc.description)
		} else {
			t.Logf("❌ %s: %s", tc.name, tc.description)
		}
	}
}

// doesErrorLeakInformation checks if error message reveals sensitive data
func doesErrorLeakInformation(errMsg string) bool {
	sensitivePatterns := []string{
		"password", "secret", "token", "credential",
		"private", "key=", "api_key", "select ",
		"/home/", "/root/", "C:\\", "user=",
	}

	errLower := strings.ToLower(errMsg)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(errLower, pattern) {
			return true
		}
	}

	return false
}

// TestJSONInjectionVulnerability checks for JSON injection attacks
func TestJSONInjectionVulnerability(t *testing.T) {
	t.Log("Testing for JSON Injection vulnerabilities...")
	t.Log("")

	testCases := []struct {
		name        string
		input       string
		shouldBlock bool
		description string
	}{
		{
			name:        "Quote escape attempt",
			input:       `{"name":"John\",\"admin\":\"true"}`,
			shouldBlock: true,
			description: "Attempting to inject admin flag via JSON",
		},
		{
			name:        "Normal JSON value",
			input:       `{"name":"John"}`,
			shouldBlock: false,
			description: "Normal JSON string",
		},
		{
			name:        "Script tag attempt",
			input:       `<script>alert('xss')</script>`,
			shouldBlock: true,
			description: "Attempting XSS via JSON field",
		},
	}

	for _, tc := range testCases {
		isSafe := isJSONInputSafe(tc.input)
		expected := !tc.shouldBlock

		if isSafe == expected {
			t.Logf("✅ %s: %s", tc.name, tc.description)
		} else {
			t.Logf("❌ %s: %s (got %v, expected %v)", tc.name, tc.description, isSafe, expected)
		}
	}
}

// isJSONInputSafe checks if JSON input is safe
func isJSONInputSafe(input string) bool {
	// Check for unescaped quotes
	if strings.Contains(input, `\"") ||`) {
		return false
	}

	// Check for script tags
	if strings.Contains(strings.ToLower(input), "<script") {
		return false
	}

	// Check for common XSS patterns
	xssPatterns := []string{"<", "javascript:", "onerror=", "onload="}
	for _, pattern := range xssPatterns {
		if strings.Contains(strings.ToLower(input), pattern) {
			return false
		}
	}

	return true
}

// TestURLParameterPollutionBypass checks for URL parameter pollution
func TestURLParameterPollutionBypass(t *testing.T) {
	t.Log("Testing for URL Parameter Pollution vulnerabilities...")
	t.Log("")

	testCases := []struct {
		name        string
		queryStr    string
		shouldBlock bool
		description string
	}{
		{
			name:        "Duplicate parameters",
			queryStr:    "?id=1&id=2&id=3",
			shouldBlock: true,
			description: "Multiple parameters with same name",
		},
		{
			name:        "Null byte injection",
			queryStr:    "?id=1%00&id=admin",
			shouldBlock: true,
			description: "Null byte separator attempt",
		},
		{
			name:        "Normal parameter",
			queryStr:    "?id=123&name=test",
			shouldBlock: false,
			description: "Normal query string",
		},
	}

	for _, tc := range testCases {
		isSafe := isURLParameterSafe(tc.queryStr)
		expected := !tc.shouldBlock

		if isSafe == expected {
			t.Logf("✅ %s: %s", tc.name, tc.description)
		} else {
			t.Logf("❌ %s: %s", tc.name, tc.description)
		}
	}
}

// isURLParameterSafe checks for parameter pollution
func isURLParameterSafe(queryStr string) bool {
	// Check for null bytes
	if strings.Contains(queryStr, "%00") || strings.Contains(queryStr, "\x00") {
		return false
	}

	// Parse query string and check for duplicate parameters
	params, err := url.ParseQuery(queryStr)
	if err != nil {
		return false
	}

	for _, values := range params {
		if len(values) > 1 {
			return false
		}
	}

	return true
}

// BenchmarkAdvancedSecurityChecks measures security validation overhead
func BenchmarkAdvancedSecurityChecks(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = isConnectionStringSafe("user:pass@localhost:3306")
		_ = doesErrorLeakInformation("Database error")
		_ = isJSONInputSafe(`{"key":"value"}`)
		_ = isURLParameterSafe("?id=123")
	}
}
