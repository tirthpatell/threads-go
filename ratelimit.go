package threads

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter manages API rate limiting with intelligent backoff
type RateLimiter struct {
	mu                sync.RWMutex
	limit             int           // Maximum requests per window
	remaining         int           // Remaining requests in current window
	resetTime         time.Time     // When the rate limit window resets
	lastRequestTime   time.Time     // Time of last request
	requestQueue      chan struct{} // Channel for queuing requests
	backoffMultiplier float64       // Multiplier for exponential backoff
	maxBackoff        time.Duration // Maximum backoff duration
	logger            Logger        // Logger for rate limit events
	rateLimited       bool          // True if we've been rate limited by the API
	lastRateLimitTime time.Time     // When we were last rate limited
}

// RateLimiterConfig holds configuration for the rate limiter
type RateLimiterConfig struct {
	InitialLimit      int           // Initial rate limit (will be updated from API responses)
	BackoffMultiplier float64       // Exponential backoff multiplier
	MaxBackoff        time.Duration // Maximum backoff duration
	QueueSize         int           // Size of request queue
	Logger            Logger        // Logger instance
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	if config.InitialLimit <= 0 {
		config.InitialLimit = 100 // Default limit
	}
	if config.BackoffMultiplier <= 0 {
		config.BackoffMultiplier = 2.0
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = 5 * time.Minute
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100
	}

	return &RateLimiter{
		limit:             config.InitialLimit,
		remaining:         config.InitialLimit,
		resetTime:         time.Now().Add(time.Hour), // Default 1-hour window
		requestQueue:      make(chan struct{}, config.QueueSize),
		backoffMultiplier: config.BackoffMultiplier,
		maxBackoff:        config.MaxBackoff,
		logger:            config.Logger,
	}
}

// ShouldWait returns true if we should wait before making a request
// Only returns true if we've been explicitly rate limited by the API
func (rl *RateLimiter) ShouldWait() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Clear rate limited flag if the window has reset
	if time.Now().After(rl.resetTime) {
		rl.rateLimited = false
	}

	// Only wait if we've been rate limited and the rate limit hasn't reset yet
	return rl.rateLimited && time.Now().Before(rl.resetTime)
}

// Wait blocks until it's safe to make a request, only when actually rate limited
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if rate limit window has reset
	if time.Now().After(rl.resetTime) {
		rl.remaining = rl.limit
		rl.resetTime = time.Now().Add(time.Hour) // Reset to 1 hour from now
		rl.rateLimited = false                   // Clear rate limited flag
		rl.logRateLimitReset()
		return nil // No need to wait if window has reset
	}

	// Only wait if we've been explicitly rate limited
	if !rl.rateLimited {
		rl.lastRequestTime = time.Now()
		return nil
	}

	// Calculate wait time until reset
	waitTime := time.Until(rl.resetTime)

	// Apply exponential backoff if we're hitting limits frequently
	if time.Since(rl.lastRateLimitTime) < time.Minute {
		backoffTime := rl.calculateBackoff()
		if backoffTime > waitTime {
			waitTime = backoffTime
		}
	}

	rl.logRateLimitWait(waitTime)

	// Wait for either the context to be cancelled or the wait time to elapse
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		// After waiting, clear the rate limited flag
		rl.rateLimited = false
		rl.lastRequestTime = time.Now()
		return nil
	}
}

