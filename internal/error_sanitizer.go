package internal

import (
	"fmt"
	"regexp"
	"strings"
)

// ErrorCategory classifies errors into different types
type ErrorCategory string

const (
	ErrorCategoryUser     ErrorCategory = "user"      // User input/query errors
	ErrorCategorySystem   ErrorCategory = "system"    // System/infrastructure errors
	ErrorCategoryInternal ErrorCategory = "internal"  // Internal implementation errors
	ErrorCategoryAuth     ErrorCategory = "auth"      // Authentication/authorization errors
	ErrorCategoryTimeout  ErrorCategory = "timeout"   // Timeout errors
	ErrorCategoryNetwork  ErrorCategory = "network"   // Network/connection errors
)

// ErrorSeverity classifies error severity levels
type ErrorSeverity string

const (
	ErrorSeverityInfo     ErrorSeverity = "info"
	ErrorSeverityWarning  ErrorSeverity = "warning"
	ErrorSeverityError    ErrorSeverity = "error"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// SanitizedError provides client-safe error information
type SanitizedError struct {
	// Code is a machine-readable error code
	Code string `json:"code"`
	// Message is the sanitized, client-safe error message
	Message string `json:"message"`
	// Category is the error classification
	Category ErrorCategory `json:"category"`
	// Severity is the error severity level
	Severity ErrorSeverity `json:"severity"`
	// IsRetryable indicates if the operation can be retried
	IsRetryable bool `json:"is_retryable"`
	// Details contains client-safe additional context
	Details map[string]interface{} `json:"details,omitempty"`
	// InternalMessage is the full internal error (never sent to client)
	InternalMessage string `json:"-"`
}

// ErrorSanitizer sanitizes errors for client consumption
type ErrorSanitizer struct {
	sensitivePatterns []*regexp.Regexp
	internalPatterns  []*regexp.Regexp
}

// NewErrorSanitizer creates a new error sanitizer
func NewErrorSanitizer() *ErrorSanitizer {
	es := &ErrorSanitizer{
		sensitivePatterns: make([]*regexp.Regexp, 0),
		internalPatterns:  make([]*regexp.Regexp, 0),
	}

	// Add patterns for sensitive information
	es.addSensitivePatterns()
	es.addInternalPatterns()

	return es
}

// addSensitivePatterns registers patterns for sensitive information
func (es *ErrorSanitizer) addSensitivePatterns() {
	// IP addresses and hostnames
	es.sensitivePatterns = append(es.sensitivePatterns,
		regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`), // IPv4
		regexp.MustCompile(`(?:[0-9a-f]{0,4}:){2,7}[0-9a-f]{0,4}`), // IPv6
		regexp.MustCompile(`\bhostname[:\s=]+\S+`),
		regexp.MustCompile(`\bhost[:\s=]+\S+`),
	)

	// Database names and table names (context-specific)
	es.sensitivePatterns = append(es.sensitivePatterns,
		regexp.MustCompile(`\bdatabase[:\s=]+'[^']*'`),
		regexp.MustCompile(`\btable[:\s=]+'[^']*'`),
	)

	// Port numbers
	es.sensitivePatterns = append(es.sensitivePatterns,
		regexp.MustCompile(`\bport[:\s=]+\d+`),
	)

	// File paths
	es.sensitivePatterns = append(es.sensitivePatterns,
		regexp.MustCompile(`(?:[A-Z]:\\|/)[^\s"']+`),
	)

	// SQL queries (truncate long ones)
	es.sensitivePatterns = append(es.sensitivePatterns,
		regexp.MustCompile(`\b(SELECT|INSERT|UPDATE|DELETE)\s+.*?\s+(FROM|VALUES|INTO)\b`),
	)
}

// addInternalPatterns registers patterns for internal-only information
func (es *ErrorSanitizer) addInternalPatterns() {
	// Stack traces and code locations
	es.internalPatterns = append(es.internalPatterns,
		regexp.MustCompile(`\s+at\s+\w+\.\w+\s*\([^)]*\)`),
		regexp.MustCompile(`/[^/]*\.go:\d+`),
	)

	// Function names and internal details
	es.internalPatterns = append(es.internalPatterns,
		regexp.MustCompile(`func\s+\w+`),
		regexp.MustCompile(`goroutine\s+\d+`),
	)
}

// Sanitize sanitizes an error for client consumption
func (es *ErrorSanitizer) Sanitize(err error) *SanitizedError {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	return es.SanitizeString(errStr)
}

// SanitizeString sanitizes an error string for client consumption
func (es *ErrorSanitizer) SanitizeString(errStr string) *SanitizedError {
	category := es.classifyError(errStr)
	severity := es.classifySeverity(errStr)
	sanitized := es.removeSensitiveInfo(errStr)
	code := es.generateErrorCode(category, errStr)
	isRetryable := es.isRetryable(category, errStr)

	return &SanitizedError{
		Code:            code,
		Message:         sanitized,
		Category:        category,
		Severity:        severity,
		IsRetryable:     isRetryable,
		InternalMessage: errStr,
	}
}

// removeSensitiveInfo removes sensitive information from error message
func (es *ErrorSanitizer) removeSensitiveInfo(errStr string) string {
	result := errStr

	// Remove sensitive patterns
	for _, pattern := range es.sensitivePatterns {
		result = pattern.ReplaceAllString(result, "[REDACTED]")
	}

	// Remove internal patterns
	for _, pattern := range es.internalPatterns {
		result = pattern.ReplaceAllString(result, "[INTERNAL]")
	}

	// Truncate if too long
	maxLen := 200
	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return strings.TrimSpace(result)
}

