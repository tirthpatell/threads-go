package threads

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiter_NotRateLimitedByDefault(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	if rl.ShouldWait() {
		t.Error("should not wait by default")
	}
	if rl.IsRateLimited() {
		t.Error("should not be rate limited by default")
	}
}

func TestRateLimiter_MarkRateLimited(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.MarkRateLimited(time.Now().Add(30 * time.Second))
	if !rl.IsRateLimited() {
		t.Error("expected to be rate limited after marking")
	}
	if !rl.ShouldWait() {
		t.Error("expected to should wait after being rate limited")
	}
}

func TestRateLimiter_MarkRateLimited_WithLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, Logger: &noopLogger{}})
	rl.MarkRateLimited(time.Now().Add(30 * time.Second))
	if !rl.IsRateLimited() {
		t.Error("expected to be rate limited")
	}
}

func TestRateLimiter_WaitRespectsContext(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.MarkRateLimited(time.Now().Add(10 * time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Fatal("expected context timeout error")
	}
}

func TestRateLimiter_WaitAfterReset(t *testing.T) {
	// When resetTime is in the past, Wait should return immediately
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.mu.Lock()
	rl.resetTime = time.Now().Add(-time.Second) // Already past
	rl.rateLimited = true
	rl.mu.Unlock()

	err := rl.Wait(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimiter_WaitNotRateLimited(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	// Not rate limited, should return immediately
	err := rl.Wait(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimiter_WaitWithBackoff(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{
		InitialLimit:      100,
		BackoffMultiplier: 2.0,
		MaxBackoff:        5 * time.Second,
	})
	// Set up recently rate limited state to trigger backoff
	rl.mu.Lock()
	rl.rateLimited = true
	rl.lastRateLimitTime = time.Now() // recently rate limited
	rl.lastRequestTime = time.Now()   // recently requested
	rl.resetTime = time.Now().Add(50 * time.Millisecond)
	rl.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	// Should either succeed (backoff < 200ms) or timeout
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimiter_WaitWithResetInPast(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, Logger: &noopLogger{}})
	rl.mu.Lock()
	rl.resetTime = time.Now().Add(-time.Minute) // already reset
	rl.mu.Unlock()

	err := rl.Wait(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimiter_UpdateFromHeaders(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.UpdateFromHeaders(&RateLimitInfo{
		Limit:     200,
		Remaining: 150,
		Reset:     time.Now().Add(time.Hour),
	})

	status := rl.GetStatus()
	if status.Limit != 200 {
		t.Errorf("expected limit 200, got %d", status.Limit)
	}
	if status.Remaining != 150 {
		t.Errorf("expected remaining 150, got %d", status.Remaining)
	}
}

func TestRateLimiter_UpdateFromHeaders_Nil(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.UpdateFromHeaders(nil)
	status := rl.GetStatus()
	if status.Limit != 100 {
		t.Errorf("expected limit unchanged at 100, got %d", status.Limit)
	}
}

func TestRateLimiter_IsNearLimit(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.UpdateFromHeaders(&RateLimitInfo{Limit: 100, Remaining: 10})

	if !rl.IsNearLimit(0.8) {
		t.Error("expected near limit at 80% threshold")
	}
	if rl.IsNearLimit(0.95) {
		t.Error("expected not near limit at 95% threshold")
	}
}

func TestRateLimiter_IsNearLimit_ZeroLimit(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.mu.Lock()
	rl.limit = 0
	rl.mu.Unlock()
	if rl.IsNearLimit(0.8) {
		t.Error("expected not near limit with zero limit")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.MarkRateLimited(time.Now().Add(time.Hour))
	_ = rl.QueueRequest(context.Background())
	rl.Reset()
	if rl.IsRateLimited() {
		t.Error("expected not rate limited after reset")
	}
	if rl.GetQueueLength() != 0 {
		t.Errorf("expected queue length 0 after reset, got %d", rl.GetQueueLength())
	}
}

func TestRateLimiter_QueueRequest(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, QueueSize: 2})
	_ = rl.QueueRequest(context.Background())
	_ = rl.QueueRequest(context.Background())

	err := rl.QueueRequest(context.Background())
	if err == nil {
		t.Fatal("expected error when queue is full")
	}
}

func TestRateLimiter_QueueRequest_ContextCanceled(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, QueueSize: 1})
	_ = rl.QueueRequest(context.Background()) // fill queue

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	err := rl.QueueRequest(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRateLimiter_GetQueueLength(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, QueueSize: 10})
	if rl.GetQueueLength() != 0 {
		t.Errorf("expected 0, got %d", rl.GetQueueLength())
	}
	_ = rl.QueueRequest(context.Background())
	if rl.GetQueueLength() != 1 {
		t.Errorf("expected 1, got %d", rl.GetQueueLength())
	}
	_ = rl.QueueRequest(context.Background())
	if rl.GetQueueLength() != 2 {
		t.Errorf("expected 2, got %d", rl.GetQueueLength())
	}
}

func TestRateLimiter_ProcessQueue(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, QueueSize: 10})

	// Add items to queue
	_ = rl.QueueRequest(context.Background())
	_ = rl.QueueRequest(context.Background())

	var processed int32

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() {
		_ = rl.ProcessQueue(ctx, func() error {
			atomic.AddInt32(&processed, 1)
			return nil
		})
	}()

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&processed) != 2 {
		t.Errorf("expected 2 processed, got %d", atomic.LoadInt32(&processed))
	}
}

