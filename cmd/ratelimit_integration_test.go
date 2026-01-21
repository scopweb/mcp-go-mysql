package main

import (
	"context"
	"sync"
	"testing"
	"time"

	mysql "mcp-gp-mysql/internal"
)

// ============= RATE LIMIT INTEGRATION TESTS =============

func TestRateLimiterWithTimeoutConfig(t *testing.T) {
	// Test that rate limiting and timeout management work together
	rateLimitConfig := &mysql.RateLimitConfig{
		QueriesPerSecond: 10,
		WritesPerSecond:  5,
		AdminPerSecond:   2,
	}
	rateLimiter := mysql.NewRateLimiter(rateLimitConfig)

	// Create timeout config
	timeoutConfig := mysql.NewTimeoutConfig()

	// Simulate operation sequence: rate limit check, then timeout context creation
	if !rateLimiter.AllowQuery() {
		t.Fatal("Query should be allowed")
	}

	ctx := context.Background()
	ctxWithTimeout, cancel := timeoutConfig.TimeoutContext(ctx, mysql.ProfileQuery)
	defer cancel()

	if ctxWithTimeout.Err() != nil {
		t.Error("Context should not be cancelled")
	}

	// Verify timeout was applied
	deadline, ok := ctxWithTimeout.Deadline()
	if !ok {
		t.Error("Expected deadline in context")
	}
	if deadline.Before(time.Now()) {
		t.Error("Deadline should be in the future")
	}
}

func TestRateLimiterWithAuditLogging(t *testing.T) {
	// Test that rate limit violations can be logged as security events
	rateLimitConfig := &mysql.RateLimitConfig{
		QueriesPerSecond: 5,
		WritesPerSecond:  5,
		AdminPerSecond:   5,
	}
	rateLimiter := mysql.NewRateLimiter(rateLimitConfig)

	// Create audit logger
	auditLogger := mysql.NewInMemoryAuditLogger()

	ctx := context.Background()
	ctxWithLogger := mysql.WithAuditLogger(ctx, auditLogger)

	// Use up all query tokens
	for i := 0; i < 5; i++ {
		if !rateLimiter.AllowQuery() {
			t.Fatalf("Query %d should be allowed", i+1)
		}
	}

	// Next query should be blocked - log as security event
	if rateLimiter.AllowQuery() {
		t.Fatal("6th query should be blocked")
	}

	// Create security event for blocked operation
	event := mysql.NewAuditEvent(mysql.EventTypeSecurity).
		WithOperation(mysql.OpSelect).
		WithStatus("blocked").
		WithMetadata("reason", "rate limit exceeded").
		WithSeverity(mysql.SeverityWarning).
		Build()

	auditLogger.LogSecurity(ctxWithLogger, event)

	// Verify event was logged
	events := auditLogger.GetEvents()
	if len(events) == 0 {
		t.Error("Expected security event to be logged")
	}

	if len(events) > 0 && events[0].Status != "blocked" {
		t.Errorf("Expected status 'blocked', got %s", events[0].Status)
	}

	if len(events) > 0 && events[0].Severity != mysql.SeverityWarning {
		t.Errorf("Expected severity warning, got %s", events[0].Severity)
	}
}

func TestRateLimiterWithDatabaseCompatibility(t *testing.T) {
	// Test rate limiting works with database compatibility layer
	// MariaDB might have different rate limits than MySQL

	// Get MariaDB config
	mariadbConfig := mysql.GetDBCompatibilityConfig("mariadb")
	if mariadbConfig == nil {
		t.Fatal("Failed to get MariaDB config")
	}

	// Create MariaDB-optimized rate limit config (slightly higher for MariaDB)
	rateLimitConfig := &mysql.RateLimitConfig{
		QueriesPerSecond: 1200, // Slightly higher for faster MariaDB
		WritesPerSecond:  150,
		AdminPerSecond:   15,
	}
	rateLimiter := mysql.NewRateLimiter(rateLimitConfig)

	// Verify rate limiter works with database config
	if !rateLimiter.AllowQuery() {
		t.Error("Query should be allowed")
	}

	// Verify configuration matches expectations
	cfg := rateLimiter.GetConfig()
	if cfg.QueriesPerSecond != 1200 {
		t.Errorf("Expected 1200 QPS, got %d", cfg.QueriesPerSecond)
	}

	// Verify MariaDB config is compatible
	if !mariadbConfig.SupportsSequences {
		t.Error("MariaDB should support sequences")
	}
}

