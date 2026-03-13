package threads

import (
	"context"
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

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	rl.MarkRateLimited(time.Now().Add(time.Hour))
	rl.Reset()
	if rl.IsRateLimited() {
		t.Error("expected not rate limited after reset")
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
