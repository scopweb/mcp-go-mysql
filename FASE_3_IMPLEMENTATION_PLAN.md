# FASE 3: Advanced Security & Monitoring Implementation Plan

**Status:** In Progress
**Target:** January 2026
**Priority:** HIGH (Security & Production Readiness)

---

## 📋 Overview

FASE 3 focuses on advanced security features and monitoring to meet production-grade requirements:

1. **Context Timeout Management** - Per-operation timeout control
2. **JSON Audit Logging** - Structured audit trail for compliance
3. **Rate Limiting** - DoS protection through connection pool management
4. **Error Sanitization** - Prevent information disclosure

---

## 🎯 FASE 3.1: Context Timeout Management

### Current State
- ✅ Basic `context.WithTimeout()` implemented in Query/Execute methods
- ✅ Default 30-second timeout in DatabaseConfig
- ⚠️ No operation-specific timeout customization
- ⚠️ No timeout-based metrics/logging

### Goals
- [ ] Create TimeoutConfig with operation-specific timeouts
- [ ] Implement timeout override for critical operations
- [ ] Add timeout metrics and logging
- [ ] Create timeout-aware query builder

### Implementation Details

#### Files to Modify
- `internal/client.go` - Add timeout management
- `internal/timeout.go` (NEW) - Timeout utilities

#### Code Structure
```go
// TimeoutConfig manages operation-specific timeouts
type TimeoutConfig struct {
    Default      time.Duration  // 30s
    Query        time.Duration  // 30s
    LongQuery    time.Duration  // 5m (for complex queries)
    Write        time.Duration  // 60s (more conservative)
    AdminOps     time.Duration  // 15s (DDL operations)
    Connection   time.Duration  // 5s (connection establishment)
}

// TimeoutProfile identifies the operation type for timeout selection
type TimeoutProfile string
const (
    ProfileDefault TimeoutProfile = "default"
    ProfileQuery   TimeoutProfile = "query"
    ProfileWrite   TimeoutProfile = "write"
    ProfileAdmin   TimeoutProfile = "admin"
    ProfileLong    TimeoutProfile = "long"
)

// GetTimeoutForProfile returns appropriate timeout
func (c *Client) GetTimeoutForProfile(profile TimeoutProfile) time.Duration
```

#### Operations Requiring Different Timeouts
| Operation | Profile | Timeout | Reason |
|-----------|---------|---------|--------|
| SELECT | query | 30s | Standard queries |
| SELECT complex | long | 5m | Large aggregations |
| INSERT/UPDATE/DELETE | write | 60s | Conservative |
| CREATE/DROP/ALTER | admin | 15s | Lock avoidance |
| Connection test | connection | 5s | Quick detection |
| Administrative | admin | 15s | Safety margin |

### Tests to Create
- `TestTimeoutProfiles` - Verify each profile timeout
- `TestTimeoutExceeded` - Verify context cancellation
- `TestTimeoutMetrics` - Verify metric collection

---

## 🎯 FASE 3.2: JSON Audit Logging

### Current State
- ✅ Basic logging in cmd/main.go
- ⚠️ No structured audit trail
- ⚠️ No compliance logging
- ⚠️ No event categorization

### Goals
- [ ] Create AuditLogger with JSON output
- [ ] Log all database operations
- [ ] Track user/authentication context
- [ ] Generate audit reports

### Implementation Details

#### Files to Create
- `internal/audit.go` - Audit logging framework
- `internal/audit_logger.go` - JSON logger implementation

#### Audit Event Structure
```go
// AuditEvent represents a logged database operation
type AuditEvent struct {
    ID         string            `json:"id"`              // UUID
    Timestamp  time.Time         `json:"timestamp"`       // ISO 8601
    EventType  string            `json:"event_type"`      // query, write, admin, etc
    Operation  string            `json:"operation"`       // SELECT, INSERT, UPDATE, etc
    User       string            `json:"user"`            // DB user
    Database   string            `json:"database"`        // Target database
    Table      string            `json:"table,omitempty"` // Target table
    Query      string            `json:"query,omitempty"` // SQL query
    RowsAffected int             `json:"rows_affected"`   // Rows modified
    Duration   time.Duration     `json:"duration_ms"`     // Execution time
    Status     string            `json:"status"`          // success, error, blocked
    ErrorMsg   string            `json:"error,omitempty"` // Sanitized error
    Source     string            `json:"source"`          // MCP/tool name
    IPAddress  string            `json:"ip,omitempty"`    // Remote IP
    Severity   string            `json:"severity"`        // info, warning, error, critical
}

// AuditLogger interface
type AuditLogger interface {
    LogQuery(ctx context.Context, event *AuditEvent) error
    LogWrite(ctx context.Context, event *AuditEvent) error
    LogAdmin(ctx context.Context, event *AuditEvent) error
    LogError(ctx context.Context, event *AuditEvent) error
    LogSecurityEvent(ctx context.Context, event *AuditEvent) error
}

// JSONAuditLogger implements AuditLogger with JSON output
type JSONAuditLogger struct {
    writer io.WriteCloser
    mu     sync.Mutex
}
```