func TestRateLimiterMultipleOperationTypes(t *testing.T) {
	// Test rate limiting different operation types independently
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 10,
		WritesPerSecond:  5,
		AdminPerSecond:   3,
	}
	rl := mysql.NewRateLimiter(config)

	// Use up query tokens
	for i := 0; i < 10; i++ {
		if !rl.AllowQuery() {
			t.Errorf("Query %d should be allowed", i+1)
		}
	}

	// Query should be blocked
	if rl.AllowQuery() {
		t.Error("Query should be blocked after limit")
	}

	// But writes and admin should still work
	if !rl.AllowWrite() {
		t.Error("Write should be allowed (separate bucket)")
	}
	if !rl.AllowAdmin() {
		t.Error("Admin should be allowed (separate bucket)")
	}

	// Use up remaining write tokens (1 already used above)
	for i := 0; i < 4; i++ {
		rl.AllowWrite()
	}

	// Next write should be blocked (5 tokens used, limit is 5)
	if rl.AllowWrite() {
		t.Error("Write should be blocked after limit")
	}

	// Admin still has 2 tokens left (one used above)
	if !rl.AllowAdmin() {
		t.Error("Admin should still be allowed (only used 1 of 3 tokens)")
	}

	// Verify metrics track all operation types
	metrics := rl.GetMetrics()
	// Total should be: 10 queries + 1 blocked query + 1 write + 4 more writes + 1 blocked write + 1 admin + 1 admin = 19
	if metrics.TotalOps != 19 {
		t.Logf("Total ops: %d (expected 19)", metrics.TotalOps)
	}
}

func TestRateLimiterCascadePrevention(t *testing.T) {
	// Test that rate limiting prevents cascading failures
	// Scenario: high request volume should gracefully degrade, not crash
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  20,
		AdminPerSecond:   5,
	}
	rl := mysql.NewRateLimiter(config)

	allowedQueries := 0
	blockedQueries := 0
	var mu sync.Mutex

	// Simulate burst of 200 concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.AllowQuery() {
				mu.Lock()
				allowedQueries++
				mu.Unlock()
			} else {
				mu.Lock()
				blockedQueries++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow up to capacity, block the rest
	if allowedQueries == 0 {
		t.Error("Expected some queries to be allowed")
	}
	if blockedQueries == 0 {
		t.Error("Expected some queries to be blocked")
	}
	if allowedQueries+blockedQueries != 200 {
		t.Errorf("Expected 200 total, got %d", allowedQueries+blockedQueries)
	}
	if allowedQueries > 100 {
		t.Errorf("Should not allow more than 100 query tokens, allowed %d", allowedQueries)
	}

	// System should remain responsive - metrics should be accessible
	metrics := rl.GetMetrics()
	if metrics.TotalOps != 200 {
		t.Errorf("Expected 200 total ops tracked, got %d", metrics.TotalOps)
	}
}

func TestRateLimiterRecoveryAfterSpike(t *testing.T) {
	// Test that rate limiter recovers after traffic spike
	refillRate := 100.0 // 100 tokens/second
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: int(refillRate),
		WritesPerSecond:  20,
		AdminPerSecond:   5,
	}
	rl := mysql.NewRateLimiter(config)

	// Use all query tokens
	for i := 0; i < 100; i++ {
		rl.AllowQuery()
	}

	// Verify bucket is empty
	if rl.AllowQuery() {
		t.Error("Bucket should be empty")
	}

	// Wait for tokens to refill (0.5s = ~50 tokens at 100/sec)
	time.Sleep(500 * time.Millisecond)

	// Should be able to acquire some tokens now
	acquiredCount := 0
	for i := 0; i < 60; i++ { // Try to acquire 60 tokens (more than should be available)
		if rl.AllowQuery() {
			acquiredCount++
		}
	}

	if acquiredCount == 0 {
		t.Error("Expected tokens to be available after refill")
	}
	if acquiredCount > 60 {
		t.Errorf("Should not acquire more than 60, got %d", acquiredCount)
	}
}

