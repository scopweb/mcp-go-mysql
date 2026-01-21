package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// EventType categorizes the type of database operation
type EventType string

const (
	// EventTypeAuth for authentication/connection events
	EventTypeAuth EventType = "auth"
	// EventTypeQuery for SELECT operations
	EventTypeQuery EventType = "query"
	// EventTypeWrite for INSERT/UPDATE/DELETE operations
	EventTypeWrite EventType = "write"
	// EventTypeAdmin for DDL/administrative operations
	EventTypeAdmin EventType = "admin"
	// EventTypeSecurity for security violations and blocked queries
	EventTypeSecurity EventType = "security"
	// EventTypeError for database errors
	EventTypeError EventType = "error"
	// EventTypeConnection for connection lifecycle events
	EventTypeConnection EventType = "connection"
)

// OperationType categorizes specific SQL operations
type OperationType string

const (
	OpSelect   OperationType = "SELECT"
	OpInsert   OperationType = "INSERT"
	OpUpdate   OperationType = "UPDATE"
	OpDelete   OperationType = "DELETE"
	OpCreate   OperationType = "CREATE"
	OpDrop     OperationType = "DROP"
	OpAlter    OperationType = "ALTER"
	OpTruncate OperationType = "TRUNCATE"
	OpCall     OperationType = "CALL"
	OpOther    OperationType = "OTHER"
)

// Severity categorizes the severity of an event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// AuditEvent represents a logged database operation
type AuditEvent struct {
	// ID is a unique identifier for this event (UUID)
	ID string `json:"id"`
	// Timestamp is when the event occurred (ISO 8601)
	Timestamp time.Time `json:"timestamp"`
	// EventType categorizes the event
	EventType EventType `json:"event_type"`
	// Operation is the specific SQL operation
	Operation OperationType `json:"operation"`
	// User is the database user
	User string `json:"user"`
	// Database is the target database
	Database string `json:"database"`
	// Table is the target table (if applicable)
	Table string `json:"table,omitempty"`
	// Query is the SQL query (may be truncated or sanitized)
	Query string `json:"query,omitempty"`
	// RowsAffected is the number of rows modified
	RowsAffected int `json:"rows_affected"`
	// Duration is execution time in milliseconds
	Duration time.Duration `json:"duration_ms"`
	// Status is the result status (success, error, blocked, etc)
	Status string `json:"status"`
	// ErrorMsg is a sanitized error message
	ErrorMsg string `json:"error,omitempty"`
	// Source is the tool/component name
	Source string `json:"source"`
	// IPAddress is the remote client IP (if applicable)
	IPAddress string `json:"ip,omitempty"`
	// Severity is the event importance level
	Severity Severity `json:"severity"`
	// Additional fields for extensibility
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// String returns a JSON representation of the audit event
func (ae *AuditEvent) String() string {
	data, err := json.Marshal(ae)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal audit event: %v"}`, err)
	}
	return string(data)
}

// MarshalJSON implements json.Marshaler with proper formatting
func (ae *AuditEvent) MarshalJSON() ([]byte, error) {
	type Alias AuditEvent
	return json.Marshal(&struct {
		*Alias
		Timestamp string        `json:"timestamp"`
		Duration  int64         `json:"duration_ms"`
	}{
		Alias:     (*Alias)(ae),
		Timestamp: ae.Timestamp.UTC().Format(time.RFC3339Nano),
		Duration:  ae.Duration.Milliseconds(),
	})
}

// AuditLogger is the interface for audit logging implementations
type AuditLogger interface {
	// LogQuery logs a SELECT operation
	LogQuery(ctx context.Context, event *AuditEvent) error
	// LogWrite logs INSERT/UPDATE/DELETE operations
	LogWrite(ctx context.Context, event *AuditEvent) error
	// LogAdmin logs DDL operations
	LogAdmin(ctx context.Context, event *AuditEvent) error
	// LogError logs database errors
	LogError(ctx context.Context, event *AuditEvent) error
	// LogSecurity logs security violations
	LogSecurity(ctx context.Context, event *AuditEvent) error
	// Close closes the audit logger
	Close() error
}

// NoOpAuditLogger is a no-operation audit logger
type NoOpAuditLogger struct{}

func (n *NoOpAuditLogger) LogQuery(ctx context.Context, event *AuditEvent) error   { return nil }
func (n *NoOpAuditLogger) LogWrite(ctx context.Context, event *AuditEvent) error   { return nil }
func (n *NoOpAuditLogger) LogAdmin(ctx context.Context, event *AuditEvent) error   { return nil }
func (n *NoOpAuditLogger) LogError(ctx context.Context, event *AuditEvent) error   { return nil }
func (n *NoOpAuditLogger) LogSecurity(ctx context.Context, event *AuditEvent) error { return nil }
func (n *NoOpAuditLogger) Close() error                                             { return nil }

// InMemoryAuditLogger stores audit events in memory (for testing)
type InMemoryAuditLogger struct {
	events []*AuditEvent
	mu     sync.RWMutex
}

// NewInMemoryAuditLogger creates an in-memory audit logger
func NewInMemoryAuditLogger() *InMemoryAuditLogger {
	return &InMemoryAuditLogger{
		events: make([]*AuditEvent, 0),
	}
}

// LogQuery logs a SELECT operation
func (ial *InMemoryAuditLogger) LogQuery(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event cannot be nil")
	}
	ial.mu.Lock()
	defer ial.mu.Unlock()
	ial.events = append(ial.events, event)
	return nil
}