// UpdateFromHeaders updates rate limit information from API response headers
func (rl *RateLimiter) UpdateFromHeaders(rateLimitInfo *RateLimitInfo) {
	if rateLimitInfo == nil {
		return
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Update rate limit information from headers
	if rateLimitInfo.Limit > 0 {
		rl.limit = rateLimitInfo.Limit
	}

	if rateLimitInfo.Remaining >= 0 {
		rl.remaining = rateLimitInfo.Remaining
	}

	if !rateLimitInfo.Reset.IsZero() {
		rl.resetTime = rateLimitInfo.Reset
	}

	rl.logRateLimitUpdate(rateLimitInfo)
}

// MarkRateLimited marks that we've been rate limited by the API
func (rl *RateLimiter) MarkRateLimited(resetTime time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.rateLimited = true
	rl.lastRateLimitTime = time.Now()

	if !resetTime.IsZero() {
		rl.resetTime = resetTime
	}

	if rl.logger != nil {
		rl.logger.Info("Marked as rate limited by API",
			"reset_time", rl.resetTime.Format(time.RFC3339),
		)
	}
}

// GetStatus returns current rate limit status
func (rl *RateLimiter) GetStatus() RateLimitStatus {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return RateLimitStatus{
		Limit:     rl.limit,
		Remaining: rl.remaining,
		ResetTime: rl.resetTime,
		ResetIn:   time.Until(rl.resetTime),
	}
}

// RateLimitStatus represents the current rate limit status
type RateLimitStatus struct {
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	ResetTime time.Time     `json:"reset_time"`
	ResetIn   time.Duration `json:"reset_in"`
}

// IsNearLimit returns true if we're close to hitting the rate limit
// This is now informational only and doesn't block requests
func (rl *RateLimiter) IsNearLimit(threshold float64) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if rl.limit == 0 {
		return false
	}

	usedPercentage := float64(rl.limit-rl.remaining) / float64(rl.limit)
	return usedPercentage >= threshold
}

// IsRateLimited returns true if we're currently rate limited by the API
func (rl *RateLimiter) IsRateLimited() bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return rl.rateLimited && time.Now().Before(rl.resetTime)
}

// calculateBackoff calculates exponential backoff duration
func (rl *RateLimiter) calculateBackoff() time.Duration {
	// Start with 1-second base delay
	baseDelay := time.Second

	// Calculate how many times we've hit the limit recently
	timeSinceLastRequest := time.Since(rl.lastRequestTime)
	if timeSinceLastRequest < time.Minute {
		// Apply exponential backoff
		backoff := time.Duration(float64(baseDelay) * rl.backoffMultiplier)
		if backoff > rl.maxBackoff {
			backoff = rl.maxBackoff
		}
		return backoff
	}

	return baseDelay
}

// QueueRequest queues a request for later execution when rate limits allow
func (rl *RateLimiter) QueueRequest(ctx context.Context) error {
	select {
	case rl.requestQueue <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("request queue is full (capacity: %d)", cap(rl.requestQueue))
	}
}

// ProcessQueue processes queued requests respecting rate limits
func (rl *RateLimiter) ProcessQueue(ctx context.Context, processor func() error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-rl.requestQueue:
			// Wait for rate limit before processing
			if err := rl.Wait(ctx); err != nil {
				return err
			}

			// Process the request
			if err := processor(); err != nil {
				rl.logQueueProcessError(err)
				// Continue processing other requests even if one fails
				continue
			}
		}
	}
}

// GetQueueLength returns the current number of queued requests
func (rl *RateLimiter) GetQueueLength() int {
	return len(rl.requestQueue)
}

// Reset resets the rate limiter state (useful for testing)
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.remaining = rl.limit
	rl.resetTime = time.Now().Add(time.Hour)
	rl.lastRequestTime = time.Time{}

	// Drain the queue
	for len(rl.requestQueue) > 0 {
		<-rl.requestQueue
	}
}

// Logging methods

func (rl *RateLimiter) logRateLimitUpdate(info *RateLimitInfo) {
	if rl.logger == nil {
		return
	}

	rl.logger.Debug("Rate limit updated",
		"limit", info.Limit,
		"remaining", info.Remaining,
		"reset_time", info.Reset.Format(time.RFC3339),
	)
}

func (rl *RateLimiter) logRateLimitWait(waitTime time.Duration) {
	if rl.logger == nil {
		return
	}

	rl.logger.Info("API rate limit enforced, waiting",
		"wait_duration", waitTime.String(),
		"remaining", rl.remaining,
		"reset_time", rl.resetTime.Format(time.RFC3339),
		"reason", "received_429_from_api",
	)
}

func (rl *RateLimiter) logRateLimitReset() {
	if rl.logger == nil {
		return
	}

	rl.logger.Debug("Rate limit window reset",
		"limit", rl.limit,
		"remaining", rl.remaining,
		"reset_time", rl.resetTime.Format(time.RFC3339),
	)
}

func (rl *RateLimiter) logQueueProcessError(err error) {
	if rl.logger == nil {
		return
	}

	rl.logger.Error("Error processing queued request",
		"error", err.Error(),
		"queue_length", len(rl.requestQueue),
	)
}
