package threads

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
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

func TestHTTPClient_PUT(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 1.0,
	})

	resp, err := httpClient.PUT("/test", url.Values{"key": {"value"}}, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHTTPClient_DELETE(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 1.0,
	})

	resp, err := httpClient.DELETE("/test", "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHTTPClient_POST_URLValues(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 1.0,
	})

	_, err := httpClient.POST("/test", url.Values{"key": {"val"}}, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_POST_JSONBody(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 1.0,
	})

	// Passing a struct should be JSON-encoded
	_, err := httpClient.POST("/test", map[string]string{"key": "value"}, "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_POST_StringBody(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("expected text/plain, got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 1.0,
	})

	_, err := httpClient.POST("/test", "hello body", "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_POST_ByteBody(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("expected application/octet-stream, got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}

	httpClient := newTestHTTPClient(t, http.HandlerFunc(handler), &RetryConfig{
		MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 1.0,
	})

	_, err := httpClient.POST("/test", []byte("binary data"), "token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrapNetworkError_Timeout(t *testing.T) {
	httpClient := &HTTPClient{
		logger: &noopLogger{},
	}
	// Create a timeout error
	timeoutErr := &net.DNSError{IsTimeout: true}
	wrapped := httpClient.wrapNetworkError(timeoutErr)
	var netErr *NetworkError
	if !errors.As(wrapped, &netErr) {
		t.Fatalf("expected NetworkError, got %T", wrapped)
	}
	if !netErr.Temporary {
		t.Error("expected temporary error for timeout")
	}
}

func TestWrapNetworkError_Generic(t *testing.T) {
	httpClient := &HTTPClient{
		logger: &noopLogger{},
	}
	wrapped := httpClient.wrapNetworkError(errors.New("some network problem"))
	var netErr *NetworkError
	if !errors.As(wrapped, &netErr) {
		t.Fatalf("expected NetworkError, got %T", wrapped)
	}
	if netErr.Temporary {
		t.Error("expected non-temporary error for generic error")
	}
}

func TestCreateErrorFromResponse_401(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	resp := &Response{
		StatusCode: 401,
		Body:       []byte(`{"error":{"message":"Unauthorized","type":"OAuthException","code":190}}`),
	}
	err := httpClient.createErrorFromResponse(resp)
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestCreateErrorFromResponse_429(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	resp := &Response{
		StatusCode: 429,
		Body:       []byte(`{"error":{"message":"rate limited","type":"rate_limit","code":429}}`),
		RateLimit:  &RateLimitInfo{RetryAfter: 60 * time.Second, Reset: time.Now().Add(time.Minute)},
	}
	err := httpClient.createErrorFromResponse(resp)
	if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestCreateErrorFromResponse_429_NoRateLimit(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	resp := &Response{
		StatusCode: 429,
		Body:       []byte(`{"error":{"message":"rate limited"}}`),
	}
	err := httpClient.createErrorFromResponse(resp)
	if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestCreateErrorFromResponse_429_WithRateLimiter(t *testing.T) {
	rl := NewRateLimiter(&RateLimiterConfig{InitialLimit: 100})
	httpClient := &HTTPClient{logger: &noopLogger{}, rateLimiter: rl}
	resp := &Response{
		StatusCode: 429,
		Body:       []byte(`{"error":{"message":"rate limited"}}`),
		RateLimit:  &RateLimitInfo{RetryAfter: 30 * time.Second},
	}
	_ = httpClient.createErrorFromResponse(resp)
	if !rl.IsRateLimited() {
		t.Error("expected rate limiter to be marked as rate limited")
	}
}

func TestCreateErrorFromResponse_400(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	resp := &Response{
		StatusCode: 400,
		Body:       []byte(`{"error":{"message":"Bad request","type":"validation","code":100}}`),
	}
	err := httpClient.createErrorFromResponse(resp)
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateErrorFromResponse_422(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	resp := &Response{
		StatusCode: 422,
		Body:       []byte(`{"error":{"message":"Unprocessable","type":"validation","code":100}}`),
	}
	err := httpClient.createErrorFromResponse(resp)
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateErrorFromResponse_500(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	resp := &Response{
		StatusCode: 500,
		Body:       []byte(`{"error":{"message":"Server error","type":"server","code":500,"is_transient":true}}`),
	}
	err := httpClient.createErrorFromResponse(resp)
	if !IsAPIError(err) {
		t.Errorf("expected APIError, got %T", err)
	}
}

func TestCreateErrorFromResponse_EmptyBody(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	resp := &Response{
		StatusCode: 500,
		Body:       nil,
	}
	err := httpClient.createErrorFromResponse(resp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateErrorFromResponse_LongBody(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	// Create a body longer than 500 chars
	longBody := make([]byte, 600)
	for i := range longBody {
		longBody[i] = 'x'
	}
	resp := &Response{
		StatusCode: 500,
		Body:       longBody,
	}
	err := httpClient.createErrorFromResponse(resp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLogRequest_WithLogger(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	// Should not panic
	httpClient.logRequest(req, nil)
}

func TestLogRequest_WithJSONBody(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	req, _ := http.NewRequest("POST", "http://example.com/test", nil)
	req.Header.Set("Content-Type", "application/json")
	httpClient.logRequest(req, map[string]string{"key": "value"})
}

func TestLogRequest_WithNonJSONBody(t *testing.T) {
	httpClient := &HTTPClient{logger: &noopLogger{}}
	req, _ := http.NewRequest("POST", "http://example.com/test", nil)
	req.Header.Set("Content-Type", "text/plain")
	httpClient.logRequest(req, "hello")
}

func TestLogRequest_NilLogger(t *testing.T) {
	httpClient := &HTTPClient{logger: nil}
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	// Should not panic
	httpClient.logRequest(req, nil)
}

func TestHTTPClient_ParseRetryAfterHeader(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("Retry-After", "60")
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
	if resp.RateLimit.RetryAfter != 60*time.Second {
		t.Errorf("expected 60s retry after, got %v", resp.RateLimit.RetryAfter)
	}
}

func TestHTTPClient_NoRateLimitHeaders(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
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
	if resp.RateLimit != nil {
		t.Error("expected nil rate limit info when no headers")
	}
}