func TestRateLimiter_ProcessQueue_WithError(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, QueueSize: 10, Logger: &noopLogger{}})
	_ = rl.QueueRequest(context.Background())
	_ = rl.QueueRequest(context.Background())

	var processed int32
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() {
		_ = rl.ProcessQueue(ctx, func() error {
			count := atomic.AddInt32(&processed, 1)
			if count == 1 {
				return errors.New("test error")
			}
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)
	// Both should be processed even though first errored
	if atomic.LoadInt32(&processed) != 2 {
		t.Errorf("expected 2 processed (even with error), got %d", atomic.LoadInt32(&processed))
	}
}

func TestRateLimiter_ProcessQueue_ContextCancel(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, QueueSize: 10})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := rl.ProcessQueue(ctx, func() error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRateLimiter_CalculateBackoff(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{
		InitialLimit:      100,
		BackoffMultiplier: 2.0,
		MaxBackoff:        5 * time.Minute,
	})

	// When last request was recent, should apply multiplier
	rl.mu.Lock()
	rl.lastRequestTime = time.Now()
	rl.mu.Unlock()

	backoff := rl.calculateBackoff()
	if backoff != 2*time.Second {
		t.Errorf("expected 2s backoff, got %v", backoff)
	}
}

func TestRateLimiter_CalculateBackoff_NoRecentRequest(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{
		InitialLimit:      100,
		BackoffMultiplier: 2.0,
		MaxBackoff:        5 * time.Minute,
	})

	// When last request was long ago, should use base delay
	rl.mu.Lock()
	rl.lastRequestTime = time.Now().Add(-2 * time.Minute)
	rl.mu.Unlock()

	backoff := rl.calculateBackoff()
	if backoff != time.Second {
		t.Errorf("expected 1s base delay, got %v", backoff)
	}
}

func TestRateLimiter_CalculateBackoff_MaxBackoff(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{
		InitialLimit:      100,
		BackoffMultiplier: 1000.0, // very high multiplier
		MaxBackoff:        3 * time.Second,
	})

	rl.mu.Lock()
	rl.lastRequestTime = time.Now()
	rl.mu.Unlock()

	backoff := rl.calculateBackoff()
	if backoff > 3*time.Second {
		t.Errorf("expected backoff capped at 3s, got %v", backoff)
	}
}

func TestRateLimiter_LogRateLimitReset_NilLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	// Should not panic
	rl.logRateLimitReset()
}

