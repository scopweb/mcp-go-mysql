package internal

import (
	"fmt"
	"sync"
	"time"
)

// TokenBucket implements the token bucket algorithm for rate limiting.
type TokenBucket struct {
	capacity       float64
	tokens         float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	mu             sync.RWMutex
}

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	QueriesPerSecond  int
	WritesPerSecond   int
	AdminPerSecond    int
	BackpressureDelay time.Duration
	MaxQueuedOps      int
}

// RateLimiter provides operation-level rate limiting with separate buckets
// for queries, writes, and admin operations.
type RateLimiter struct {
	queryBucket *TokenBucket
	writeBucket *TokenBucket
	adminBucket *TokenBucket
	config      *RateLimitConfig
	metrics     *RateLimitMetrics
	mu          sync.Mutex
}

// RateLimitMetrics tracks rate limiting statistics.
type RateLimitMetrics struct {
	TotalOps       int64
	BlockedOps     int64
	ThrottledOps   int64
	AvgWaitTime    time.Duration
	ViolationCount int64
	mu             sync.RWMutex
}

// NewTokenBucket creates a new token bucket with the given capacity
// and refill rate (tokens per second).
func NewTokenBucket(capacity float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		tokens:         capacity, // Start with full capacity
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// refillTokens calculates and adds tokens based on elapsed time.
func (tb *TokenBucket) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()
	if elapsed > 0 {
		newTokens := elapsed * tb.refillRate
		tb.tokens = min(tb.capacity, tb.tokens+newTokens)
		tb.lastRefillTime = now
	}
}

// AcquireToken attempts to acquire the specified number of tokens without waiting.
// Returns true if tokens were acquired, false otherwise.
func (tb *TokenBucket) AcquireToken(count float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refillTokens()

	if tb.tokens >= count {
		tb.tokens -= count
		return true
	}
	return false
}

// AcquireTokenWithWait attempts to acquire tokens, waiting up to the specified
// timeout if necessary.
func (tb *TokenBucket) AcquireTokenWithWait(count float64, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for {
		if tb.AcquireToken(count) {
			return true
		}

		if time.Now().After(deadline) {
			return false
		}

		// Sleep briefly before retrying
		time.Sleep(1 * time.Millisecond)
	}
}

// GetTokens returns the current number of tokens.
func (tb *TokenBucket) GetTokens() float64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	// Recalculate based on elapsed time without modifying state
	elapsed := time.Now().Sub(tb.lastRefillTime).Seconds()
	tokens := tb.tokens + (elapsed * tb.refillRate)
	if tokens > tb.capacity {
		tokens = tb.capacity
	}
	return tokens
}

// GetCapacity returns the bucket capacity.
func (tb *TokenBucket) GetCapacity() float64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return tb.capacity
}

// Reset resets the bucket to full capacity.
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.tokens = tb.capacity
	tb.lastRefillTime = time.Now()
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	return &RateLimiter{
		queryBucket: NewTokenBucket(
			float64(config.QueriesPerSecond),
			float64(config.QueriesPerSecond),
		),
		writeBucket: NewTokenBucket(
			float64(config.WritesPerSecond),
			float64(config.WritesPerSecond),
		),
		adminBucket: NewTokenBucket(
			float64(config.AdminPerSecond),
			float64(config.AdminPerSecond),
		),
		config:  config,
		metrics: &RateLimitMetrics{},
	}
}

// DefaultRateLimitConfig returns the default rate limiting configuration.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		QueriesPerSecond:  1000,
		WritesPerSecond:   100,
		AdminPerSecond:    10,
		BackpressureDelay: 100 * time.Millisecond,
		MaxQueuedOps:      500,
	}
}

// AllowQuery checks if a query operation is allowed under the current rate limits.
func (rl *RateLimiter) AllowQuery() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.metrics.mu.Lock()
	rl.metrics.TotalOps++
	rl.metrics.mu.Unlock()

	if rl.queryBucket.AcquireToken(1.0) {
		return true
	}

	rl.metrics.mu.Lock()
	rl.metrics.BlockedOps++
	rl.metrics.ViolationCount++
	rl.metrics.mu.Unlock()

	return false
}

// AllowWrite checks if a write operation is allowed under the current rate limits.
func (rl *RateLimiter) AllowWrite() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.metrics.mu.Lock()
	rl.metrics.TotalOps++
	rl.metrics.mu.Unlock()

	if rl.writeBucket.AcquireToken(1.0) {
		return true
	}

	rl.metrics.mu.Lock()
	rl.metrics.BlockedOps++
	rl.metrics.ViolationCount++
	rl.metrics.mu.Unlock()

	return false
}

