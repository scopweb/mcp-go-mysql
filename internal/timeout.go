package internal

import (
	"context"
	"fmt"
	"time"
)

// TimeoutProfile identifies the operation type for timeout selection
type TimeoutProfile string

const (
	// ProfileDefault is the standard timeout for generic operations
	ProfileDefault TimeoutProfile = "default"
	// ProfileQuery is for SELECT operations
	ProfileQuery TimeoutProfile = "query"
	// ProfileLongQuery is for complex/aggregation queries
	ProfileLongQuery TimeoutProfile = "long_query"
	// ProfileWrite is for INSERT/UPDATE/DELETE operations
	ProfileWrite TimeoutProfile = "write"
	// ProfileAdmin is for DDL operations (CREATE, DROP, ALTER)
	ProfileAdmin TimeoutProfile = "admin"
	// ProfileConnection is for connection establishment
	ProfileConnection TimeoutProfile = "connection"
)

// TimeoutConfig manages operation-specific timeouts
type TimeoutConfig struct {
	// Default is the standard timeout for undefined operations
	Default time.Duration
	// Query is the timeout for SELECT operations
	Query time.Duration
	// LongQuery is the timeout for complex/aggregation queries
	LongQuery time.Duration
	// Write is the timeout for INSERT/UPDATE/DELETE operations
	Write time.Duration
	// Admin is the timeout for DDL operations
	Admin time.Duration
	// Connection is the timeout for connection establishment
	Connection time.Duration
}

// NewTimeoutConfig creates a new timeout configuration with sensible defaults
func NewTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		Default:    30 * time.Second,
		Query:      30 * time.Second,
		LongQuery:  5 * time.Minute,
		Write:      60 * time.Second,
		Admin:      15 * time.Second,
		Connection: 5 * time.Second,
	}
}

// GetTimeout returns the timeout for the given profile
func (tc *TimeoutConfig) GetTimeout(profile TimeoutProfile) time.Duration {
	if tc == nil {
		return 30 * time.Second // Fallback to safe default
	}

	switch profile {
	case ProfileQuery:
		return tc.Query
	case ProfileLongQuery:
		return tc.LongQuery
	case ProfileWrite:
		return tc.Write
	case ProfileAdmin:
		return tc.Admin
	case ProfileConnection:
		return tc.Connection
	default:
		return tc.Default
	}
}

// TimeoutContext creates a context with appropriate timeout for the operation type
func (tc *TimeoutConfig) TimeoutContext(ctx context.Context, profile TimeoutProfile) (context.Context, context.CancelFunc) {
	timeout := tc.GetTimeout(profile)
	return context.WithTimeout(ctx, timeout)
}

// TimeoutDetails tracks timeout-related information for an operation
type TimeoutDetails struct {
	Profile        TimeoutProfile `json:"profile"`
	Timeout        time.Duration  `json:"timeout_ms"`
	Elapsed        time.Duration  `json:"elapsed_ms"`
	IsTimeout      bool           `json:"is_timeout"`
	RemainingTime  time.Duration  `json:"remaining_ms"`
	StartTime      time.Time      `json:"start_time"`
}

// NewTimeoutDetails creates timeout tracking for an operation
func NewTimeoutDetails(profile TimeoutProfile, timeout time.Duration) *TimeoutDetails {
	return &TimeoutDetails{
		Profile:   profile,
		Timeout:   timeout,
		StartTime: time.Now(),
	}
}

// Record updates the timeout details after operation completion
func (td *TimeoutDetails) Record(isTimeout bool) {
	td.Elapsed = time.Since(td.StartTime)
	td.IsTimeout = isTimeout
	if td.Timeout > td.Elapsed {
		td.RemainingTime = td.Timeout - td.Elapsed
	} else {
		td.RemainingTime = 0
	}
}

// IsNearDeadline checks if timeout is approaching (within 1 second)
func (td *TimeoutDetails) IsNearDeadline() bool {
	if td == nil {
		return false
	}
	elapsed := time.Since(td.StartTime)
	remaining := td.Timeout - elapsed
	return remaining <= 1*time.Second
}

// String returns a human-readable representation of timeout details
func (td *TimeoutDetails) String() string {
	if td == nil {
		return "no timeout details"
	}
	return fmt.Sprintf("profile=%s timeout=%s elapsed=%s remaining=%s",
		td.Profile, td.Timeout, td.Elapsed, td.RemainingTime)
}

// ContextWithTimeoutMetrics is a context value key for timeout metrics
type contextKeyTimeoutMetrics struct{}

// WithTimeoutMetrics attaches timeout details to a context
func WithTimeoutMetrics(ctx context.Context, details *TimeoutDetails) context.Context {
	return context.WithValue(ctx, contextKeyTimeoutMetrics{}, details)
}

// GetTimeoutMetrics retrieves timeout details from a context
func GetTimeoutMetrics(ctx context.Context) *TimeoutDetails {
	if details, ok := ctx.Value(contextKeyTimeoutMetrics{}).(*TimeoutDetails); ok {
		return details
	}
	return nil
}

// ValidateTimeoutDuration ensures a timeout duration is sensible
func ValidateTimeoutDuration(timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", timeout)
	}
	if timeout > 24*time.Hour {
		return fmt.Errorf("timeout is unusually long (%v), please check configuration", timeout)
	}
	return nil
}
