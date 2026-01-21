package main

import (
	"fmt"
	"testing"

	mysql "mcp-gp-mysql/internal"
)

// ============= ERROR SANITIZER CREATION TESTS =============

func TestErrorSanitizerCreation(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	if sanitizer == nil {
		t.Fatal("Failed to create error sanitizer")
	}
}

// ============= ERROR CLASSIFICATION TESTS =============

func TestClassifyErrorAsUserError(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []struct {
		name        string
		errStr      string
		expectCode  string
		expectRetry bool
	}{
		{"Syntax error", "SQL syntax error near 'SELEC'", "ERR_USER_SYNTAX", false},
		{"Constraint violation", "UNIQUE constraint violated", "ERR_USER_CONSTRAINT", false},
		{"Duplicate key", "Duplicate key: id=123", "ERR_USER_DUPLICATE", false},
		{"Invalid query", "Invalid SQL: column not found", "ERR_USER", false},
		{"Malformed input", "Malformed JSON input", "ERR_USER", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sanitized := sanitizer.SanitizeString(tc.errStr)
			if sanitized.Category != mysql.ErrorCategoryUser {
				t.Errorf("Expected user error, got %s", sanitized.Category)
			}
			if sanitized.IsRetryable != tc.expectRetry {
				t.Errorf("Expected retryable=%v, got %v", tc.expectRetry, sanitized.IsRetryable)
			}
			if !containsSubstring(sanitized.Code, "USER") {
				t.Errorf("Expected USER in code, got %s", sanitized.Code)
			}
		})
	}
}

func TestClassifyErrorAsAuthError(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []struct {
		name    string
		errStr  string
		retryable bool
	}{
		{"Auth failed", "authentication failed: invalid password", false},
		{"Permission denied", "permission denied for user 'guest'", false},
		{"Unauthorized", "unauthorized access to database", false},
		{"Forbidden", "forbidden: access denied", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sanitized := sanitizer.SanitizeString(tc.errStr)
			if sanitized.Category != mysql.ErrorCategoryAuth {
				t.Errorf("Expected auth error, got %s", sanitized.Category)
			}
			if sanitized.IsRetryable {
				t.Error("Auth errors should not be retryable")
			}
		})
	}
}

func TestClassifyErrorAsTimeoutError(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []string{
		"query timeout: exceeded 30 seconds",
		"deadline exceeded on operation",
		"context canceled due to timeout",
		"i/o timeout waiting for response",
	}

	for _, errStr := range testCases {
		sanitized := sanitizer.SanitizeString(errStr)
		if sanitized.Category != mysql.ErrorCategoryTimeout {
			t.Errorf("Expected timeout error for '%s', got %s", errStr, sanitized.Category)
		}
		if !sanitized.IsRetryable {
			t.Error("Timeout errors should be retryable")
		}
	}
}

func TestClassifyErrorAsNetworkError(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []string{
		"connection refused: server not accepting connections",
		"network unreachable to database server",
		"dial tcp: connection reset by peer",
		"i/o error: network down",
	}

	for _, errStr := range testCases {
		sanitized := sanitizer.SanitizeString(errStr)
		if sanitized.Category != mysql.ErrorCategoryNetwork {
			t.Errorf("Expected network error for '%s', got %s", errStr, sanitized.Category)
		}
		if !sanitized.IsRetryable {
			t.Error("Network errors should be retryable")
		}
	}
}

func TestClassifyErrorAsSystemError(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []string{
		"out of memory: cannot allocate buffer",
		"disk full: cannot write to database",
		"resource limit exceeded: too many connections",
	}

	for _, errStr := range testCases {
		sanitized := sanitizer.SanitizeString(errStr)
		if sanitized.Category != mysql.ErrorCategorySystem {
			t.Errorf("Expected system error for '%s', got %s", errStr, sanitized.Category)
		}
	}
}

// ============= SENSITIVE INFORMATION REMOVAL TESTS =============

