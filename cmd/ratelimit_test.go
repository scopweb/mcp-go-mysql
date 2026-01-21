package main

import (
	"math"
	"sync"
	"testing"
	"time"

	mysql "mcp-gp-mysql/internal"
)

// ============= TOKEN BUCKET TESTS =============

func TestTokenBucketCreation(t *testing.T) {
	tests := []struct {
		name       string
		capacity   float64
		refillRate float64
	}{
		{"Standard bucket", 100, 50},
		{"High capacity", 1000, 100},
		{"Low capacity", 10, 5},
		{"Fractional rate", 100, 33.33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := mysql.NewTokenBucket(tt.capacity, tt.refillRate)
			if tb.GetCapacity() != tt.capacity {
				t.Errorf("Expected capacity %v, got %v", tt.capacity, tb.GetCapacity())
			}
			if tb.GetTokens() != tt.capacity {
				t.Errorf("Expected initial tokens %v, got %v", tt.capacity, tb.GetTokens())
			}
		})
	}
}

func TestTokenBucketAcquireToken(t *testing.T) {
	// Test 1: Acquire less than available
	tb := mysql.NewTokenBucket(100, 0)
	if !tb.AcquireToken(50) {
		t.Error("Failed to acquire 50 tokens from 100")
	}
	if tb.GetTokens() != 50 {
		t.Errorf("Expected 50 tokens remaining, got %v", tb.GetTokens())
	}

	// Test 2: Acquire exactly available
	tb2 := mysql.NewTokenBucket(100, 0)
	if !tb2.AcquireToken(100) {
		t.Error("Failed to acquire exactly 100 tokens")
	}
	if tb2.GetTokens() > 0.01 {
		t.Errorf("Expected ~0 tokens remaining, got %v", tb2.GetTokens())
	}

	// Test 3: Acquire more than available
	tb3 := mysql.NewTokenBucket(100, 0)
	tb3.AcquireToken(50) // Use 50 tokens first
	if tb3.AcquireToken(100) {
		t.Error("Should not allow acquiring 100 when only 50 available")
	}
	if tb3.GetTokens() != 50 {
		t.Errorf("Expected 50 tokens remaining, got %v", tb3.GetTokens())
	}

	// Test 4: Acquire zero tokens
	tb4 := mysql.NewTokenBucket(100, 0)
	if !tb4.AcquireToken(0) {
		t.Error("Should allow acquiring 0 tokens")
	}
	if tb4.GetTokens() != 100 {
		t.Errorf("Expected 100 tokens, got %v", tb4.GetTokens())
	}

	// Test 5: Acquire from empty bucket
	tb5 := mysql.NewTokenBucket(100, 0)
	tb5.AcquireToken(100) // Empty it
	if tb5.AcquireToken(1) {
		t.Error("Should not allow acquiring from empty bucket")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	refillRate := 100.0 // 100 tokens per second
	capacity := 100.0
	tb := mysql.NewTokenBucket(capacity, refillRate)

	// Use all tokens
	if !tb.AcquireToken(capacity) {
		t.Fatal("Failed to acquire initial tokens")
	}

	// Bucket should be empty
	if tb.GetTokens() > 0.1 {
		t.Errorf("Expected empty bucket, got %v tokens", tb.GetTokens())
	}

	// Wait for partial refill (0.2 seconds = ~20 tokens)
	time.Sleep(200 * time.Millisecond)

	tokens := tb.GetTokens()
	// Should have refilled roughly 20 tokens (100 tokens/sec * 0.2 sec)
	if tokens < 15 || tokens > 30 {
		t.Errorf("Expected ~20 tokens after 0.2s refill, got %v", tokens)
	}

	// Wait for full refill
	time.Sleep(1 * time.Second)
	tokens = tb.GetTokens()
	if tokens < 90 {
		t.Errorf("Expected ~100 tokens after refill, got %v", tokens)
	}
}

func TestTokenBucketConcurrency(t *testing.T) {
	tb := mysql.NewTokenBucket(1000, 100)

	var wg sync.WaitGroup
	acquired := 0
	blocked := 0
	mu := sync.Mutex{}

	// 100 goroutines trying to acquire tokens simultaneously
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.AcquireToken(10) {
				mu.Lock()
				acquired++
				mu.Unlock()
			} else {
				mu.Lock()
				blocked++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if acquired == 0 {
		t.Error("Expected some acquisitions with concurrent access")
	}
	if acquired+blocked != 100 {
		t.Errorf("Expected 100 total operations, got %d", acquired+blocked)
	}
}

func TestTokenBucketFractionalTokens(t *testing.T) {
	tb := mysql.NewTokenBucket(100, 33.33)

	// Acquire fractional amount
	if !tb.AcquireToken(33.33) {
		t.Error("Failed to acquire fractional tokens")
	}

	remaining := tb.GetTokens()
	expected := 100.0 - 33.33
	if math.Abs(remaining-expected) > 0.1 {
		t.Errorf("Expected %v tokens, got %v", expected, remaining)
	}
}

func TestTokenBucketAcquireWithWait(t *testing.T) {
	refillRate := 100.0
	tb := mysql.NewTokenBucket(100, refillRate)

	// Use all tokens
	tb.AcquireToken(100)

	// Try to acquire with wait - should succeed after refill
	start := time.Now()
	acquired := tb.AcquireTokenWithWait(50, 500*time.Millisecond)
	elapsed := time.Since(start)

	if !acquired {
		t.Error("Expected token acquisition with wait to succeed")
	}
	// Should wait at least 400ms to accumulate 50 tokens
	if elapsed < 400*time.Millisecond {
		t.Errorf("Expected wait time ~400ms+, got %v", elapsed)
	}
}

func TestTokenBucketAcquireWithWaitTimeout(t *testing.T) {
	tb := mysql.NewTokenBucket(10, 1) // Very low refill rate

	// Use all tokens
	tb.AcquireToken(10)

	// Try to acquire more than can refill in timeout
	start := time.Now()
	acquired := tb.AcquireTokenWithWait(100, 100*time.Millisecond)
	elapsed := time.Since(start)

	if acquired {
		t.Error("Expected token acquisition to timeout")
	}
	if elapsed < 100*time.Millisecond {
		t.Errorf("Expected timeout around 100ms, got %v", elapsed)
	}
}

func TestTokenBucketReset(t *testing.T) {
	tb := mysql.NewTokenBucket(100, 0)

	// Use some tokens
	tb.AcquireToken(50)
	if tb.GetTokens() > 50 {
		t.Errorf("Expected 50 tokens after acquisition")
	}

	// Reset
	tb.Reset()
	if tb.GetTokens() != 100 {
		t.Errorf("Expected 100 tokens after reset, got %v", tb.GetTokens())
	}
}

// ============= RATE LIMITER TESTS =============

func TestRateLimiterCreation(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 1000,
		WritesPerSecond:  100,
		AdminPerSecond:   10,
	}

	rl := mysql.NewRateLimiter(config)
	if rl == nil {
		t.Fatal("Failed to create rate limiter")
	}
	cfg := rl.GetConfig()
	if cfg.QueriesPerSecond != 1000 || cfg.WritesPerSecond != 100 || cfg.AdminPerSecond != 10 {
		t.Error("Rate limiter config mismatch")
	}
}

func TestRateLimiterDefaultConfig(t *testing.T) {
	rl := mysql.NewRateLimiter(nil)
	cfg := rl.GetConfig()

	if cfg.QueriesPerSecond != 1000 {
		t.Errorf("Expected 1000 QPS, got %d", cfg.QueriesPerSecond)
	}
	if cfg.WritesPerSecond != 100 {
		t.Errorf("Expected 100 WPS, got %d", cfg.WritesPerSecond)
	}
	if cfg.AdminPerSecond != 10 {
		t.Errorf("Expected 10 APS, got %d", cfg.AdminPerSecond)
	}
}

func TestRateLimiterAllowQuery(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 10,
		WritesPerSecond:  10,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Should allow initial queries (up to capacity)
	for i := 0; i < 10; i++ {
		if !rl.AllowQuery() {
			t.Errorf("Failed to allow query %d", i)
		}
	}

	// 11th query should be blocked
	if rl.AllowQuery() {
		t.Error("Expected 11th query to be blocked")
	}
}

func TestRateLimiterAllowWrite(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  5,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Should allow initial writes
	for i := 0; i < 5; i++ {
		if !rl.AllowWrite() {
			t.Errorf("Failed to allow write %d", i)
		}
	}

	// 6th write should be blocked
	if rl.AllowWrite() {
		t.Error("Expected 6th write to be blocked")
	}
}

func TestRateLimiterAllowAdmin(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  100,
		AdminPerSecond:   3,
	}
	rl := mysql.NewRateLimiter(config)

	// Should allow initial admin ops
	for i := 0; i < 3; i++ {
		if !rl.AllowAdmin() {
			t.Errorf("Failed to allow admin op %d", i)
		}
	}

	// 4th admin op should be blocked
	if rl.AllowAdmin() {
		t.Error("Expected 4th admin op to be blocked")
	}
}

