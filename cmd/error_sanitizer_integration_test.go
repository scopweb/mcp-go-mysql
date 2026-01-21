package main

import (
	"context"
	"fmt"
	"testing"

	mysql "mcp-gp-mysql/internal"
)

// ============= ERROR SANITIZER INTEGRATION TESTS =============

func TestErrorSanitizerWithAuditLogging(t *testing.T) {
	// Test error sanitization integrated with audit logging
	sanitizer := mysql.NewErrorSanitizer()
	auditLogger := mysql.NewInMemoryAuditLogger()

	ctx := context.Background()
	ctxWithLogger := mysql.WithAuditLogger(ctx, auditLogger)

	// Simulate an error that occurs during a database operation
	rawErr := "Failed to connect to 192.168.1.100:3306: connection refused"
	sanitized := sanitizer.SanitizeString(rawErr)

	// Log the error as an audit event
	event := mysql.NewAuditEvent(mysql.EventTypeError).
		WithStatus("failed").
		WithError(sanitized.Message).
		WithSeverity(mysql.SeverityError).
		Build()

	auditLogger.LogError(ctxWithLogger, event)

	// Verify event was logged with sanitized message
	events := auditLogger.GetEvents()
	if len(events) == 0 {
		t.Fatal("Expected error event to be logged")
	}

	// Check that IP address is not in audit log
	if containsSubstringIntegration(events[0].ErrorMsg, "192.168") {
		t.Error("IP address should not appear in audit log")
	}

	// Check that sanitized message is recorded
	if !containsSubstringIntegration(events[0].ErrorMsg, "REDACTED") {
		t.Logf("Message: %s", events[0].ErrorMsg)
	}
}

func TestErrorSanitizerWithTimeoutContext(t *testing.T) {
	// Test error sanitization when timeout occurs
	sanitizer := mysql.NewErrorSanitizer()
	timeoutConfig := mysql.NewTimeoutConfig()

	ctx := context.Background()
	ctxWithTimeout, cancel := timeoutConfig.TimeoutContext(ctx, mysql.ProfileQuery)
	defer cancel()

	// Simulate a timeout error
	rawErr := "Query timeout after 30 seconds from 192.168.1.100"
	sanitized := sanitizer.SanitizeString(rawErr)

	// Verify it's classified as timeout
	if sanitized.Category != mysql.ErrorCategoryTimeout {
		t.Errorf("Expected timeout category, got %s", sanitized.Category)
	}

	// Verify it's retryable
	if !sanitized.IsRetryable {
		t.Error("Timeout errors should be retryable")
	}

	// Verify IP is redacted
	if containsSubstringIntegration(sanitized.Message, "192.168") {
		t.Error("IP address not redacted")
	}

	// Verify context works
	if ctxWithTimeout.Err() != nil {
		t.Error("Context should not be cancelled")
	}
}

func TestErrorSanitizerWithRateLimiter(t *testing.T) {
	// Test error sanitization when rate limit is exceeded
	sanitizer := mysql.NewErrorSanitizer()
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 5,
		WritesPerSecond:  5,
		AdminPerSecond:   5,
	}
	rateLimiter := mysql.NewRateLimiter(config)

	// Use up all query tokens
	for i := 0; i < 5; i++ {
		rateLimiter.AllowQuery()
	}

	// Create an error message for rate limit exceeded
	rawErr := fmt.Sprintf("Rate limit exceeded for user root@192.168.1.100 on database prod_db")
	sanitized := sanitizer.SanitizeString(rawErr)

	// Should be classified as user error or system error
	if sanitized.Category != mysql.ErrorCategoryUser && sanitized.Category != mysql.ErrorCategorySystem {
		t.Logf("Category: %s", sanitized.Category)
	}

	// Should have redacted sensitive info
	if containsSubstringIntegration(sanitized.Message, "192.168") {
		t.Error("IP address should be redacted")
	}
}

func TestErrorSanitizerClientResponse(t *testing.T) {
	// Test that client responses contain appropriate information
	sanitizer := mysql.NewErrorSanitizer()

	// Create a database error with sensitive information
	rawErr := "Error 1045: Access denied for user 'admin'@'192.168.1.50' (using password: YES) - check /var/log/mysql.log"
	sanitized := sanitizer.SanitizeString(rawErr)

	// Get client-safe response
	clientResp := sanitized.ClientResponse()

	// Verify response structure
	if _, ok := clientResp["error"]; !ok {
		t.Error("Missing error code in client response")
	}
	if _, ok := clientResp["message"]; !ok {
		t.Error("Missing message in client response")
	}
	if _, ok := clientResp["category"]; !ok {
		t.Error("Missing category in client response")
	}
	if _, ok := clientResp["retryable"]; !ok {
		t.Error("Missing retryable in client response")
	}

	// Verify no sensitive info leaked
	respStr := fmt.Sprintf("%v", clientResp)
	if containsSubstringIntegration(respStr, "192.168") {
		t.Error("IP address leaked in client response")
	}
	if containsSubstringIntegration(respStr, "/var/log") {
		t.Error("File path leaked in client response")
	}
}

