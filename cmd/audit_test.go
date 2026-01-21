package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mysql "mcp-gp-mysql/internal"
)

// TestAuditEventMarshal verifies JSON serialization
func TestAuditEventMarshal(t *testing.T) {
	event := &mysql.AuditEvent{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:   time.Date(2026, 1, 21, 10, 30, 45, 123000000, time.UTC),
		EventType:   mysql.EventTypeQuery,
		Operation:   mysql.OpSelect,
		User:        "app_user",
		Database:    "myapp",
		Table:       "users",
		Query:       "SELECT id, name FROM users WHERE id = ?",
		RowsAffected: 1,
		Duration:    25 * time.Millisecond,
		Status:      "success",
		Source:      "mcp-mysql",
		Severity:    mysql.SeverityInfo,
	}

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		t.Errorf("Failed to marshal event: %v", err)
		return
	}

	// Verify JSON is valid
	var unmarshalled map[string]interface{}
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
		return
	}

	// Verify key fields are present
	requiredFields := []string{"id", "timestamp", "event_type", "operation", "user", "database", "status"}
	for _, field := range requiredFields {
		if _, exists := unmarshalled[field]; !exists {
			t.Errorf("Field %s missing from JSON", field)
		}
	}
}

// TestAuditEventString verifies string representation
func TestAuditEventString(t *testing.T) {
	event := &mysql.AuditEvent{
		ID:        "test-id",
		EventType: mysql.EventTypeQuery,
		Operation: mysql.OpSelect,
		Status:    "success",
	}

	str := event.String()
	if str == "" {
		t.Errorf("Event string representation should not be empty")
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	err := json.Unmarshal([]byte(str), &data)
	if err != nil {
		t.Errorf("Event string should be valid JSON: %v", err)
	}
}

// TestNewAuditEvent verifies event creation
func TestNewAuditEvent(t *testing.T) {
	event := mysql.NewAuditEvent(mysql.EventTypeQuery).Build()

	if event == nil {
		t.Errorf("Event should not be nil")
	}

	if event.EventType != mysql.EventTypeQuery {
		t.Errorf("Event type mismatch: expected %s, got %s", mysql.EventTypeQuery, event.EventType)
	}

	if event.Timestamp.IsZero() {
		t.Errorf("Timestamp should be set")
	}

	if event.Status != "pending" {
		t.Errorf("Default status should be 'pending', got %s", event.Status)
	}
}

// TestAuditEventBuilder verifies fluent builder pattern
func TestAuditEventBuilder(t *testing.T) {
	event := mysql.NewAuditEvent(mysql.EventTypeQuery).
		WithID("test-id").
		WithOperation(mysql.OpSelect).
		WithUser("testuser").
		WithDatabase("testdb").
		WithTable("users").
		WithQuery("SELECT * FROM users").
		WithRowsAffected(5).
		WithDuration(100 * time.Millisecond).
		WithStatus("success").
		WithSource("mcp-mysql").
		WithSeverity(mysql.SeverityInfo).
		WithMetadata("test_key", "test_value").
		Build()

	if event.ID != "test-id" {
		t.Errorf("ID mismatch")
	}

	if event.Operation != mysql.OpSelect {
		t.Errorf("Operation mismatch")
	}

	if event.User != "testuser" {
		t.Errorf("User mismatch")
	}

	if event.Database != "testdb" {
		t.Errorf("Database mismatch")
	}

	if event.Table != "users" {
		t.Errorf("Table mismatch")
	}

	if event.Query != "SELECT * FROM users" {
		t.Errorf("Query mismatch")
	}

	if event.RowsAffected != 5 {
		t.Errorf("RowsAffected mismatch: expected 5, got %d", event.RowsAffected)
	}

	if event.Duration != 100*time.Millisecond {
		t.Errorf("Duration mismatch")
	}

	if event.Status != "success" {
		t.Errorf("Status mismatch")
	}

	if event.Source != "mcp-mysql" {
		t.Errorf("Source mismatch")
	}

	if event.Severity != mysql.SeverityInfo {
		t.Errorf("Severity mismatch")
	}

	if val, exists := event.Metadata["test_key"]; !exists || val != "test_value" {
		t.Errorf("Metadata not set correctly")
	}
}

// TestInMemoryAuditLogger verifies in-memory storage
func TestInMemoryAuditLogger(t *testing.T) {
	logger := mysql.NewInMemoryAuditLogger()
	ctx := context.Background()

	event1 := mysql.NewAuditEvent(mysql.EventTypeQuery).
		WithID("event-1").
		WithOperation(mysql.OpSelect).
		WithStatus("success").
		Build()

	event2 := mysql.NewAuditEvent(mysql.EventTypeWrite).
		WithID("event-2").
		WithOperation(mysql.OpInsert).
		WithStatus("success").
		Build()

	// Log events
	err := logger.LogQuery(ctx, event1)
	if err != nil {
		t.Errorf("LogQuery failed: %v", err)
	}

	err = logger.LogWrite(ctx, event2)
	if err != nil {
		t.Errorf("LogWrite failed: %v", err)
	}

	// Retrieve events
	events := logger.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].ID != "event-1" {
		t.Errorf("Event 1 ID mismatch")
	}

	if events[1].ID != "event-2" {
		t.Errorf("Event 2 ID mismatch")
	}
}

