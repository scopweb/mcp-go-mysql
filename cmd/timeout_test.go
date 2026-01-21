package main

import (
	"context"
	"testing"
	"time"

	mysql "mcp-gp-mysql/internal"
)

// TestTimeoutConfigDefaults verifies timeout configuration defaults
func TestTimeoutConfigDefaults(t *testing.T) {
	tc := mysql.NewTimeoutConfig()

	tests := []struct {
		name     string
		profile  mysql.TimeoutProfile
		expected time.Duration
	}{
		{"Default profile", mysql.ProfileDefault, 30 * time.Second},
		{"Query profile", mysql.ProfileQuery, 30 * time.Second},
		{"Long query profile", mysql.ProfileLongQuery, 5 * time.Minute},
		{"Write profile", mysql.ProfileWrite, 60 * time.Second},
		{"Admin profile", mysql.ProfileAdmin, 15 * time.Second},
		{"Connection profile", mysql.ProfileConnection, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := tc.GetTimeout(tt.profile)
			if timeout != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, timeout)
			}
		})
	}
}

// TestTimeoutConfigCustom verifies custom timeout configuration
func TestTimeoutConfigCustom(t *testing.T) {
	tc := &mysql.TimeoutConfig{
		Default:    45 * time.Second,
		Query:      45 * time.Second,
		LongQuery:  10 * time.Minute,
		Write:      90 * time.Second,
		Admin:      20 * time.Second,
		Connection: 10 * time.Second,
	}

	if tc.GetTimeout(mysql.ProfileQuery) != 45*time.Second {
		t.Errorf("Custom query timeout not set correctly")
	}

	if tc.GetTimeout(mysql.ProfileLongQuery) != 10*time.Minute {
		t.Errorf("Custom long query timeout not set correctly")
	}
}

// TestTimeoutContext verifies context creation with timeout
func TestTimeoutContext(t *testing.T) {
	tc := mysql.NewTimeoutConfig()
	ctx := context.Background()

	now := time.Now()
	ctxWithTimeout, cancel := tc.TimeoutContext(ctx, mysql.ProfileQuery)
	defer cancel()

	// Verify context has deadline
	deadline, ok := ctxWithTimeout.Deadline()
	if !ok {
		t.Errorf("Context should have deadline")
		return
	}

	// Verify deadline is approximately 30 seconds from now
	expectedDeadline := now.Add(30 * time.Second)
	diff := deadline.Sub(expectedDeadline)

	// Allow 500ms tolerance
	if diff < -500*time.Millisecond || diff > 500*time.Millisecond {
		t.Errorf("Deadline not approximately 30 seconds away: diff=%v", diff)
	}
}

// TestTimeoutDetails tracks timeout information
func TestTimeoutDetails(t *testing.T) {
	profile := mysql.ProfileQuery
	timeout := 30 * time.Second

	details := mysql.NewTimeoutDetails(profile, timeout)

	if details.Profile != profile {
		t.Errorf("Profile mismatch: expected %s, got %s", profile, details.Profile)
	}

	if details.Timeout != timeout {
		t.Errorf("Timeout mismatch: expected %v, got %v", timeout, details.Timeout)
	}

	if details.IsTimeout {
		t.Errorf("Should not be timeout initially")
	}

	if details.Elapsed != 0 {
		t.Errorf("Elapsed should be 0 initially")
	}

	// Sleep briefly and record
	time.Sleep(10 * time.Millisecond)
	details.Record(false)

	if details.Elapsed < 10*time.Millisecond {
		t.Errorf("Elapsed time too short: %v", details.Elapsed)
	}

	if details.IsTimeout {
		t.Errorf("Should not be timeout after record(false)")
	}

	remaining := details.RemainingTime
	if remaining >= timeout {
		t.Errorf("Remaining time should be less than timeout: %v >= %v", remaining, timeout)
	}
}

// TestTimeoutDetailsString verifies string representation
func TestTimeoutDetailsString(t *testing.T) {
	details := mysql.NewTimeoutDetails(mysql.ProfileQuery, 30*time.Second)
	str := details.String()

	if str == "" {
		t.Errorf("String representation should not be empty")
	}

	if !containsString(str, "query") {
		t.Errorf("String should contain profile name")
	}

	if !containsString(str, "30s") {
		t.Errorf("String should contain timeout value")
	}
}

// TestIsNearDeadline checks deadline proximity detection
func TestIsNearDeadline(t *testing.T) {
	// Test with non-nil details
	details := mysql.NewTimeoutDetails(mysql.ProfileQuery, 100*time.Millisecond)

	// Should not be near deadline initially
	if details.IsNearDeadline() {
		t.Errorf("Should not be near deadline initially")
	}

	// Sleep until near deadline
	time.Sleep(50 * time.Millisecond)
	// At this point, remaining time should be ~50ms, but we check for <= 1 second

	// Sleep more to get within 1 second of deadline
	details2 := mysql.NewTimeoutDetails(mysql.ProfileQuery, 500*time.Millisecond)
	time.Sleep(400 * time.Millisecond)

	if !details2.IsNearDeadline() {
		t.Errorf("Should be near deadline when remaining < 1s")
	}

	// Test with nil details
	if mysql.GetTimeoutMetrics(context.Background()) != nil {
		t.Errorf("Should not have timeout details in fresh context")
	}
}