func TestRateLimiterIndependentBuckets(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 5,
		WritesPerSecond:  5,
		AdminPerSecond:   5,
	}
	rl := mysql.NewRateLimiter(config)

	// Use up query bucket
	for i := 0; i < 5; i++ {
		rl.AllowQuery()
	}

	// Writes and admin should still be allowed
	if !rl.AllowWrite() {
		t.Error("Write should be allowed when query bucket empty")
	}
	if !rl.AllowAdmin() {
		t.Error("Admin should be allowed when query bucket empty")
	}
}

func TestRateLimiterMetrics(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 10,
		WritesPerSecond:  10,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Perform some operations within limits
	for i := 0; i < 4; i++ {
		rl.AllowQuery()
	}
	rl.AllowWrite()
	rl.AllowAdmin()

	metrics := rl.GetMetrics()
	if metrics.TotalOps != 6 {
		t.Errorf("Expected 6 total ops, got %d", metrics.TotalOps)
	}
	if metrics.BlockedOps != 0 {
		t.Errorf("Expected 0 blocked ops, got %d", metrics.BlockedOps)
	}

	// Use up remaining query tokens (10 total - 4 already used = 6 remaining)
	for i := 0; i < 6; i++ {
		rl.AllowQuery()
	}

	// Now additional query attempts should be blocked
	blockedCount := 0
	for i := 0; i < 5; i++ {
		if !rl.AllowQuery() {
			blockedCount++
		}
	}

	if blockedCount == 0 {
		t.Error("Expected some query attempts to be blocked after limit exceeded")
	}

	metrics = rl.GetMetrics()
	if metrics.TotalOps != 17 { // 6 initial + 6 to use up + 5 blocked attempts
		t.Errorf("Expected 17 total ops, got %d", metrics.TotalOps)
	}
	if metrics.BlockedOps < 1 {
		t.Error("Expected blocked ops after exceeding limit")
	}
	if metrics.ViolationCount != metrics.BlockedOps {
		t.Error("Expected violation count to match blocked ops")
	}
}

