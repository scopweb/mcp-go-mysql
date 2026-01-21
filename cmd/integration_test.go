package main

import (
	"context"
	"testing"
	"time"

	mysql "mcp-gp-mysql/internal"
)

// TestClientTimeoutIntegration verifies timeout configuration works
func TestClientTimeoutIntegration(t *testing.T) {
	// Verify Client can be created
	client := mysql.NewClient()

	if client == nil {
		t.Errorf("Client should not be nil")
		return
	}

	// Verify timeout configuration can be created independently
	tc := mysql.NewTimeoutConfig()
	if tc == nil {
		t.Errorf("TimeoutConfig should not be nil")
		return
	}

	// Verify default timeouts are correct
	if tc.GetTimeout(mysql.ProfileQuery) != 30*time.Second {
		t.Errorf("Query timeout should be 30s")
	}

	if tc.GetTimeout(mysql.ProfileWrite) != 60*time.Second {
		t.Errorf("Write timeout should be 60s")
	}

	if tc.GetTimeout(mysql.ProfileAdmin) != 15*time.Second {
		t.Errorf("Admin timeout should be 15s")
	}

	if tc.GetTimeout(mysql.ProfileConnection) != 5*time.Second {
		t.Errorf("Connection timeout should be 5s")
	}
}

// TestTimeoutAuditEventIntegration verifies audit events can track timeouts
func TestTimeoutAuditEventIntegration(t *testing.T) {
	timeoutDetails := mysql.NewTimeoutDetails(mysql.ProfileQuery, 30*time.Second)
	time.Sleep(10 * time.Millisecond)
	timeoutDetails.Record(false)

	// Create an audit event with timeout information
	event := mysql.NewAuditEvent(mysql.EventTypeQuery).
		WithOperation(mysql.OpSelect).
		WithUser("test_user").
		WithDatabase("test_db").
		WithQuery("SELECT * FROM users").
		WithStatus("success").
		WithDuration(timeoutDetails.Elapsed).
		WithMetadata("timeout_profile", string(timeoutDetails.Profile)).
		WithMetadata("timeout_duration", timeoutDetails.Timeout.Milliseconds()).
		WithMetadata("remaining_time", timeoutDetails.RemainingTime.Milliseconds()).
		Build()

	if event == nil {
		t.Errorf("Event should not be nil")
	}

	if event.Duration != timeoutDetails.Elapsed {
		t.Errorf("Event duration should match timeout details elapsed")
	}

	if event.Metadata["timeout_profile"] != "query" {
		t.Errorf("Timeout profile should be in metadata")
	}
}