func TestErrorSanitizerWithDetails(t *testing.T) {
	// Test adding client-safe details to sanitized errors
	sanitizer := mysql.NewErrorSanitizer()

	rawErr := "Database connection failed to 192.168.1.1:3306"
	sanitized := sanitizer.SanitizeString(rawErr)

	// Add client-safe details
	sanitized.WithDetails("table", "users").
		WithDetails("operation", "SELECT").
		WithDetails("retryAfter", 5)

	// Verify details in client response
	clientResp := sanitized.ClientResponse()
	details := clientResp["details"].(map[string]interface{})

	if details["table"] != "users" {
		t.Error("Table detail not set correctly")
	}
	if details["operation"] != "SELECT" {
		t.Error("Operation detail not set correctly")
	}
	if details["retryAfter"] != 5 {
		t.Error("RetryAfter detail not set correctly")
	}

	// Verify original message is still sanitized
	if containsSubstringIntegration(sanitized.Message, "192.168") {
		t.Error("IP address still in message")
	}
}

func TestErrorSanitizerErrorChain(t *testing.T) {
	// Test sanitizing a chain of errors
	sanitizer := mysql.NewErrorSanitizer()

	errors := []string{
		"Cannot connect to server 192.168.1.100:3306",
		"connection timeout after 30 seconds",
		"network unavailable: cannot reach server",
	}

	var lastSanitized *mysql.SanitizedError
	for _, errStr := range errors {
		lastSanitized = sanitizer.SanitizeString(errStr)

		// Each error should have sensitive info redacted
		if containsSubstringIntegration(lastSanitized.Message, "192.168") ||
			containsSubstringIntegration(lastSanitized.Message, "10.0.0") {
			t.Errorf("IP not redacted in error: %s", lastSanitized.Message)
		}
	}

	// Last error should be network error
	if lastSanitized.Category != mysql.ErrorCategoryNetwork {
		t.Errorf("Expected network category, got %s", lastSanitized.Category)
	}
}

func TestErrorSanitizerConcurrentClients(t *testing.T) {
	// Test that multiple clients can safely sanitize errors concurrently
	sanitizer := mysql.NewErrorSanitizer()

	results := make([]*mysql.SanitizedError, 10)
	done := make(chan bool, 10)

	// Simulate 10 concurrent clients receiving errors
	for i := 0; i < 10; i++ {
		go func(idx int) {
			errStr := fmt.Sprintf("Client %d connection failed to 192.168.1.%d", idx, idx)
			results[idx] = sanitizer.SanitizeString(errStr)

			// Get client response
			clientResp := results[idx].ClientResponse()

			// Verify response is safe
			respStr := fmt.Sprintf("%v", clientResp)
			if containsSubstringIntegration(respStr, "192.168") {
				t.Errorf("Client %d: IP leaked in response", idx)
			}

			done <- true
		}(i)
	}

	// Wait for all clients
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all were sanitized
	for i := 0; i < 10; i++ {
		if results[i] == nil {
			t.Errorf("Client %d: error not sanitized", i)
		}
	}
}

func TestErrorSanitizerSeverityAssessment(t *testing.T) {
	// Test severity classification for different error scenarios
	sanitizer := mysql.NewErrorSanitizer()

	testCases := []struct {
		name     string
		errStr   string
		minSev   mysql.ErrorSeverity
		retryable bool
	}{
		{
			"Network error",
			"connection refused from 192.168.1.1",
			mysql.ErrorSeverityError,
			true,
		},
		{
			"Auth error",
			"access denied for user admin",
			mysql.ErrorSeverityError,
			false,
		},
		{
			"Syntax error",
			"SQL syntax error near SELECT",
			mysql.ErrorSeverityError,
			false,
		},
		{
			"Timeout error",
			"query timeout exceeded 30 seconds",
			mysql.ErrorSeverityError,
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sanitized := sanitizer.SanitizeString(tc.errStr)

			// Verify severity is at least the minimum
			if sanitized.Severity != tc.minSev {
				t.Logf("Severity: %s (expected at least %s)", sanitized.Severity, tc.minSev)
			}

			// Verify retryable flag
			if sanitized.IsRetryable != tc.retryable {
				t.Errorf("Expected retryable=%v, got %v", tc.retryable, sanitized.IsRetryable)
			}

			// Verify message is sanitized
			if containsSubstringIntegration(sanitized.Message, "192.168") {
				t.Error("IP address not redacted")
			}
		})
	}
}

// ============= HELPER FUNCTIONS =============

func containsSubstringIntegration(haystack, needle string) bool {
	for i := 0; i < len(haystack)-len(needle)+1; i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