func TestRateLimiterReset(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 5,
		WritesPerSecond:  5,
		AdminPerSecond:   5,
	}
	rl := mysql.NewRateLimiter(config)

	// Use up all query tokens
	for i := 0; i < 5; i++ {
		rl.AllowQuery()
	}

	// Should be blocked
	if rl.AllowQuery() {
		t.Error("Expected query to be blocked before reset")
	}

	// Reset
	rl.Reset()

	// Should now be allowed
	if !rl.AllowQuery() {
		t.Error("Expected query to be allowed after reset")
	}

	// Metrics should be reset
	metrics := rl.GetMetrics()
	if metrics.TotalOps != 1 || metrics.BlockedOps != 0 {
		t.Error("Metrics not properly reset")
	}
}

func TestRateLimiterAllowQueryWithWait(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 10,
		WritesPerSecond:  10,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Use up tokens
	for i := 0; i < 10; i++ {
		rl.AllowQuery()
	}

	// Should be blocked immediately
	if rl.AllowQuery() {
		t.Error("Expected immediate block")
	}

	// Try with wait - should timeout before refill happens
	acquired := rl.AllowQueryWithWait(50 * time.Millisecond)
	if acquired {
		t.Error("Expected wait to timeout")
	}
}

func TestRateLimiterAllowWriteWithWait(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  10,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Use up write tokens
	for i := 0; i < 10; i++ {
		rl.AllowWrite()
	}

	// Query should still be allowed
	if !rl.AllowQuery() {
		t.Error("Query should be allowed")
	}

	// Write with wait should timeout quickly
	acquired := rl.AllowWriteWithWait(50 * time.Millisecond)
	if acquired {
		t.Error("Expected write wait to timeout")
	}
}