// TestContextWithTimeoutMetrics attaches/retrieves timeout details
func TestContextWithTimeoutMetrics(t *testing.T) {
	ctx := context.Background()
	details := mysql.NewTimeoutDetails(mysql.ProfileWrite, 60*time.Second)

	// Attach to context
	ctxWithMetrics := mysql.WithTimeoutMetrics(ctx, details)

	// Retrieve from context
	retrieved := mysql.GetTimeoutMetrics(ctxWithMetrics)
	if retrieved == nil {
		t.Errorf("Should retrieve timeout details from context")
	}

	if retrieved.Profile != mysql.ProfileWrite {
		t.Errorf("Profile mismatch: expected %s, got %s", mysql.ProfileWrite, retrieved.Profile)
	}

	if retrieved.Timeout != 60*time.Second {
		t.Errorf("Timeout mismatch: expected 60s, got %v", retrieved.Timeout)
	}

	// Verify original context doesn't have details
	original := mysql.GetTimeoutMetrics(ctx)
	if original != nil {
		t.Errorf("Original context should not have timeout details")
	}
}

// TestValidateTimeoutDuration checks timeout validation
func TestValidateTimeoutDuration(t *testing.T) {
	tests := []struct {
		name      string
		duration  time.Duration
		shouldErr bool
	}{
		{"Valid 5 seconds", 5 * time.Second, false},
		{"Valid 30 seconds", 30 * time.Second, false},
		{"Valid 1 minute", 1 * time.Minute, false},
		{"Valid 1 hour", 1 * time.Hour, false},
		{"Invalid zero", 0, true},
		{"Invalid negative", -1 * time.Second, true},
		{"Invalid too long", 25 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mysql.ValidateTimeoutDuration(tt.duration)
			if (err != nil) != tt.shouldErr {
				t.Errorf("Expected error: %v, got: %v", tt.shouldErr, err)
			}
		})
	}
}

// TestTimeoutProfileConversion verifies all profiles are handled
func TestTimeoutProfileConversion(t *testing.T) {
	tc := mysql.NewTimeoutConfig()

	profiles := []mysql.TimeoutProfile{
		mysql.ProfileDefault,
		mysql.ProfileQuery,
		mysql.ProfileLongQuery,
		mysql.ProfileWrite,
		mysql.ProfileAdmin,
		mysql.ProfileConnection,
	}

	for _, profile := range profiles {
		timeout := tc.GetTimeout(profile)
		if timeout <= 0 {
			t.Errorf("Profile %s should have positive timeout, got %v", profile, timeout)
		}
	}
}

// TestTimeoutCancellation verifies context cancellation works
func TestTimeoutCancellation(t *testing.T) {
	tc := mysql.NewTimeoutConfig()
	ctx := context.Background()

	// Create context with very short timeout
	ctxWithTimeout, cancel := tc.TimeoutContext(ctx, mysql.ProfileConnection)

	// Cancel immediately
	cancel()

	// Check if context is cancelled
	select {
	case <-ctxWithTimeout.Done():
		// Success - context was cancelled
	default:
		t.Errorf("Context should be cancelled after calling cancel()")
	}
}

// TestTimeoutRecordAccuracy verifies timeout recording accuracy
func TestTimeoutRecordAccuracy(t *testing.T) {
	const sleepDuration = 50 * time.Millisecond
	const tolerance = 10 * time.Millisecond

	details := mysql.NewTimeoutDetails(mysql.ProfileQuery, 1*time.Second)
	start := time.Now()
	time.Sleep(sleepDuration)
	details.Record(false)

	elapsed := details.Elapsed
	actualElapsed := time.Since(start)

	// Verify recorded elapsed time is approximately correct
	if elapsed < sleepDuration-tolerance || elapsed > sleepDuration+tolerance {
		t.Errorf("Recorded elapsed %v not near actual %v", elapsed, actualElapsed)
	}

	// Verify remaining time is calculated correctly
	expected := 1*time.Second - elapsed
	if details.RemainingTime < expected-tolerance || details.RemainingTime > expected+tolerance {
		t.Errorf("Remaining time %v not near expected %v", details.RemainingTime, expected)
	}
}

// TestTimeoutDetailsNil verifies nil handling
func TestTimeoutDetailsNil(t *testing.T) {
	var details *mysql.TimeoutDetails

	str := details.String()
	if str != "no timeout details" {
		t.Errorf("Nil details string should be 'no timeout details', got %q", str)
	}

	if details.IsNearDeadline() {
		t.Errorf("Nil details IsNearDeadline should return false")
	}
}

// containsString checks if a string contains a substring (avoids duplicate helper)
func containsString(str, substr string) bool {
	for i := 0; i < len(str)-len(substr)+1; i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
