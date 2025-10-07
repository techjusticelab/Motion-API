package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// tokenBucketLimiter implements rate limiting using the token bucket algorithm
type tokenBucketLimiter struct {
	rate       float64       // tokens per second
	burst      int           // maximum burst size
	tokens     float64       // current tokens
	lastUpdate time.Time     // last token update time
	mutex      sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(requestsPerMinute int, burstSize int) RateLimiter {
	rate := float64(requestsPerMinute) / 60.0 // convert to requests per second
	
	return &tokenBucketLimiter{
		rate:       rate,
		burst:      burstSize,
		tokens:     float64(burstSize), // start with full bucket
		lastUpdate: time.Now(),
	}
}

// Allow returns true if an operation is allowed
func (r *tokenBucketLimiter) Allow() bool {
	return r.AllowN(1)
}

// AllowN returns true if n operations are allowed
func (r *tokenBucketLimiter) AllowN(n int) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.updateTokens()
	
	if r.tokens >= float64(n) {
		r.tokens -= float64(n)
		return true
	}
	
	return false
}

// Reserve reserves permission for an operation
func (r *tokenBucketLimiter) Reserve() Reservation {
	return r.ReserveN(1)
}

// ReserveN reserves permission for n operations
func (r *tokenBucketLimiter) ReserveN(n int) Reservation {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.updateTokens()
	
	if r.tokens >= float64(n) {
		r.tokens -= float64(n)
		return &reservation{ok: true, delay: 0}
	}
	
	// Calculate delay needed
	needed := float64(n) - r.tokens
	delay := time.Duration(needed/r.rate) * time.Second
	
	// Use all available tokens
	r.tokens = 0
	
	return &reservation{ok: true, delay: delay}
}

// Wait waits until permission is granted
func (r *tokenBucketLimiter) Wait(ctx context.Context) error {
	return r.WaitN(ctx, 1)
}

// WaitN waits until permission for n operations is granted
func (r *tokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	reservation := r.ReserveN(n)
	
	if !reservation.OK() {
		return fmt.Errorf("rate limit exceeded")
	}
	
	delay := reservation.Delay()
	if delay <= 0 {
		return nil
	}
	
	timer := time.NewTimer(delay)
	defer timer.Stop()
	
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// updateTokens updates the token count based on elapsed time
func (r *tokenBucketLimiter) updateTokens() {
	now := time.Now()
	elapsed := now.Sub(r.lastUpdate).Seconds()
	
	// Add tokens based on elapsed time
	r.tokens += elapsed * r.rate
	
	// Cap at burst size
	if r.tokens > float64(r.burst) {
		r.tokens = float64(r.burst)
	}
	
	r.lastUpdate = now
}

// reservation implements the Reservation interface
type reservation struct {
	ok    bool
	delay time.Duration
}

// OK returns true if the reservation is valid
func (r *reservation) OK() bool {
	return r.ok
}

// Delay returns the duration to wait before acting
func (r *reservation) Delay() time.Duration {
	return r.delay
}

// Cancel cancels the reservation (no-op for this implementation)
func (r *reservation) Cancel() {
	// No-op for token bucket
}

// adaptiveRateLimiter adjusts rate limiting based on response times and errors
type adaptiveRateLimiter struct {
	baseLimiter     RateLimiter
	mutex           sync.RWMutex
	responseTimeSum time.Duration
	responseCount   int64
	errorCount      int64
	successCount    int64
	lastAdjustment  time.Time
	
	// Configuration
	baseRate           int
	burstSize          int
	maxRate            int
	minRate            int
	adjustmentInterval time.Duration
	errorThreshold     float64
	responseThreshold  time.Duration
}

// NewAdaptiveRateLimiter creates a rate limiter that adapts based on API performance
func NewAdaptiveRateLimiter(baseRate, burstSize, minRate, maxRate int) RateLimiter {
	limiter := &adaptiveRateLimiter{
		baseRate:           baseRate,
		burstSize:          burstSize,
		maxRate:            maxRate,
		minRate:            minRate,
		adjustmentInterval: 30 * time.Second,
		errorThreshold:     0.1,  // 10% error rate
		responseThreshold:  5 * time.Second,
		lastAdjustment:     time.Now(),
	}
	
	limiter.baseLimiter = NewTokenBucketLimiter(baseRate, burstSize)
	return limiter
}

// Allow returns true if an operation is allowed
func (a *adaptiveRateLimiter) Allow() bool {
	a.adjustRateIfNeeded()
	return a.baseLimiter.Allow()
}

// AllowN returns true if n operations are allowed
func (a *adaptiveRateLimiter) AllowN(n int) bool {
	a.adjustRateIfNeeded()
	return a.baseLimiter.AllowN(n)
}

// Reserve reserves permission for an operation
func (a *adaptiveRateLimiter) Reserve() Reservation {
	a.adjustRateIfNeeded()
	return a.baseLimiter.Reserve()
}

// ReserveN reserves permission for n operations
func (a *adaptiveRateLimiter) ReserveN(n int) Reservation {
	a.adjustRateIfNeeded()
	return a.baseLimiter.ReserveN(n)
}

// Wait waits until permission is granted
func (a *adaptiveRateLimiter) Wait(ctx context.Context) error {
	a.adjustRateIfNeeded()
	return a.baseLimiter.Wait(ctx)
}

// WaitN waits until permission for n operations is granted
func (a *adaptiveRateLimiter) WaitN(ctx context.Context, n int) error {
	a.adjustRateIfNeeded()
	return a.baseLimiter.WaitN(ctx, n)
}

// RecordResponse records a successful response time
func (a *adaptiveRateLimiter) RecordResponse(duration time.Duration) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	a.responseTimeSum += duration
	a.responseCount++
	a.successCount++
}