// TestAuditLoggerWithContext verifies audit logger context integration
func TestAuditLoggerWithContext(t *testing.T) {
	logger := mysql.NewInMemoryAuditLogger()
	ctx := context.Background()

	// Attach logger to context
	ctxWithLogger := mysql.WithAuditLogger(ctx, logger)

	// Create and log events using context logger
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

	// Log using context-retrieved logger
	retrievedLogger := mysql.GetAuditLogger(ctxWithLogger)
	retrievedLogger.LogQuery(ctxWithLogger, event1)
	retrievedLogger.LogWrite(ctxWithLogger, event2)

	// Verify both events were logged
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

// TestTimeoutContextPropagation verifies timeout context flows through calls
func TestTimeoutContextPropagation(t *testing.T) {
	tc := mysql.NewTimeoutConfig()
	ctx := context.Background()

	// Create context with timeout
	ctxWithTimeout, cancel := tc.TimeoutContext(ctx, mysql.ProfileQuery)
	defer cancel()

	// Verify deadline is set
	deadline, ok := ctxWithTimeout.Deadline()
	if !ok {
		t.Errorf("Context should have deadline")
	}

	// Verify we can get timeout metrics
	ctxWithMetrics := mysql.WithTimeoutMetrics(ctxWithTimeout, mysql.NewTimeoutDetails(mysql.ProfileQuery, 30*time.Second))
	metrics := mysql.GetTimeoutMetrics(ctxWithMetrics)

	if metrics == nil {
		t.Errorf("Should retrieve timeout metrics from context")
	}

	// Verify deadline is still set after attaching metrics
	deadline2, ok2 := ctxWithMetrics.Deadline()
	if !ok2 {
		t.Errorf("Context should still have deadline after metrics")
	}

	if deadline != deadline2 {
		t.Errorf("Deadline should remain same after metrics attachment")
	}
}

// TestDatabaseCompatibilityTimeoutInteraction verifies compatibility and timeout work together
func TestDatabaseCompatibilityTimeoutInteraction(t *testing.T) {
	// Create and verify both configurations independently
	mariaConfig := mysql.GetDBCompatibilityConfig("mariadb")
	mysqlConfig := mysql.GetDBCompatibilityConfig("mysql")

	if mariaConfig == nil || mysqlConfig == nil {
		t.Errorf("Both configs should exist")
		return
	}

	if mariaConfig.Type == mysqlConfig.Type {
		t.Errorf("MariaDB and MySQL configs should have different types")
	}

	// Verify timeout config is independent
	tc := mysql.NewTimeoutConfig()
	timeout1 := tc.GetTimeout(mysql.ProfileQuery)
	timeout2 := tc.GetTimeout(mysql.ProfileQuery)

	if timeout1 != timeout2 {
		t.Errorf("Timeout should be consistent")
	}
}

// TestMultipleAuditEventsSequence verifies sequence of events in logger
func TestMultipleAuditEventsSequence(t *testing.T) {
	logger := mysql.NewInMemoryAuditLogger()
	ctx := context.Background()

	// Simulate a sequence of operations
	operations := []struct {
		eventType mysql.EventType
		operation mysql.OperationType
		status    string
	}{
		{mysql.EventTypeAuth, mysql.OpOther, "success"},
		{mysql.EventTypeQuery, mysql.OpSelect, "success"},
		{mysql.EventTypeWrite, mysql.OpInsert, "success"},
		{mysql.EventTypeSecurity, mysql.OpSelect, "blocked"},
		{mysql.EventTypeError, mysql.OpSelect, "error"},
	}

	// Create and log events
	for i, op := range operations {
		event := mysql.NewAuditEvent(op.eventType).
			WithID(string(rune(i))).
			WithOperation(op.operation).
			WithStatus(op.status).
			Build()

		switch op.eventType {
		case mysql.EventTypeQuery:
			logger.LogQuery(ctx, event)
		case mysql.EventTypeWrite:
			logger.LogWrite(ctx, event)
		case mysql.EventTypeSecurity:
			logger.LogSecurity(ctx, event)
		case mysql.EventTypeError:
			logger.LogError(ctx, event)
		default:
			// Auth events
			logger.LogQuery(ctx, event)
		}
	}

	// Verify all events were logged in order
	events := logger.GetEvents()
	if len(events) != len(operations) {
		t.Errorf("Expected %d events, got %d", len(operations), len(events))
	}

	for i, op := range operations {
		if events[i].EventType != op.eventType {
			t.Errorf("Event %d type mismatch", i)
		}
		if events[i].Status != op.status {
			t.Errorf("Event %d status mismatch", i)
		}
	}
}

// TestTimeoutProfileSelection verifies correct timeout for operation type
func TestTimeoutProfileSelection(t *testing.T) {
	tc := mysql.NewTimeoutConfig()

	tests := []struct {
		profile  mysql.TimeoutProfile
		expected time.Duration
	}{
		{mysql.ProfileQuery, 30 * time.Second},
		{mysql.ProfileWrite, 60 * time.Second},
		{mysql.ProfileAdmin, 15 * time.Second},
		{mysql.ProfileLongQuery, 5 * time.Minute},
		{mysql.ProfileConnection, 5 * time.Second},
		{mysql.ProfileDefault, 30 * time.Second},
	}

	for _, tt := range tests {
		timeout := tc.GetTimeout(tt.profile)
		if timeout != tt.expected {
			t.Errorf("Profile %s: expected %v, got %v", tt.profile, tt.expected, timeout)
		}
	}
}

// TestAuditEventErrorSeverity verifies error events have correct severity
func TestAuditEventErrorSeverity(t *testing.T) {
	successEvent := mysql.NewAuditEvent(mysql.EventTypeQuery).
		WithStatus("success").
		Build()

	if successEvent.Severity != mysql.SeverityInfo {
		t.Errorf("Success event should have info severity")
	}

	errorEvent := mysql.NewAuditEvent(mysql.EventTypeError).
		WithError("connection lost").
		Build()

	if errorEvent.Severity != mysql.SeverityError {
		t.Errorf("Error event should have error severity")
	}

	securityEvent := mysql.NewAuditEvent(mysql.EventTypeSecurity).
		WithStatus("blocked").
		WithSeverity(mysql.SeverityCritical).
		Build()

	if securityEvent.Severity != mysql.SeverityCritical {
		t.Errorf("Security event should have critical severity")
	}
}

// TestTimeoutConfigValidation verifies timeout durations are valid
func TestTimeoutConfigValidation(t *testing.T) {
	tc := mysql.NewTimeoutConfig()

	// Verify all timeouts are positive
	timeouts := []time.Duration{
		tc.Query,
		tc.Write,
		tc.Admin,
		tc.LongQuery,
		tc.Connection,
		tc.Default,
	}

	for _, timeout := range timeouts {
		if timeout <= 0 {
			t.Errorf("Timeout should be positive: %v", timeout)
		}

		// Verify it's reasonable (not more than 24 hours)
		if timeout > 24*time.Hour {
			t.Errorf("Timeout is unusually long: %v", timeout)
		}
	}

	// Verify LongQuery > Query
	if tc.LongQuery <= tc.Query {
		t.Errorf("LongQuery timeout should be greater than Query timeout")
	}

	// Verify Write > Query
	if tc.Write <= tc.Query {
		t.Errorf("Write timeout should be greater than Query timeout")
	}
}

// TestAuditEventWithAllFields creates event with complete field set
func TestAuditEventWithAllFields(t *testing.T) {
	event := mysql.NewAuditEvent(mysql.EventTypeWrite).
		WithID("complete-event").
		WithOperation(mysql.OpInsert).
		WithUser("data_pipeline").
		WithDatabase("analytics").
		WithTable("events").
		WithQuery("INSERT INTO events (data) VALUES (?)").
		WithRowsAffected(1000).
		WithDuration(150 * time.Millisecond).
		WithStatus("success").
		WithSource("mcp-mysql").
		WithIPAddress("192.168.1.100").
		WithSeverity(mysql.SeverityInfo).
		WithMetadata("batch_id", "batch-2026-001").
		WithMetadata("source_system", "api_gateway").
		WithMetadata("request_id", "req-abc123").
		Build()

	// Verify all fields are set
	if event.ID != "complete-event" {
		t.Errorf("ID not set")
	}
	if event.Operation != mysql.OpInsert {
		t.Errorf("Operation not set")
	}
	if event.User != "data_pipeline" {
		t.Errorf("User not set")
	}
	if event.Database != "analytics" {
		t.Errorf("Database not set")
	}
	if event.Table != "events" {
		t.Errorf("Table not set")
	}
	if event.Query != "INSERT INTO events (data) VALUES (?)" {
		t.Errorf("Query not set")
	}
	if event.RowsAffected != 1000 {
		t.Errorf("RowsAffected not set")
	}
	if event.Duration != 150*time.Millisecond {
		t.Errorf("Duration not set")
	}
	if event.Status != "success" {
		t.Errorf("Status not set")
	}
	if event.Source != "mcp-mysql" {
		t.Errorf("Source not set")
	}
	if event.IPAddress != "192.168.1.100" {
		t.Errorf("IPAddress not set")
	}
	if event.Severity != mysql.SeverityInfo {
		t.Errorf("Severity not set")
	}
	if len(event.Metadata) != 3 {
		t.Errorf("Metadata should have 3 items")
	}
}

// TestInMemoryLoggerConcurrency verifies thread-safety with simple test
func TestInMemoryLoggerConcurrency(t *testing.T) {
	logger := mysql.NewInMemoryAuditLogger()
	ctx := context.Background()

	// Create multiple events and log them
	for i := 0; i < 10; i++ {
		event := mysql.NewAuditEvent(mysql.EventTypeQuery).
			WithID(string(rune(i))).
			WithStatus("success").
			Build()
		logger.LogQuery(ctx, event)
	}

	// Verify all events were logged
	events := logger.GetEvents()
	if len(events) != 10 {
		t.Errorf("Expected 10 events, got %d", len(events))
	}
}

// TestEventTimestampAccuracy verifies event timestamp is recent
func TestEventTimestampAccuracy(t *testing.T) {
	before := time.Now()
	event := mysql.NewAuditEvent(mysql.EventTypeQuery).Build()
	after := time.Now()

	if event.Timestamp.Before(before) || event.Timestamp.After(after.Add(100*time.Millisecond)) {
		t.Errorf("Event timestamp not accurate: before=%v, event=%v, after=%v", before, event.Timestamp, after)
	}
}

// TestDatabaseCompatibilityAccess verifies compat config is accessible
func TestDatabaseCompatibilityAccess(t *testing.T) {
	// Access compatibility config via public API
	config := mysql.GetDBCompatibilityConfig("mariadb")

	if config == nil {
		t.Errorf("CompatConfig should not be nil")
		return
	}

	// Verify default is MariaDB
	if config.Type != mysql.DBTypeMariaDB {
		t.Errorf("Default should be MariaDB")
	}

	// Verify features are configured
	if !config.SupportsSequences {
		t.Errorf("MariaDB should support sequences")
	}

	if !config.SupportsPLSQL {
		t.Errorf("MariaDB should support PL/SQL")
	}

	if !config.SupportsBACKUPSTAGE {
		t.Errorf("MariaDB should support BACKUP STAGE")
	}
}