// classifyError classifies the error category
func (es *ErrorSanitizer) classifyError(errStr string) ErrorCategory {
	errStr = strings.ToLower(errStr)

	// Timeout errors (check before network to avoid confusion)
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") ||
		strings.Contains(errStr, "context canceled") {
		return ErrorCategoryTimeout
	}

	// Auth errors (check before network/permission keywords)
	if strings.Contains(errStr, "access denied") || strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "authentication failed") || strings.Contains(errStr, "password") ||
		strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "forbidden") {
		return ErrorCategoryAuth
	}

	// Network errors
	if strings.Contains(errStr, "can't connect") || strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") || strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "dial") || strings.Contains(errStr, "unreachable") {
		return ErrorCategoryNetwork
	}

	// System errors (before checking generic "resource" keyword)
	if strings.Contains(errStr, "out of memory") || strings.Contains(errStr, "memory") ||
		strings.Contains(errStr, "disk full") || strings.Contains(errStr, "disk space") ||
		strings.Contains(errStr, "resource limit") || strings.Contains(errStr, "too many") {
		return ErrorCategorySystem
	}

	// User errors (query, syntax, constraint violations)
	if strings.Contains(errStr, "syntax") || strings.Contains(errStr, "constraint") ||
		strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "invalid") || strings.Contains(errStr, "malformed") {
		return ErrorCategoryUser
	}

	// Fallback network for other connection issues
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "i/o") {
		return ErrorCategoryNetwork
	}

	// Default to internal
	return ErrorCategoryInternal
}

// classifySeverity classifies the error severity
func (es *ErrorSanitizer) classifySeverity(errStr string) ErrorSeverity {
	errStr = strings.ToLower(errStr)

	// Critical errors
	if strings.Contains(errStr, "fatal") || strings.Contains(errStr, "critical") ||
		strings.Contains(errStr, "panic") || strings.Contains(errStr, "crash") {
		return ErrorSeverityCritical
	}

	// Errors
	if strings.Contains(errStr, "error") || strings.Contains(errStr, "failed") ||
		strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") {
		return ErrorSeverityError
	}

	// Warnings
	if strings.Contains(errStr, "warning") || strings.Contains(errStr, "deprecated") ||
		strings.Contains(errStr, "retry") {
		return ErrorSeverityWarning
	}

	// Default to info
	return ErrorSeverityInfo
}

// isRetryable determines if an operation should be retried
func (es *ErrorSanitizer) isRetryable(category ErrorCategory, errStr string) bool {
	errStr = strings.ToLower(errStr)

	// Definitely retryable
	if category == ErrorCategoryNetwork || category == ErrorCategoryTimeout {
		return true
	}

	// Check for retryable keywords
	if strings.Contains(errStr, "temporary") || strings.Contains(errStr, "transient") ||
		strings.Contains(errStr, "unavailable") || strings.Contains(errStr, "busy") {
		return true
	}

	// Not retryable
	if category == ErrorCategoryAuth {
		return false
	}
	if category == ErrorCategoryUser {
		return false
	}

	// Default: retry for system and internal errors
	return category == ErrorCategorySystem || category == ErrorCategoryInternal
}

// generateErrorCode generates a machine-readable error code
func (es *ErrorSanitizer) generateErrorCode(category ErrorCategory, errStr string) string {
	baseCode := fmt.Sprintf("ERR_%s", strings.ToUpper(string(category)))

	// Add subcategory if detectable
	errStr = strings.ToLower(errStr)
	if strings.Contains(errStr, "timeout") {
		baseCode += "_TIMEOUT"
	} else if strings.Contains(errStr, "connection") {
		baseCode += "_CONNECTION"
	} else if strings.Contains(errStr, "syntax") {
		baseCode += "_SYNTAX"
	} else if strings.Contains(errStr, "constraint") {
		baseCode += "_CONSTRAINT"
	} else if strings.Contains(errStr, "duplicate") {
		baseCode += "_DUPLICATE"
	} else if strings.Contains(errStr, "not found") {
		baseCode += "_NOT_FOUND"
	} else if strings.Contains(errStr, "permission") {
		baseCode += "_PERMISSION"
	}

	return baseCode
}

// WithDetails adds client-safe details to the sanitized error
func (se *SanitizedError) WithDetails(key string, value interface{}) *SanitizedError {
	if se.Details == nil {
		se.Details = make(map[string]interface{})
	}
	se.Details[key] = value
	return se
}

// String returns the client-safe error message
func (se *SanitizedError) String() string {
	return se.Message
}

// Error implements the error interface with the client-safe message
func (se *SanitizedError) Error() string {
	return se.Message
}

// GetInternalMessage returns the full internal error (for logging only)
func (se *SanitizedError) GetInternalMessage() string {
	return se.InternalMessage
}

// ClientResponse returns a response suitable for sending to client
func (se *SanitizedError) ClientResponse() map[string]interface{} {
	return map[string]interface{}{
		"error":       se.Code,
		"message":     se.Message,
		"category":    string(se.Category),
		"severity":    string(se.Severity),
		"retryable":   se.IsRetryable,
		"details":     se.Details,
	}
}