func TestRateLimiterAllowAdminWithWait(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  100,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Use up admin tokens
	for i := 0; i < 10; i++ {
		rl.AllowAdmin()
	}

	// Admin with wait should timeout quickly
	acquired := rl.AllowAdminWithWait(50 * time.Millisecond)
	if acquired {
		t.Error("Expected admin wait to timeout")
	}
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 1000,
		WritesPerSecond:  100,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	var wg sync.WaitGroup
	queryCount := 0
	writeCount := 0
	adminCount := 0
	mu := sync.Mutex{}

	// 100 goroutines doing different operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if id%3 == 0 {
				if rl.AllowQuery() {
					mu.Lock()
					queryCount++
					mu.Unlock()
				}
			} else if id%3 == 1 {
				if rl.AllowWrite() {
					mu.Lock()
					writeCount++
					mu.Unlock()
				}
			} else {
				if rl.AllowAdmin() {
					mu.Lock()
					adminCount++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify counts
	metrics := rl.GetMetrics()
	if metrics.TotalOps != 100 {
		t.Errorf("Expected 100 total ops, got %d", metrics.TotalOps)
	}

	if queryCount+writeCount+adminCount == 0 {
		t.Error("Expected at least some operations to succeed")
	}
}

func TestRateLimiterString(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  50,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	str := rl.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	if !containsRateLimitString(str, "RateLimiter") {
		t.Error("Expected 'RateLimiter' in string representation")
	}
}

func TestRateLimiterBucketTokens(t *testing.T) {
	config := &mysql.RateLimitConfig{
		QueriesPerSecond: 100,
		WritesPerSecond:  50,
		AdminPerSecond:   10,
	}
	rl := mysql.NewRateLimiter(config)

	// Initial state - full buckets
	queryTokens := rl.GetQueryBucketTokens()
	if queryTokens != 100 {
		t.Errorf("Expected 100 query tokens, got %v", queryTokens)
	}

	writeTokens := rl.GetWriteBucketTokens()
	if writeTokens != 50 {
		t.Errorf("Expected 50 write tokens, got %v", writeTokens)
	}

	adminTokens := rl.GetAdminBucketTokens()
	if adminTokens != 10 {
		t.Errorf("Expected 10 admin tokens, got %v", adminTokens)
	}

	// After some acquisitions
	rl.AllowQuery()
	rl.AllowWrite()
	rl.AllowAdmin()

	queryTokens = rl.GetQueryBucketTokens()
	if queryTokens >= 100 {
		t.Errorf("Expected less than 100 query tokens after acquisition")
	}
}

// ============= HELPER FUNCTIONS =============

func containsRateLimitString(haystack, needle string) bool {
	for i := 0; i < len(haystack)-len(needle)+1; i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