// TestInMemoryAuditLoggerNilEvent verifies nil handling
func TestInMemoryAuditLoggerNilEvent(t *testing.T) {
	logger := mysql.NewInMemoryAuditLogger()
	ctx := context.Background()

	// Try to log nil event
	err := logger.LogQuery(ctx, nil)
	if err == nil {
		t.Errorf("Should return error for nil event")
	}
}

// TestInMemoryAuditLoggerClear verifies event clearing
func TestInMemoryAuditLoggerClear(t *testing.T) {
	logger := mysql.NewInMemoryAuditLogger()
	ctx := context.Background()

	event := mysql.NewAuditEvent(mysql.EventTypeQuery).Build()
	logger.LogQuery(ctx, event)

	if len(logger.GetEvents()) != 1 {
		t.Errorf("Should have 1 event before clear")
	}

	logger.Clear()

	if len(logger.GetEvents()) != 0 {
		t.Errorf("Should have 0 events after clear")
	}
}

// TestNoOpAuditLogger verifies no-op implementation
func TestNoOpAuditLogger(t *testing.T) {
	logger := &mysql.NoOpAuditLogger{}
	ctx := context.Background()

	event := mysql.NewAuditEvent(mysql.EventTypeQuery).Build()

	// These should all succeed silently
	err := logger.LogQuery(ctx, event)
	if err != nil {
		t.Errorf("NoOpAuditLogger.LogQuery should not error: %v", err)
	}

	err = logger.LogWrite(ctx, event)
	if err != nil {
		t.Errorf("NoOpAuditLogger.LogWrite should not error: %v", err)
	}

	err = logger.LogAdmin(ctx, event)
	if err != nil {
		t.Errorf("NoOpAuditLogger.LogAdmin should not error: %v", err)
	}

	err = logger.LogError(ctx, event)
	if err != nil {
		t.Errorf("NoOpAuditLogger.LogError should not error: %v", err)
	}

	err = logger.LogSecurity(ctx, event)
	if err != nil {
		t.Errorf("NoOpAuditLogger.LogSecurity should not error: %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("NoOpAuditLogger.Close should not error: %v", err)
	}
}

// TestAuditEventTypes verifies event type enumeration
func TestAuditEventTypes(t *testing.T) {
	tests := []struct {
		eventType mysql.EventType
		name      string
	}{
		{mysql.EventTypeAuth, "auth"},
		{mysql.EventTypeQuery, "query"},
		{mysql.EventTypeWrite, "write"},
		{mysql.EventTypeAdmin, "admin"},
		{mysql.EventTypeSecurity, "security"},
		{mysql.EventTypeError, "error"},
		{mysql.EventTypeConnection, "connection"},
	}

	for _, tt := range tests {
		if string(tt.eventType) != tt.name {
			t.Errorf("EventType %s has unexpected value %s", tt.name, tt.eventType)
		}
	}
}

// TestAuditOperationTypes verifies operation type enumeration
func TestAuditOperationTypes(t *testing.T) {
	tests := []struct {
		op   mysql.OperationType
		name string
	}{
		{mysql.OpSelect, "SELECT"},
		{mysql.OpInsert, "INSERT"},
		{mysql.OpUpdate, "UPDATE"},
		{mysql.OpDelete, "DELETE"},
		{mysql.OpCreate, "CREATE"},
		{mysql.OpDrop, "DROP"},
		{mysql.OpAlter, "ALTER"},
		{mysql.OpTruncate, "TRUNCATE"},
		{mysql.OpCall, "CALL"},
		{mysql.OpOther, "OTHER"},
	}

	for _, tt := range tests {
		if string(tt.op) != tt.name {
			t.Errorf("OperationType %s has unexpected value %s", tt.name, tt.op)
		}
	}
}

// TestAuditSeverityTypes verifies severity enumeration
func TestAuditSeverityTypes(t *testing.T) {
	tests := []struct {
		severity mysql.Severity
		name     string
	}{
		{mysql.SeverityInfo, "info"},
		{mysql.SeverityWarning, "warning"},
		{mysql.SeverityError, "error"},
		{mysql.SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.name {
			t.Errorf("Severity %s has unexpected value %s", tt.name, tt.severity)
		}
	}
}

// TestAuditEventBuilderWithError verifies error handling
func TestAuditEventBuilderWithError(t *testing.T) {
	event := mysql.NewAuditEvent(mysql.EventTypeError).
		WithError("connection refused").
		Build()

	if event.Status != "error" {
		t.Errorf("Status should be 'error', got %s", event.Status)
	}

	if event.ErrorMsg != "connection refused" {
		t.Errorf("ErrorMsg mismatch")
	}

	if event.Severity != mysql.SeverityError {
		t.Errorf("Severity should be error")
	}
}

// TestContextWithAuditLogger verifies context integration
func TestContextWithAuditLogger(t *testing.T) {
	logger := mysql.NewInMemoryAuditLogger()
	ctx := context.Background()

	// Attach logger to context
	ctxWithLogger := mysql.WithAuditLogger(ctx, logger)

	// Retrieve logger from context
	retrieved := mysql.GetAuditLogger(ctxWithLogger)
	if retrieved == nil {
		t.Errorf("Should retrieve audit logger from context")
	}

	// Verify it's the same logger
	if retrieved != logger {
		t.Errorf("Retrieved logger should be the same instance")
	}

	// Verify original context has no logger
	defaultLogger := mysql.GetAuditLogger(ctx)
	if _, ok := defaultLogger.(*mysql.NoOpAuditLogger); !ok {
		t.Errorf("Original context should have NoOpAuditLogger")
	}
}

// TestAuditEventJSONFormatting verifies proper JSON formatting
func TestAuditEventJSONFormatting(t *testing.T) {
	timestamp := time.Date(2026, 1, 21, 10, 30, 45, 123456789, time.UTC)
	event := &mysql.AuditEvent{
		ID:        "test-id",
		Timestamp: timestamp,
		EventType: mysql.EventTypeQuery,
		Operation: mysql.OpSelect,
		User:      "testuser",
		Database:  "testdb",
		Status:    "success",
		Duration:  100 * time.Millisecond,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Errorf("Failed to marshal: %v", err)
		return
	}

	var unmarshalled map[string]interface{}
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Errorf("Failed to unmarshal: %v", err)
		return
	}

	// Verify timestamp format (ISO 8601)
	if timestampStr, ok := unmarshalled["timestamp"].(string); ok {
		// Should contain 'T' (ISO 8601 format)
		if !containsAuditString(timestampStr, "T") {
			t.Errorf("Timestamp should be ISO 8601 format, got %s", timestampStr)
		}
	} else {
		t.Errorf("Timestamp should be a string")
	}

	// Verify duration is in milliseconds
	if durationMs, ok := unmarshalled["duration_ms"].(float64); ok {
		if durationMs != 100 {
			t.Errorf("Duration should be 100ms, got %v", durationMs)
		}
	} else {
		t.Errorf("Duration should be a number")
	}
}

// TestAuditEventMetadataExtensibility verifies metadata support
func TestAuditEventMetadataExtensibility(t *testing.T) {
	event := mysql.NewAuditEvent(mysql.EventTypeQuery).
		WithMetadata("request_id", "req-123").
		WithMetadata("session_id", "sess-456").
		WithMetadata("timeout_occurred", true).
		WithMetadata("rows_scanned", 1000).
		Build()

	if event.Metadata == nil {
		t.Errorf("Metadata should be initialized")
		return
	}

	if event.Metadata["request_id"] != "req-123" {
		t.Errorf("String metadata not set correctly")
	}

	if event.Metadata["session_id"] != "sess-456" {
		t.Errorf("String metadata not set correctly")
	}

	if val, ok := event.Metadata["timeout_occurred"].(bool); !ok || val != true {
		t.Errorf("Boolean metadata not set correctly")
	}

	if val, ok := event.Metadata["rows_scanned"].(int); !ok || val != 1000 {
		t.Errorf("Integer metadata not set correctly")
	}
}

// containsAuditString is a helper to check string containment
func containsAuditString(str, substr string) bool {
	for i := 0; i < len(str)-len(substr)+1; i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