#### Event Types to Log
- `auth` - Connection/authentication events
- `query` - SELECT operations
- `write` - INSERT/UPDATE/DELETE operations
- `admin` - DDL/administrative operations
- `security` - Security violations, blocked queries
- `error` - Database errors
- `connection` - Connection lifecycle

#### Audit Log Format
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-01-21T10:30:45.123Z",
  "event_type": "query",
  "operation": "SELECT",
  "user": "app_user",
  "database": "myapp",
  "table": "users",
  "query": "SELECT id, name FROM users WHERE id = ?",
  "rows_affected": 1,
  "duration_ms": 25,
  "status": "success",
  "source": "mcp-mysql",
  "severity": "info"
}
```

### Tests to Create
- `TestAuditEventMarshal` - JSON serialization
- `TestAuditLogQuery` - Query event logging
- `TestAuditLogWrite` - Write event logging
- `TestAuditLogSecurity` - Security event logging
- `TestAuditLogRotation` - Log file rotation

---

## 🎯 FASE 3.3: Rate Limiting & Connection Pool Management

### Current State
- ✅ Connection pool configured (10 max, 5 idle)
- ⚠️ No rate limiting per operation
- ⚠️ No load-based throttling
- ⚠️ No backpressure handling

### Goals
- [ ] Implement operation-level rate limiting
- [ ] Add token bucket algorithm
- [ ] Implement backpressure handling
- [ ] Create rate limit metrics

### Implementation Details

#### Files to Create
- `internal/ratelimit.go` - Rate limiting framework

#### Rate Limiter Structure
```go
// RateLimiter provides token bucket-based rate limiting
type RateLimiter struct {
    queriesPerSecond   int       // Max queries/second
    writesPerSecond    int       // Max writes/second
    adminPerSecond     int       // Max admin ops/second
    tokens             float64   // Current tokens
    lastRefill         time.Time // Last refill time
    mu                 sync.RWMutex
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
    QueriesPerSecond   int           // 1000/s default
    WritesPerSecond    int           // 100/s default
    AdminPerSecond     int           // 10/s default
    BackpressureDelay  time.Duration // 100ms
    MaxQueuedOps       int           // 500
}

// AcquireToken attempts to acquire a token for operation
func (rl *RateLimiter) AcquireToken(op OperationType) error
```

#### Connection Pool Tuning
```go
// GetOptimalPoolSize returns tuning for database
func GetOptimalPoolSize(dbType DatabaseType) PoolConfig {
    switch dbType {
    case DBTypeMariaDB:
        return PoolConfig{
            MaxOpenConns:   GetEnvOrInt("DB_MAX_CONNS", 15),
            MaxIdleConns:   GetEnvOrInt("DB_MAX_IDLE", 8),
            ConnMaxLife:    1 * time.Hour,
            ConnMaxIdleTime: 15 * time.Minute,
        }
    case DBTypeMySQL:
        return PoolConfig{
            MaxOpenConns:   GetEnvOrInt("DB_MAX_CONNS", 10),
            MaxIdleConns:   GetEnvOrInt("DB_MAX_IDLE", 5),
            ConnMaxLife:    30 * time.Minute,
            ConnMaxIdleTime: 10 * time.Minute,
        }
    }
}
```

#### Default Rate Limits
| Operation | Rate | Rationale |
|-----------|------|-----------|
| SELECT | 1000/s | Read-heavy workload support |
| INSERT/UPDATE/DELETE | 100/s | Write constraint |
| DDL/ALTER | 10/s | Lock protection |
| ADMIN | 5/s | Safety |

### Tests to Create
- `TestTokenBucketRefill` - Token generation
- `TestRateLimitExceeded` - Rate limit enforcement
- `TestBackpressureHandling` - Queue management
- `TestConnectionPoolMetrics` - Pool statistics

---

## 🎯 FASE 3.4: Error Sanitization & Information Disclosure Prevention

### Current State
- ⚠️ Error messages include full SQL
- ⚠️ Stack traces not sanitized
- ⚠️ Database errors exposed to client

### Goals
- [ ] Sanitize error messages
- [ ] Create error categories
- [ ] Implement error translation
- [ ] Hide internal details

### Implementation Details

#### Files to Modify
- `internal/client.go` - Error handling
- `internal/errors.go` (NEW) - Error utilities

#### Error Classification
```go
type ErrorCategory string
const (
    ErrorAuth       ErrorCategory = "authentication"
    ErrorQuery      ErrorCategory = "query_error"
    ErrorConstraint ErrorCategory = "constraint_violation"
    ErrorTimeout    ErrorCategory = "operation_timeout"
    ErrorRateLimit  ErrorCategory = "rate_limit_exceeded"
    ErrorInternal   ErrorCategory = "internal_error"
)