// RecordError records an error
func (a *adaptiveRateLimiter) RecordError() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	a.errorCount++
}

// adjustRateIfNeeded adjusts the rate limit based on performance metrics
func (a *adaptiveRateLimiter) adjustRateIfNeeded() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	now := time.Now()
	if now.Sub(a.lastAdjustment) < a.adjustmentInterval {
		return
	}
	
	if a.responseCount == 0 && a.errorCount == 0 {
		return // No data to adjust on
	}
	
	totalRequests := a.successCount + a.errorCount
	errorRate := float64(a.errorCount) / float64(totalRequests)
	
	var avgResponseTime time.Duration
	if a.responseCount > 0 {
		avgResponseTime = a.responseTimeSum / time.Duration(a.responseCount)
	}
	
	newRate := a.baseRate
	
	// Decrease rate if error rate is high
	if errorRate > a.errorThreshold {
		newRate = int(float64(a.baseRate) * 0.8) // Reduce by 20%
	}
	
	// Decrease rate if response time is too high
	if avgResponseTime > a.responseThreshold {
		newRate = int(float64(newRate) * 0.9) // Reduce by 10%
	}
	
	// Increase rate if everything is good
	if errorRate < a.errorThreshold/2 && avgResponseTime < a.responseThreshold/2 {
		newRate = int(float64(newRate) * 1.1) // Increase by 10%
	}
	
	// Apply bounds
	if newRate < a.minRate {
		newRate = a.minRate
	}
	if newRate > a.maxRate {
		newRate = a.maxRate
	}
	
	// Update base rate if it changed significantly
	if abs(newRate-a.baseRate) > a.baseRate/10 { // 10% change threshold
		a.baseRate = newRate
		a.baseLimiter = NewTokenBucketLimiter(newRate, a.burstSize)
	}
	
	// Reset metrics
	a.responseTimeSum = 0
	a.responseCount = 0
	a.errorCount = 0
	a.successCount = 0
	a.lastAdjustment = now
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GetCurrentRate returns the current rate limit
func (a *adaptiveRateLimiter) GetCurrentRate() int {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.baseRate
}

// GetStats returns rate limiter statistics
func (a *adaptiveRateLimiter) GetStats() map[string]interface{} {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["current_rate"] = a.baseRate
	stats["burst_size"] = a.burstSize
	stats["success_count"] = a.successCount
	stats["error_count"] = a.errorCount
	
	if a.responseCount > 0 {
		stats["avg_response_time"] = a.responseTimeSum / time.Duration(a.responseCount)
	}
	
	totalRequests := a.successCount + a.errorCount
	if totalRequests > 0 {
		stats["error_rate"] = float64(a.errorCount) / float64(totalRequests)
	}
	
	return stats
}