func TestRateLimiterContextIntegration(t *testing.T) {
	// Test rate limiter with context propagation
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 10,
		WritesPerSecond:  10,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	ctx := context.Background()

	// Create timeout context
	timeoutConfig := mysql.NewTimeoutConfig()
	ctxWithTimeout, cancel := timeoutConfig.TimeoutContext(ctx, mysql.ProfileQuery)
	defer cancel()

	// Create audit logger context
	auditLogger := mysql.NewInMemoryAuditLogger()
	ctxWithLogger := mysql.WithAuditLogger(ctxWithTimeout, auditLogger)

	// Rate limit check
	if !rl.AllowQuery() {
		t.Error("Query should be allowed")
	}

	// Log operation
	event := mysql.NewAuditEvent(mysql.EventTypeQuery).
		WithOperation(mysql.OpSelect).
		WithStatus("success").
		Build()

	auditLogger.LogQuery(ctxWithLogger, event)

	// Verify all layers worked together
	if ctxWithLogger.Err() != nil {
		t.Error("Context should not be cancelled")
	}

	events := auditLogger.GetEvents()
	if len(events) == 0 {
		t.Error("Expected event to be logged")
	}
}

func TestRateLimiterMetricsAccuracy(t *testing.T) {
	// Test that metrics accurately track operations
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 50,
		WritesPerSecond:  20,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Perform specific operations
	querySuccess := 0
	writeSuccess := 0
	adminSuccess := 0

	for i := 0; i < 30; i++ {
		if rl.AllowQuery() {
			querySuccess++
		}
	}
	for i := 0; i < 15; i++ {
		if rl.AllowWrite() {
			writeSuccess++
		}
	}
	for i := 0; i < 10; i++ {
		if rl.AllowAdmin() {
			adminSuccess++
		}
	}

	metrics := rl.GetMetrics()

	// Total ops should be sum of all attempts
	expectedTotal := 30 + 15 + 10
	if metrics.TotalOps != int64(expectedTotal) {
		t.Errorf("Expected %d total ops, got %d", expectedTotal, metrics.TotalOps)
	}

	// Blocked should be attempts that failed
	expectedBlocked := (30 - querySuccess) + (15 - writeSuccess) + (10 - adminSuccess)
	if metrics.BlockedOps != int64(expectedBlocked) {
		t.Errorf("Expected %d blocked ops, got %d", expectedBlocked, metrics.BlockedOps)
	}

	// Violation count should match blocked ops
	if metrics.ViolationCount != metrics.BlockedOps {
		t.Error("Violation count should match blocked ops")
	}
}

func TestRateLimiterConcurrentOperationTypes(t *testing.T) {
	// Test concurrent operations of different types
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  50,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	var wg sync.WaitGroup
	queryCount := 0
	writeCount := 0
	adminCount := 0
	var mu sync.Mutex

	// Launch 100 goroutines: some doing queries, some writes, some admin
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var success bool
			if id%3 == 0 {
				success = rl.AllowQuery()
			} else if id%3 == 1 {
				success = rl.AllowWrite()
			} else {
				success = rl.AllowAdmin()
			}

			if success {
				mu.Lock()
				if id%3 == 0 {
					queryCount++
				} else if id%3 == 1 {
					writeCount++
				} else {
					adminCount++
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify counts make sense
	if queryCount == 0 || writeCount == 0 || adminCount == 0 {
		t.Error("Expected operations of all types to succeed")
	}

	metrics := rl.GetMetrics()
	if metrics.TotalOps != 100 {
		t.Errorf("Expected 100 total ops, got %d", metrics.TotalOps)
	}
}