// AllowAdmin checks if an admin operation is allowed under the current rate limits.
func (rl *RateLimiter) AllowAdmin() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.metrics.mu.Lock()
	rl.metrics.TotalOps++
	rl.metrics.mu.Unlock()

	if rl.adminBucket.AcquireToken(1.0) {
		return true
	}

	rl.metrics.mu.Lock()
	rl.metrics.BlockedOps++
	rl.metrics.ViolationCount++
	rl.metrics.mu.Unlock()

	return false
}

// AllowQueryWithWait attempts to acquire a query token, waiting if necessary.
func (rl *RateLimiter) AllowQueryWithWait(timeout time.Duration) bool {
	rl.mu.Lock()
	rl.metrics.mu.Lock()
	rl.metrics.TotalOps++
	rl.metrics.mu.Unlock()
	defer rl.mu.Unlock()

	if rl.queryBucket.AcquireTokenWithWait(1.0, timeout) {
		return true
	}

	rl.metrics.mu.Lock()
	rl.metrics.BlockedOps++
	rl.metrics.ViolationCount++
	rl.metrics.mu.Unlock()

	return false
}

// AllowWriteWithWait attempts to acquire a write token, waiting if necessary.
func (rl *RateLimiter) AllowWriteWithWait(timeout time.Duration) bool {
	rl.mu.Lock()
	rl.metrics.mu.Lock()
	rl.metrics.TotalOps++
	rl.metrics.mu.Unlock()
	defer rl.mu.Unlock()

	if rl.writeBucket.AcquireTokenWithWait(1.0, timeout) {
		return true
	}

	rl.metrics.mu.Lock()
	rl.metrics.BlockedOps++
	rl.metrics.ViolationCount++
	rl.metrics.mu.Unlock()

	return false
}

// AllowAdminWithWait attempts to acquire an admin token, waiting if necessary.
func (rl *RateLimiter) AllowAdminWithWait(timeout time.Duration) bool {
	rl.mu.Lock()
	rl.metrics.mu.Lock()
	rl.metrics.TotalOps++
	rl.metrics.mu.Unlock()
	defer rl.mu.Unlock()

	if rl.adminBucket.AcquireTokenWithWait(1.0, timeout) {
		return true
	}

	rl.metrics.mu.Lock()
	rl.metrics.BlockedOps++
	rl.metrics.ViolationCount++
	rl.metrics.mu.Unlock()

	return false
}

// GetMetrics returns a copy of current rate limiting metrics.
func (rl *RateLimiter) GetMetrics() *RateLimitMetrics {
	rl.metrics.mu.RLock()
	defer rl.metrics.mu.RUnlock()

	return &RateLimitMetrics{
		TotalOps:       rl.metrics.TotalOps,
		BlockedOps:     rl.metrics.BlockedOps,
		ThrottledOps:   rl.metrics.ThrottledOps,
		AvgWaitTime:    rl.metrics.AvgWaitTime,
		ViolationCount: rl.metrics.ViolationCount,
	}
}

// Reset resets all buckets and metrics to initial state.
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.queryBucket.Reset()
	rl.writeBucket.Reset()
	rl.adminBucket.Reset()

	rl.metrics.mu.Lock()
	defer rl.metrics.mu.Unlock()

	rl.metrics.TotalOps = 0
	rl.metrics.BlockedOps = 0
	rl.metrics.ThrottledOps = 0
	rl.metrics.AvgWaitTime = 0
	rl.metrics.ViolationCount = 0
}

// GetConfig returns the current rate limit configuration.
func (rl *RateLimiter) GetConfig() *RateLimitConfig {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return &RateLimitConfig{
		QueriesPerSecond:  rl.config.QueriesPerSecond,
		WritesPerSecond:   rl.config.WritesPerSecond,
		AdminPerSecond:    rl.config.AdminPerSecond,
		BackpressureDelay: rl.config.BackpressureDelay,
		MaxQueuedOps:      rl.config.MaxQueuedOps,
	}
}

// GetQueryBucketTokens returns current tokens in query bucket.
func (rl *RateLimiter) GetQueryBucketTokens() float64 {
	return rl.queryBucket.GetTokens()
}

// GetWriteBucketTokens returns current tokens in write bucket.
func (rl *RateLimiter) GetWriteBucketTokens() float64 {
	return rl.writeBucket.GetTokens()
}

// GetAdminBucketTokens returns current tokens in admin bucket.
func (rl *RateLimiter) GetAdminBucketTokens() float64 {
	return rl.adminBucket.GetTokens()
}

// String returns a string representation of rate limiter status.
func (rl *RateLimiter) String() string {
	metrics := rl.GetMetrics()
	return fmt.Sprintf(
		"RateLimiter{Total: %d, Blocked: %d, Violations: %d, QueryTokens: %.2f, WriteTokens: %.2f, AdminTokens: %.2f}",
		metrics.TotalOps,
		metrics.BlockedOps,
		metrics.ViolationCount,
		rl.GetQueryBucketTokens(),
		rl.GetWriteBucketTokens(),
		rl.GetAdminBucketTokens(),
	)
}

// min returns the minimum of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