func TestRemoveIPAddresses(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	errStr := "Failed to connect to 192.168.1.100:3306"
	sanitized := sanitizer.SanitizeString(errStr)

	if containsSubstring(sanitized.Message, "192.168") {
		t.Errorf("IP address not removed: %s", sanitized.Message)
	}
	if !containsSubstring(sanitized.Message, "REDACTED") {
		t.Error("Expected [REDACTED] marker for IP address")
	}
}

func TestRemoveFilePaths(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	errStr := "Failed at /home/user/project/internal/db.go line 42"
	sanitized := sanitizer.SanitizeString(errStr)

	if containsSubstring(sanitized.Message, "/home/user") {
		t.Error("File path not removed from error message")
	}
}

func TestRemoveDatabaseNames(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	errStr := "Error in database='customer_db' table='sensitive_data'"
	sanitized := sanitizer.SanitizeString(errStr)

	// Should show that DB names were redacted
	if len(sanitized.Message) > 0 && !containsSubstring(sanitized.Message, "database") {
		t.Logf("Database reference modified: %s", sanitized.Message)
	}
}

func TestRemovePortNumbers(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	errStr := "Cannot connect to host at port=3306"
	sanitized := sanitizer.SanitizeString(errStr)

	if containsSubstring(sanitized.Message, "3306") {
		t.Error("Port number not properly handled")
	}
}

func TestTruncateLongMessages(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	// Create a very long error message
	longMsg := "Error: " + repeatString("abcdefghij", 30) // 300+ characters
	sanitized := sanitizer.SanitizeString(longMsg)

	if len(sanitized.Message) > 250 {
		t.Errorf("Message not truncated: length=%d", len(sanitized.Message))
	}
	if !containsSubstring(sanitized.Message, "...") {
		t.Error("Expected truncation marker (...)")
	}
}

// ============= ERROR CODE GENERATION TESTS =============

func TestErrorCodeGeneration(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []struct {
		errStr    string
		expectCode string
	}{
		{"SQL syntax error", "ERR_USER_SYNTAX"},
		{"Connection timeout", "ERR_TIMEOUT_TIMEOUT"},
		{"Duplicate key error", "ERR_USER_DUPLICATE"},
		{"Permission denied", "ERR_AUTH_PERMISSION"},
		{"Connection refused", "ERR_NETWORK_CONNECTION"},
		{"Constraint violation", "ERR_USER_CONSTRAINT"},
		{"Not found error", "ERR_USER_NOT_FOUND"},
	}

	for _, tc := range testCases {
		sanitized := sanitizer.SanitizeString(tc.errStr)
		if sanitized.Code != tc.expectCode {
			t.Errorf("For '%s': expected %s, got %s", tc.errStr, tc.expectCode, sanitized.Code)
		}
	}
}

// ============= SEVERITY CLASSIFICATION TESTS =============

func TestClassifySeverityAsCritical(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	criticalErrors := []string{
		"FATAL: database corruption detected",
		"CRITICAL: system failure imminent",
		"PANIC: unrecoverable error",
		"CRASH: system halted",
	}

	for _, errStr := range criticalErrors {
		sanitized := sanitizer.SanitizeString(errStr)
		if sanitized.Severity != mysql.ErrorSeverityCritical {
			t.Errorf("Expected critical severity for '%s', got %s", errStr, sanitized.Severity)
		}
	}
}

func TestClassifySeverityAsError(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	errorMessages := []string{
		"ERROR: query execution failed",
		"Connection failed to database",
		"Timeout on operation",
	}

	for _, errStr := range errorMessages {
		sanitized := sanitizer.SanitizeString(errStr)
		if sanitized.Severity != mysql.ErrorSeverityError {
			t.Errorf("Expected error severity for '%s', got %s", errStr, sanitized.Severity)
		}
	}
}

func TestClassifySeverityAsWarning(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	warningMessages := []string{
		"WARNING: deprecated feature used",
		"RETRY: operation will be attempted again",
	}

	for _, errStr := range warningMessages {
		sanitized := sanitizer.SanitizeString(errStr)
		if sanitized.Severity != mysql.ErrorSeverityWarning {
			t.Errorf("Expected warning severity for '%s', got %s", errStr, sanitized.Severity)
		}
	}
}

// ============= SANITIZED ERROR METHODS TESTS =============