func TestRateLimiter_LogRateLimitReset_WithLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, Logger: &noopLogger{}})
	rl.logRateLimitReset()
}

func TestRateLimiter_LogQueueProcessError_NilLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	// Should not panic
	rl.logQueueProcessError(errors.New("test"))
}

func TestRateLimiter_LogQueueProcessError_WithLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, Logger: &noopLogger{}})
	rl.logQueueProcessError(errors.New("test"))
}

func TestRateLimiter_LogRateLimitUpdate_NilLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.logRateLimitUpdate(&RateLimitInfo{Limit: 100, Remaining: 50})
}

func TestRateLimiter_LogRateLimitUpdate_WithLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, Logger: &noopLogger{}})
	rl.logRateLimitUpdate(&RateLimitInfo{Limit: 100, Remaining: 50})
}

func TestRateLimiter_LogRateLimitWait_NilLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.logRateLimitWait(5 * time.Second)
}

func TestRateLimiter_LogRateLimitWait_WithLogger(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, Logger: &noopLogger{}})
	rl.logRateLimitWait(5 * time.Second)
}

func TestRateLimiter_DefaultConfig(t *testing.T) {
	// Test defaults get applied when zero values given
	rl := NewRateLimiter(&RateLimiterConfig{})
	status := rl.GetStatus()
	if status.Limit != 100 {
		t.Errorf("expected default limit 100, got %d", status.Limit)
	}
}

func TestRateLimiter_WaitCompletesAfterReset(t *testing.T) {
	// Rate limited but reset time is very soon, so Wait completes
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100, Logger: &noopLogger{}})
	rl.mu.Lock()
	rl.rateLimited = true
	rl.lastRateLimitTime = time.Now().Add(-2 * time.Minute) // not recent
	rl.resetTime = time.Now().Add(50 * time.Millisecond)
	rl.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Fatalf("expected Wait to complete successfully, got: %v", err)
	}
	// After Wait completes, should no longer be rate limited
	if rl.IsRateLimited() {
		t.Error("expected not rate limited after Wait completes")
	}
}

func TestRateLimiter_WaitWithFrequentRateLimiting(t *testing.T) {
	// Test the branch where lastRateLimitTime is recent (< 1 minute) to trigger backoff
	rl := NewRateLimiter(&RateLimiterConfig{
		InitialLimit:      100,
		BackoffMultiplier: 1.5,
		MaxBackoff:        100 * time.Millisecond,
		Logger:            &noopLogger{},
	})
	rl.mu.Lock()
	rl.rateLimited = true
	rl.lastRateLimitTime = time.Now()                    // recently rate limited
	rl.lastRequestTime = time.Now()                      // recent request
	rl.resetTime = time.Now().Add(20 * time.Millisecond) // reset soon
	rl.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDisableRateLimiting_PropagatesToHTTPClient guards against a regression
// where Client.DisableRateLimiting nulled c.rateLimiter but left the
// HTTPClient's own rate-limiter reference intact, so the request path kept
// consulting the old limiter. After disable, the HTTPClient must see nil.
func TestDisableRateLimiting_PropagatesToHTTPClient(t *testing.T) {
	config := &Config{
		ClientID:     "id",
		ClientSecret: "secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if client.httpClient.getRateLimiter() == nil {
		t.Fatal("precondition: HTTPClient should start with a rate limiter")
	}

	client.DisableRateLimiting()

	if rl := client.httpClient.getRateLimiter(); rl != nil {
		t.Errorf("HTTPClient still has a rate limiter after Disable: %p", rl)
	}
	if client.IsRateLimited() {
		t.Error("IsRateLimited must report false when limiter is disabled")
	}
	if client.IsNearRateLimit(0.5) {
		t.Error("IsNearRateLimit must report false when limiter is disabled")
	}

	client.EnableRateLimiting()
	if client.httpClient.getRateLimiter() == nil {
		t.Error("HTTPClient must have a rate limiter again after Enable")
	}
}