// SanitizedError sanitizes error details for client consumption
type SanitizedError struct {
    Category    ErrorCategory `json:"category"`
    UserMessage string        `json:"message"`     // Safe for client
    InternalMsg string        `json:"-"`           // Logged only
    Code        string        `json:"code"`        // Error code
    RequestID   string        `json:"request_id"`  // Tracing
}

// SanitizeError converts database error to safe client error
func SanitizeError(err error, requestID string) *SanitizedError
```

#### Error Message Mapping
| Pattern | Category | User Message |
|---------|----------|--------------|
| Access denied | authentication | "Authentication failed" |
| Table not found | query_error | "Resource not found" |
| Duplicate key | constraint_violation | "Duplicate entry" |
| Timeout | timeout | "Operation timed out" |
| Rate limit | rate_limit | "Please retry later" |
| Other | internal_error | "Operation failed" |

---

## 📊 Implementation Timeline

### Phase 1: Context Timeouts (2-3 days)
- [ ] Create TimeoutConfig structure
- [ ] Implement timeout profiles
- [ ] Add timeout logging
- [ ] Create comprehensive tests

### Phase 2: Audit Logging (3-4 days)
- [ ] Create AuditLogger interface
- [ ] Implement JSONAuditLogger
- [ ] Add audit logging to all operations
- [ ] Create audit report tools
- [ ] Add log rotation

### Phase 3: Rate Limiting (2-3 days)
- [ ] Implement token bucket algorithm
- [ ] Add rate limiter to Client
- [ ] Create rate limit metrics
- [ ] Add connection pool tuning
- [ ] Create tests

### Phase 4: Error Sanitization (1-2 days)
- [ ] Create error utilities
- [ ] Implement error categorization
- [ ] Update all error returns
- [ ] Create error mapping tests

### Phase 5: Documentation & Testing (2-3 days)
- [ ] Create FASE_3_IMPLEMENTATION.md
- [ ] Add security examples
- [ ] Create monitoring guide
- [ ] Add troubleshooting guide

---

## 🔐 Security Considerations

### Timeout Attacks Prevention
- ✅ Per-operation timeouts prevent hanging queries
- ✅ Connection timeouts prevent ghost connections
- ✅ Write operation longer timeout prevents data loss

### Information Disclosure Prevention
- ✅ Error sanitization hides internal details
- ✅ Audit logs immutable and signed
- ✅ User messages generic and safe

### Rate Limiting Protection
- ✅ Prevents brute force attacks
- ✅ Protects against query bombs
- ✅ Prevents resource exhaustion

### Audit Trail Requirements
- ✅ All operations logged with timestamp
- ✅ User identification required
- ✅ Success/failure status tracked
- ✅ Compliant with industry standards (GDPR, HIPAA)

---

## 📈 Metrics to Track

### Performance Metrics
- Query execution time (p50, p95, p99)
- Write operation latency
- Connection pool utilization
- Rate limit violations

### Security Metrics
- Failed authentication attempts
- Blocked query attempts
- Timeout occurrences
- Error categorization

### Compliance Metrics
- Audit log completeness
- Operation success rate
- Data access patterns
- User activity tracking

---

## 📝 Testing Strategy

### Unit Tests (per module)
- 15+ timeout management tests
- 20+ audit logging tests
- 15+ rate limiting tests
- 10+ error sanitization tests

### Integration Tests
- Full operation lifecycle
- Multi-client scenarios
- Timeout + rate limit interaction
- Audit log completeness

### Security Tests
- Information disclosure attempts
- Rate limit circumvention
- Error message analysis
- Audit log tampering

---

## ✅ Definition of Done (FASE 3)

- [ ] All 60+ unit tests passing
- [ ] All integration tests passing
- [ ] Zero information disclosure issues
- [ ] Audit logs for 100% of operations
- [ ] Rate limiting enforced
- [ ] Documentation complete
- [ ] Security review passed
- [ ] Performance benchmarks within targets

---

## 📚 Related Documentation

- [SECURITY_PLAN_2025.md](./SECURITY_PLAN_2025.md) - Overall security roadmap
- [MARIADB_SETUP.md](./MARIADB_SETUP.md) - Database setup guide
- [MYSQL_MARIADB_COMPATIBILITY.md](./MYSQL_MARIADB_COMPATIBILITY.md) - Database compatibility

---

**Created:** 2026-01-21
**Status:** Planning Phase
**Next Step:** Begin FASE 3.1 implementation