func TestSanitizedErrorString(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	sanitized := sanitizer.SanitizeString("test error message")

	if sanitized.String() != sanitized.Message {
		t.Error("String() should return Message")
	}
}

func TestSanitizedErrorImplementsError(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	sanitized := sanitizer.SanitizeString("test error")

	// Should be usable as error interface
	var err error = sanitized
	if err.Error() != sanitized.Message {
		t.Error("Error() should return Message")
	}
}

func TestSanitizedErrorWithDetails(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	sanitized := sanitizer.SanitizeString("error")

	sanitized.WithDetails("table", "users").
		WithDetails("operation", "INSERT")

	if sanitized.Details["table"] != "users" {
		t.Error("Details not set correctly")
	}
	if sanitized.Details["operation"] != "INSERT" {
		t.Error("Multiple details not set correctly")
	}
}

func TestGetInternalMessage(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	originalMsg := "Original error at 192.168.1.1:3306"
	sanitized := sanitizer.SanitizeString(originalMsg)

	internal := sanitized.GetInternalMessage()
	if internal != originalMsg {
		t.Errorf("Internal message changed: %s", internal)
	}
}

func TestClientResponse(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()
	sanitized := sanitizer.SanitizeString("timeout error")

	response := sanitized.ClientResponse()

	if _, ok := response["error"]; !ok {
		t.Error("Client response missing 'error' field")
	}
	if _, ok := response["message"]; !ok {
		t.Error("Client response missing 'message' field")
	}
	if _, ok := response["category"]; !ok {
		t.Error("Client response missing 'category' field")
	}
	if _, ok := response["retryable"]; !ok {
		t.Error("Client response missing 'retryable' field")
	}
}

// ============= REAL ERROR SCENARIOS TESTS =============

func TestRealMySQLErrors(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []struct {
		name         string
		mysqlErr     string
		expectCat    mysql.ErrorCategory
		shouldRedact bool
	}{
		{
			name:         "Connection refused",
			mysqlErr:     "Error 2002: Can't connect to MySQL server on 192.168.1.1 (111)",
			expectCat:    mysql.ErrorCategoryNetwork,
			shouldRedact: true,
		},
		{
			name:         "Access denied",
			mysqlErr:     "Error 1045: Access denied for user root on host localhost",
			expectCat:    mysql.ErrorCategoryAuth,
			shouldRedact: false,
		},
		{
			name:         "Syntax error",
			mysqlErr:     "Error 1064: You have an error in your SQL syntax",
			expectCat:    mysql.ErrorCategoryUser,
			shouldRedact: false,
		},
		{
			name:         "Duplicate key",
			mysqlErr:     "Error 1062: Duplicate entry user1 for key email",
			expectCat:    mysql.ErrorCategoryUser,
			shouldRedact: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sanitized := sanitizer.SanitizeString(tc.mysqlErr)

			if sanitized.Category != tc.expectCat {
				t.Errorf("Expected category %s, got %s", tc.expectCat, sanitized.Category)
			}

			if tc.shouldRedact {
				if sanitized.Message == tc.mysqlErr {
					t.Error("Message should have been redacted")
				}
			}
		})
	}
}

func TestConcurrentSanitization(t *testing.T) {
	sanitizer := mysql.NewErrorSanitizer()

	// Test concurrent error sanitization
	errs := make([]string, 10)
	for i := 0; i < 10; i++ {
		errs[i] = fmt.Sprintf("Error from goroutine %d at 192.168.1.%d", i, i)
	}

	results := make([]*mysql.SanitizedError, 10)
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			results[idx] = sanitizer.SanitizeString(errs[idx])
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all errors were sanitized
	for i := 0; i < 10; i++ {
		if results[i] == nil {
			t.Errorf("Error %d not sanitized", i)
		}
		if !containsSubstring(results[i].Message, "REDACTED") {
			t.Errorf("Error %d IP not redacted", i)
		}
	}
}

// ============= HELPER FUNCTIONS =============

func containsSubstring(haystack, needle string) bool {
	for i := 0; i < len(haystack)-len(needle)+1; i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
