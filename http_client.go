package threads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPClient wraps the standard HTTP client with additional functionality.
//
// rateLimiter is stored in an atomic.Pointer so Client.DisableRateLimiting
// and Client.EnableRateLimiting can swap it safely under a concurrent
// request path without tearing. Readers must go through getRateLimiter().
type HTTPClient struct {
	mu                  sync.RWMutex
	client              *http.Client
	logger              Logger
	retryConfig         *RetryConfig
	rateLimiter         atomic.Pointer[RateLimiter]
	baseURL             string
	userAgent           string
	maxResponseBodySize int64
}

// RequestOptions holds options for HTTP requests
type RequestOptions struct {
	Method      string
	Path        string
	QueryParams url.Values
	Body        interface{}
	Headers     map[string]string
	Context     context.Context
}

// Response wraps HTTP response with additional metadata
type Response struct {
	*http.Response
	Body       []byte
	RequestID  string
	RateLimit  *RateLimitInfo
	Duration   time.Duration
	StatusCode int
}

// RateLimitInfo contains rate limiting information from response headers
type RateLimitInfo struct {
	Limit      int           `json:"limit"`
	Remaining  int           `json:"remaining"`
	Reset      time.Time     `json:"reset"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// NewHTTPClient creates a new HTTP client with the provided configuration
func NewHTTPClient(config *Config, rateLimiter *RateLimiter) *HTTPClient {
	h := &HTTPClient{}
	h.updateConfig(config)
	h.rateLimiter.Store(rateLimiter)
	return h
}

func defaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2,
	}
}

func effectiveRetryConfig(config *RetryConfig) *RetryConfig {
	if config != nil {
		return config
	}
	retryConfig := defaultRetryConfig()
	return &retryConfig
}

// updateConfig atomically applies configuration used by future requests.
// Requests already in flight continue with their original snapshot.
func (h *HTTPClient) updateConfig(config *Config) {
	timeout := DefaultHTTPTimeout
	baseURL := BaseAPIURL
	userAgent := DefaultUserAgent
	maxResponseBodySize := int64(DefaultMaxResponseBodySize)
	var logger Logger
	var retryConfig *RetryConfig

	if config != nil {
		if config.HTTPTimeout > 0 {
			timeout = config.HTTPTimeout
		}
		if config.BaseURL != "" {
			baseURL = config.BaseURL
		}
		if config.UserAgent != "" {
			userAgent = config.UserAgent
		}
		if config.MaxResponseBodySize > 0 {
			maxResponseBodySize = config.MaxResponseBodySize
		}
		logger = config.Logger
		retryConfig = effectiveRetryConfig(config.RetryConfig)
	} else {
		retryConfig = effectiveRetryConfig(nil)
	}

	h.mu.Lock()
	h.client = &http.Client{Timeout: timeout}
	h.logger = logger
	h.retryConfig = retryConfig
	h.baseURL = strings.TrimRight(baseURL, "/")
	h.userAgent = userAgent
	h.maxResponseBodySize = maxResponseBodySize
	h.mu.Unlock()
}

type httpConfigSnapshot struct {
	client              *http.Client
	logger              Logger
	retryConfig         RetryConfig
	baseURL             string
	userAgent           string
	maxResponseBodySize int64
}

func (h *HTTPClient) configSnapshot() httpConfigSnapshot {
	h.mu.RLock()
	snapshot := httpConfigSnapshot{
		client:              h.client,
		logger:              h.logger,
		baseURL:             h.baseURL,
		userAgent:           h.userAgent,
		maxResponseBodySize: h.maxResponseBodySize,
		retryConfig:         defaultRetryConfig(),
	}
	if h.retryConfig != nil {
		snapshot.retryConfig = *h.retryConfig
	}
	h.mu.RUnlock()

	if snapshot.client == nil {
		snapshot.client = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	if snapshot.baseURL == "" {
		snapshot.baseURL = BaseAPIURL
	}
	if snapshot.userAgent == "" {
		snapshot.userAgent = DefaultUserAgent
	}
	if snapshot.maxResponseBodySize <= 0 {
		snapshot.maxResponseBodySize = DefaultMaxResponseBodySize
	}
	return snapshot
}

func (h *HTTPClient) getRateLimiter() *RateLimiter {
	return h.rateLimiter.Load()
}

func (h *HTTPClient) setRateLimiter(rl *RateLimiter) {
	h.rateLimiter.Store(rl)
}

// Do executes an HTTP request with retry logic and error handling
func (h *HTTPClient) Do(opts *RequestOptions, accessToken string) (*Response, error) {
	if opts.Context == nil {
		opts.Context = context.Background()
	}

	// Only wait for rate limiter if we've been explicitly rate limited by the API
	if rl := h.getRateLimiter(); rl != nil && rl.ShouldWait() {
		if err := rl.Wait(opts.Context); err != nil {
			return nil, fmt.Errorf("rate limiter wait failed: %w", err)
		}
	}

	requestConfig := h.configSnapshot()
	var lastErr error
	maxRetries := requestConfig.retryConfig.MaxRetries
	delay := requestConfig.retryConfig.InitialDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-opts.Context.Done():
				return nil, opts.Context.Err()
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * requestConfig.retryConfig.BackoffFactor)
			if delay > requestConfig.retryConfig.MaxDelay {
				delay = requestConfig.retryConfig.MaxDelay
			}
		}

		resp, err := h.executeRequest(opts, accessToken, requestConfig)
		if err != nil {
			lastErr = err

			// Check if error is retry-able
			if !h.isRetryableError(err) {
				return nil, err
			}

			h.logRetry(attempt, maxRetries, err)
			continue
		}

		// Update rate limiter with response headers
		if rl := h.getRateLimiter(); rl != nil && resp.RateLimit != nil {
			rl.UpdateFromHeaders(resp.RateLimit)
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// executeRequest performs a single HTTP request
func (h *HTTPClient) executeRequest(opts *RequestOptions, accessToken string, requestConfig httpConfigSnapshot) (*Response, error) {
	startTime := time.Now()

	if err := validateBaseURL(requestConfig.baseURL); err != nil {
		return nil, fmt.Errorf("invalid BaseURL: %w", err)
	}

	// Build URL
	fullURL, err := buildRequestURL(requestConfig.baseURL, opts.Path, opts.QueryParams)
	if err != nil {
		return nil, err
	}

	// Prepare request body
	var bodyReader io.Reader
	var contentType string

	if opts.Body != nil {
		switch body := opts.Body.(type) {
		case string:
			bodyReader = strings.NewReader(body)
			contentType = "text/plain"
		case []byte:
			bodyReader = bytes.NewReader(body)
			contentType = "application/octet-stream"
		case url.Values:
			bodyReader = strings.NewReader(body.Encode())
			contentType = "application/x-www-form-urlencoded"
		default:
			// JSON encode by default
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
			contentType = "application/json"
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(opts.Context, opts.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", requestConfig.userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	// Add custom headers
	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	// Log request
	h.logRequest(req, opts.Body)

	// Execute request
	httpResp, err := requestConfig.client.Do(req)
	if err != nil {
		return nil, h.wrapNetworkError(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil && requestConfig.logger != nil {
			requestConfig.logger.Error("Failed to close response body", "error", err)
		}
	}(httpResp.Body)

	// Read at most the configured limit plus one byte so oversized responses
	// are rejected without allowing an untrusted server to exhaust memory.
	readLimit := requestConfig.maxResponseBodySize
	if readLimit < math.MaxInt64 {
		readLimit++
	}
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, readLimit))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if int64(len(respBody)) > requestConfig.maxResponseBodySize {
		return nil, fmt.Errorf("%w: limit is %d bytes", ErrResponseTooLarge, requestConfig.maxResponseBodySize)
	}

	// Create response wrapper
	resp := &Response{
		Response:   httpResp,
		Body:       respBody,
		RequestID:  httpResp.Header.Get("X-Fb-Request-Id"),
		StatusCode: httpResp.StatusCode,
		Duration:   time.Since(startTime),
		RateLimit:  h.parseRateLimitHeaders(httpResp.Header),
	}

	// Log response
	h.logResponse(resp)

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		return resp, h.createErrorFromResponse(resp)
	}

	return resp, nil
}

func buildRequestURL(baseURL, path string, queryParams url.Values) (string, error) {
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("request path must start with /")
	}

	pathURL, err := url.Parse(path)
	if err != nil || pathURL.IsAbs() || pathURL.Host != "" || pathURL.RawQuery != "" || pathURL.ForceQuery || pathURL.Fragment != "" {
		return "", fmt.Errorf("request path must not contain an origin, query, or fragment")
	}

	requestURL, err := url.Parse(strings.TrimRight(baseURL, "/") + pathURL.EscapedPath())
	if err != nil {
		return "", fmt.Errorf("failed to build request URL: %w", err)
	}
	requestURL.RawQuery = queryParams.Encode()
	return requestURL.String(), nil
}

// parseRateLimitHeaders extracts rate limit information from response headers
func (h *HTTPClient) parseRateLimitHeaders(headers http.Header) *RateLimitInfo {
	rateLimitInfo := &RateLimitInfo{}

	if limitStr := headers.Get("X-RateLimit-Limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			rateLimitInfo.Limit = limit
		}
	}

	if remainingStr := headers.Get("X-RateLimit-Remaining"); remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil {
			rateLimitInfo.Remaining = remaining
		}
	}

	if resetStr := headers.Get("X-RateLimit-Reset"); resetStr != "" {
		if resetTime, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			rateLimitInfo.Reset = time.Unix(resetTime, 0)
		}
	}

	if retryAfterStr := headers.Get("Retry-After"); retryAfterStr != "" {
		if retryAfter, err := strconv.Atoi(retryAfterStr); err == nil {
			rateLimitInfo.RetryAfter = time.Duration(retryAfter) * time.Second
		}
	}

	// Return nil if no rate limit headers found
	if rateLimitInfo.Limit == 0 && rateLimitInfo.Remaining == 0 && rateLimitInfo.Reset.IsZero() {
		return nil
	}

	return rateLimitInfo
}

// createErrorFromResponse creates appropriate error types based on HTTP response
func (h *HTTPClient) createErrorFromResponse(resp *Response) error {
	var apiErr struct {
		Error struct {
			Message      string `json:"message"`
			Type         string `json:"type"`
			Code         int    `json:"code"`
			IsTransient  bool   `json:"is_transient"`
			ErrorSubcode int    `json:"error_subcode"`
		} `json:"error"`
	}

	// Try to parse error response
	message := fmt.Sprintf("HTTP %d", resp.StatusCode)
	errorCode := resp.StatusCode
	isTransient := false

	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &apiErr); err == nil && apiErr.Error.Message != "" {
			message = apiErr.Error.Message
			isTransient = apiErr.Error.IsTransient
			if apiErr.Error.Code != 0 {
				errorCode = apiErr.Error.Code
			}
		}
	}

	details := string(resp.Body)
	if len(details) > 500 {
		details = details[:500] + "..."
	}

	// API-level auth codes take priority over HTTP status. Meta returns 5xx
	// for token failures on some endpoints (e.g. /debug_token returns HTTP 500
	// for error 190), which would otherwise be misclassified as retryable 5xx.
	var resultErr error
	if isAuthErrorCode(errorCode) {
		authErr := NewAuthenticationError(errorCode, message, details)
		setErrorMetadata(authErr, false, resp.StatusCode, apiErr.Error.ErrorSubcode)
		return authErr
	}
	switch resp.StatusCode {
	case 401, 403:
		resultErr = NewAuthenticationError(errorCode, message, details)
	case 429:
		retryAfter := time.Duration(0)
		resetTime := time.Time{}
		if resp.RateLimit != nil {
			if resp.RateLimit.RetryAfter > 0 {
				retryAfter = resp.RateLimit.RetryAfter
			}
			if !resp.RateLimit.Reset.IsZero() {
				resetTime = resp.RateLimit.Reset
			}
		}

		// Mark the rate limiter as rate limited by the API
		if rl := h.getRateLimiter(); rl != nil {
			if resetTime.IsZero() {
				// If no reset time provided, estimate based on retry after
				resetTime = time.Now().Add(retryAfter)
			}
			rl.MarkRateLimited(resetTime)
		}

		resultErr = NewRateLimitError(errorCode, message, details, retryAfter)
	case 400, 422:
		resultErr = NewValidationError(errorCode, message, details, "")
	default:
		resultErr = NewAPIError(errorCode, message, details, resp.RequestID)
	}

	setErrorMetadata(resultErr, isTransient, resp.StatusCode, apiErr.Error.ErrorSubcode)

	return resultErr
}

// isAuthErrorCode reports whether code is a well-known Meta/Threads top-level
// auth error that should always map to AuthenticationError, regardless of HTTP
// status. Meta returns 5xx for these on some endpoints (e.g. /debug_token).
// Subcodes (463 = token expired, 467 = missing permissions) nest under 190
// and are handled by the caller via setErrorMetadata.
func isAuthErrorCode(code int) bool {
	switch code {
	case 190, // Invalid/revoked/expired access token (subcodes 463, 467)
		102: // Session invalidated / login required
		return true
	}
	return false
}

// isNonRetryablePermanentErrorCode reports whether the given Meta error code
// is a permanent failure that won't be fixed by retrying. Meta returns HTTP
// 5xx for these on some endpoints (notably /threads_publish), which would
// otherwise be misclassified as a transient 5xx and retried — wasting API
// quota that's counted per attempt.
//
// Auth codes (190, 102) are deliberately not listed here; they go through the
// IsAuthenticationError() short-circuit at the top of isRetryableError.
func isNonRetryablePermanentErrorCode(code int) bool {
	switch code {
	case 10:
		// GraphMethodException — "Application does not have permission for
		// this action". Meta returns this from /threads_publish for some
		// app/permission configurations; per the API troubleshooting docs
		// the publish may actually have succeeded even when this is
		// returned, so callers can recover by checking container status.
		// Retrying the publish never helps and burns publishing quota.
		return true
	}
	return false
}

// wrapNetworkError wraps network errors with appropriate error types.
// The original error is preserved as the Cause, so errors.Is/errors.As
// can inspect it (e.g., to detect context.Canceled).
func (h *HTTPClient) wrapNetworkError(err error) error {
	err = sanitizeNetworkError(err)

	// Check for timeout errors
	if timeoutErr, ok := err.(interface{ Timeout() bool }); ok && timeoutErr.Timeout() {
		return NewNetworkErrorWithCause(0, "Request timeout", err.Error(), true, err)
	}

	// Check for temporary errors
	if tempErr, ok := err.(interface{ Temporary() bool }); ok && tempErr.Temporary() {
		return NewNetworkErrorWithCause(0, "Temporary network error", err.Error(), true, err)
	}

	// Default to permanent network error
	return NewNetworkErrorWithCause(0, "Network error", err.Error(), false, err)
}

// sanitizeNetworkError preserves the error chain while redacting credentials
// from URLs embedded by net/http in *url.Error values.
func sanitizeNetworkError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return err
	}

	safeErr := *urlErr
	u, parseErr := url.Parse(urlErr.URL)
	if parseErr != nil {
		safeErr.URL = "[REDACTED]"
	} else {
		safeErr.URL = sanitizeURL(u)
	}
	return &safeErr
}

// isRetryableError determines if an error should trigger a retry
func (h *HTTPClient) isRetryableError(err error) bool {
	// Authentication errors are never retryable — the token is invalid and
	// no amount of retrying will fix it. Guard this before the status-code
	// check below: setErrorMetadata stores the HTTP status (e.g. 500 from
	// graph.threads.net) on AuthenticationError's embedded BaseError, so the
	// HTTPStatusCode >= 500 branch would otherwise re-enable retries.
	if IsAuthenticationError(err) {
		return false
	}

	// Rate limit errors are retry-able
	if IsRateLimitError(err) {
		return true
	}

	// Temporary network errors are retry-able
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.Temporary || netErr.IsTransient
	}

	// Check base error for transient flag or 5xx HTTP status
	baseErr := extractBaseError(err)
	if baseErr != nil {
		// Permanent error codes short-circuit the 5xx branch below. Meta
		// returns 5xx for some non-transient API codes (e.g. code 10 from
		// /threads_publish); without this check we'd retry and burn quota.
		if isNonRetryablePermanentErrorCode(baseErr.Code) {
			return false
		}
		if baseErr.IsTransient {
			return true
		}
		if baseErr.HTTPStatusCode >= 500 && baseErr.HTTPStatusCode < 600 {
			return true
		}
	}

	return false
}

// logRequest logs the outgoing HTTP request
func (h *HTTPClient) logRequest(req *http.Request, body interface{}) {
	logger := h.configSnapshot().logger
	if logger == nil {
		return
	}

	fields := []interface{}{
		"method", req.Method,
		"url", sanitizeURL(req.URL),
		"headers", h.sanitizeHeaders(req.Header),
	}

	if body != nil {
		// Don't log sensitive data
		if req.Header.Get("Content-Type") == "application/json" {
			fields = append(fields, "body_type", "json")
		} else {
			fields = append(fields, "body_type", fmt.Sprintf("%T", body))
		}
	}

	logger.Debug("HTTP request", fields...)
}

// logResponse logs the HTTP response
func (h *HTTPClient) logResponse(resp *Response) {
	logger := h.configSnapshot().logger
	if logger == nil {
		return
	}

	fields := []interface{}{
		"status_code", resp.StatusCode,
		"duration_ms", resp.Duration.Milliseconds(),
		"request_id", resp.RequestID,
	}

	if resp.RateLimit != nil {
		fields = append(fields,
			"rate_limit_remaining", resp.RateLimit.Remaining,
			"rate_limit_limit", resp.RateLimit.Limit,
		)
	}

	if resp.StatusCode >= 400 {
		fields = append(fields, "response_body", string(resp.Body))
		logger.Error("HTTP response error", fields...)
	} else {
		logger.Debug("HTTP response", fields...)
	}
}

// logRetry logs retry attempts
func (h *HTTPClient) logRetry(attempt, maxRetries int, err error) {
	logger := h.configSnapshot().logger
	if logger == nil {
		return
	}

	logger.Warn("HTTP request retry",
		"attempt", attempt+1,
		"max_retries", maxRetries+1,
		"error", err.Error(),
	)
}

// sanitizeHeaders removes sensitive headers from logging
func (h *HTTPClient) sanitizeHeaders(headers http.Header) map[string]string {
	sanitized := make(map[string]string)
	for key, values := range headers {
		if strings.ToLower(key) == "authorization" {
			sanitized[key] = "[REDACTED]"
		} else {
			sanitized[key] = strings.Join(values, ", ")
		}
	}
	return sanitized
}

// sensitiveQueryParams is the set of query-parameter names that may carry
// bearer tokens or app secrets and must be redacted before logging. The
// Threads API token/refresh/debug endpoints accept these as query params
// (see GetLongLivedToken, RefreshToken, DebugToken, GetAppAccessToken).
var sensitiveQueryParams = map[string]struct{}{
	"access_token":  {},
	"client_secret": {},
	"input_token":   {},
	"code":          {},
	"refresh_token": {},
}

// sanitizeURL returns a version of u safe to write to logs: any query
// parameter in sensitiveQueryParams is replaced with [REDACTED]. The original
// URL is not mutated.
func sanitizeURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if u.RawQuery == "" {
		return u.String()
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		clone := *u
		clone.RawQuery = "[REDACTED]"
		return clone.String()
	}
	redacted := false
	for name := range q {
		if _, ok := sensitiveQueryParams[strings.ToLower(name)]; ok {
			q.Set(name, "[REDACTED]")
			redacted = true
		}
	}
	if !redacted {
		return u.String()
	}
	clone := *u
	clone.RawQuery = q.Encode()
	return clone.String()
}

// GET performs a GET request
func (h *HTTPClient) GET(path string, queryParams url.Values, accessToken string) (*Response, error) {
	return h.Do(&RequestOptions{
		Method:      "GET",
		Path:        path,
		QueryParams: queryParams,
	}, accessToken)
}

// POST performs a POST request
func (h *HTTPClient) POST(path string, body interface{}, accessToken string) (*Response, error) {
	return h.Do(&RequestOptions{
		Method: "POST",
		Path:   path,
		Body:   body,
	}, accessToken)
}

// PUT performs a PUT request
func (h *HTTPClient) PUT(path string, body interface{}, accessToken string) (*Response, error) {
	return h.Do(&RequestOptions{
		Method: "PUT",
		Path:   path,
		Body:   body,
	}, accessToken)
}

// DELETE performs a DELETE request
func (h *HTTPClient) DELETE(path string, accessToken string) (*Response, error) {
	return h.Do(&RequestOptions{
		Method: "DELETE",
		Path:   path,
	}, accessToken)
}