// LogWrite logs INSERT/UPDATE/DELETE operations
func (ial *InMemoryAuditLogger) LogWrite(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event cannot be nil")
	}
	ial.mu.Lock()
	defer ial.mu.Unlock()
	ial.events = append(ial.events, event)
	return nil
}

// LogAdmin logs DDL operations
func (ial *InMemoryAuditLogger) LogAdmin(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event cannot be nil")
	}
	ial.mu.Lock()
	defer ial.mu.Unlock()
	ial.events = append(ial.events, event)
	return nil
}

// LogError logs database errors
func (ial *InMemoryAuditLogger) LogError(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event cannot be nil")
	}
	ial.mu.Lock()
	defer ial.mu.Unlock()
	ial.events = append(ial.events, event)
	return nil
}

// LogSecurity logs security violations
func (ial *InMemoryAuditLogger) LogSecurity(ctx context.Context, event *AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event cannot be nil")
	}
	ial.mu.Lock()
	defer ial.mu.Unlock()
	ial.events = append(ial.events, event)
	return nil
}

// GetEvents returns all logged events
func (ial *InMemoryAuditLogger) GetEvents() []*AuditEvent {
	ial.mu.RLock()
	defer ial.mu.RUnlock()
	// Return a copy to prevent external modification
	events := make([]*AuditEvent, len(ial.events))
	copy(events, ial.events)
	return events
}

// Clear removes all logged events
func (ial *InMemoryAuditLogger) Clear() {
	ial.mu.Lock()
	defer ial.mu.Unlock()
	ial.events = make([]*AuditEvent, 0)
}

// Close closes the audit logger
func (ial *InMemoryAuditLogger) Close() error {
	return nil
}

// AuditEventBuilder is a fluent builder for constructing audit events
type AuditEventBuilder struct {
	event *AuditEvent
}

// NewAuditEvent creates a new audit event builder
func NewAuditEvent(eventType EventType) *AuditEventBuilder {
	return &AuditEventBuilder{
		event: &AuditEvent{
			EventType: eventType,
			Timestamp: time.Now().UTC(),
			Status:    "pending",
			Severity:  SeverityInfo,
			Metadata:  make(map[string]interface{}),
		},
	}
}

// WithID sets the event ID
func (aeb *AuditEventBuilder) WithID(id string) *AuditEventBuilder {
	aeb.event.ID = id
	return aeb
}

// WithOperation sets the SQL operation
func (aeb *AuditEventBuilder) WithOperation(op OperationType) *AuditEventBuilder {
	aeb.event.Operation = op
	return aeb
}

// WithUser sets the database user
func (aeb *AuditEventBuilder) WithUser(user string) *AuditEventBuilder {
	aeb.event.User = user
	return aeb
}

// WithDatabase sets the target database
func (aeb *AuditEventBuilder) WithDatabase(db string) *AuditEventBuilder {
	aeb.event.Database = db
	return aeb
}

// WithTable sets the target table
func (aeb *AuditEventBuilder) WithTable(table string) *AuditEventBuilder {
	aeb.event.Table = table
	return aeb
}

// WithQuery sets the SQL query
func (aeb *AuditEventBuilder) WithQuery(query string) *AuditEventBuilder {
	aeb.event.Query = query
	return aeb
}

// WithRowsAffected sets the affected row count
func (aeb *AuditEventBuilder) WithRowsAffected(rows int) *AuditEventBuilder {
	aeb.event.RowsAffected = rows
	return aeb
}

// WithDuration sets the execution duration
func (aeb *AuditEventBuilder) WithDuration(duration time.Duration) *AuditEventBuilder {
	aeb.event.Duration = duration
	return aeb
}

// WithStatus sets the operation status
func (aeb *AuditEventBuilder) WithStatus(status string) *AuditEventBuilder {
	aeb.event.Status = status
	return aeb
}

// WithError sets the error message and marks as error
func (aeb *AuditEventBuilder) WithError(errMsg string) *AuditEventBuilder {
	aeb.event.ErrorMsg = errMsg
	aeb.event.Status = "error"
	aeb.event.Severity = SeverityError
	return aeb
}

// WithSource sets the source/tool name
func (aeb *AuditEventBuilder) WithSource(source string) *AuditEventBuilder {
	aeb.event.Source = source
	return aeb
}

// WithIPAddress sets the remote IP address
func (aeb *AuditEventBuilder) WithIPAddress(ip string) *AuditEventBuilder {
	aeb.event.IPAddress = ip
	return aeb
}

// WithSeverity sets the event severity
func (aeb *AuditEventBuilder) WithSeverity(severity Severity) *AuditEventBuilder {
	aeb.event.Severity = severity
	return aeb
}

// WithMetadata adds metadata to the event
func (aeb *AuditEventBuilder) WithMetadata(key string, value interface{}) *AuditEventBuilder {
	if aeb.event.Metadata == nil {
		aeb.event.Metadata = make(map[string]interface{})
	}
	aeb.event.Metadata[key] = value
	return aeb
}

// Build returns the constructed audit event
func (aeb *AuditEventBuilder) Build() *AuditEvent {
	return aeb.event
}

// ContextKeyAuditLogger is a context key for audit logger
type contextKeyAuditLogger struct{}

// WithAuditLogger attaches an audit logger to a context
func WithAuditLogger(ctx context.Context, logger AuditLogger) context.Context {
	return context.WithValue(ctx, contextKeyAuditLogger{}, logger)
}

// GetAuditLogger retrieves an audit logger from a context
func GetAuditLogger(ctx context.Context) AuditLogger {
	if logger, ok := ctx.Value(contextKeyAuditLogger{}).(AuditLogger); ok {
		return logger
	}
	return &NoOpAuditLogger{}
}
