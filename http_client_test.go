package threads

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClient_RetryOnServerError(t *testing.T) {
	var attempts int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":{"message":"Internal error","type":"OAuthException","code":2,"is_transient":true}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      50 * time.Millisecond,
		BackoffFactor: 2.0,
	})

	resp, err := httpClient.Do(&RequestOptions{Method: "GET", Path: "/test"}, "token")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestHTTPClient_NoRetryOnValidationError(t *testing.T) {
	var attempts int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      50 * time.Millisecond,
		BackoffFactor: 2.0,
	})

	_, err := httpClient.Do(&RequestOptions{Method: "GET", Path: "/test"}, "token")
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt (no retry for 400), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestHTTPClient_ContextCancellation(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
		w.WriteHeader(200)
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries:    0,
		InitialDelay:  time.Second,
		MaxDelay:      time.Second,
		BackoffFactor: 1.0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := httpClient.Do(&RequestOptions{Method: "GET", Path: "/slow", Context: ctx}, "token")
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestHTTPClient_ParseRateLimitHeaders(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "42")
		w.Header().Set("X-RateLimit-Reset", "1735689600")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 1.0,
	})

	resp, err := httpClient.Do(&RequestOptions{Method: "GET", Path: "/test"}, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RateLimit == nil {
		t.Fatal("expected rate limit info")
	}
	if resp.RateLimit.Limit != 100 {
		t.Errorf("expected limit 100, got %d", resp.RateLimit.Limit)
	}
	if resp.RateLimit.Remaining != 42 {
		t.Errorf("expected remaining 42, got %d", resp.RateLimit.Remaining)
	}
}
