package threads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTPClient wraps the standard HTTP client with additional functionality
type HTTPClient struct {
	client      *http.Client
	logger      Logger
	retryConfig *RetryConfig
	rateLimiter *RateLimiter
	baseURL     string
	userAgent   string
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
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
	}

	return &HTTPClient{
		client:      httpClient,
		logger:      config.Logger,
		retryConfig: config.RetryConfig,
		rateLimiter: rateLimiter,
		baseURL:     "https://graph.threads.net",
		userAgent:   "threads-go-client/1.0",
	}
}

// Do executes an HTTP request with retry logic and error handling
func (h *HTTPClient) Do(opts *RequestOptions, accessToken string) (*Response, error) {
	if opts.Context == nil {
		opts.Context = context.Background()
	}

	// Only wait for rate limiter if we've been explicitly rate limited by the API
	if h.rateLimiter != nil && h.rateLimiter.ShouldWait() {
		if err := h.rateLimiter.Wait(opts.Context); err != nil {
			return nil, fmt.Errorf("rate limiter wait failed: %w", err)
		}
	}

	var lastErr error
	maxRetries := h.retryConfig.MaxRetries
	delay := h.retryConfig.InitialDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-opts.Context.Done():
				return nil, opts.Context.Err()
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * h.retryConfig.BackoffFactor)
			if delay > h.retryConfig.MaxDelay {
				delay = h.retryConfig.MaxDelay
			}
		}

		resp, err := h.executeRequest(opts, accessToken)
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
		if h.rateLimiter != nil && resp.RateLimit != nil {
			h.rateLimiter.UpdateFromHeaders(resp.RateLimit)
		}

		// Check if we should retry based on status code
		if h.shouldRetryStatus(resp.StatusCode) {
			lastErr = h.createErrorFromResponse(resp)
			h.logRetry(attempt, maxRetries, lastErr)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// executeRequest performs a single HTTP request
func (h *HTTPClient) executeRequest(opts *RequestOptions, accessToken string) (*Response, error) {
	startTime := time.Now()

	// Build URL
	fullURL := h.baseURL + opts.Path
	if len(opts.QueryParams) > 0 {
		fullURL += "?" + opts.QueryParams.Encode()
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
	req.Header.Set("User-Agent", h.userAgent)
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
	httpResp, err := h.client.Do(req)
	if err != nil {
		return nil, h.wrapNetworkError(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			h.logger.Error("Failed to close response body", "error", err)
		}
	}(httpResp.Body)

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	// Try to parse error response
	message := fmt.Sprintf("HTTP %d", resp.StatusCode)
	errorCode := resp.StatusCode

	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &apiErr); err == nil && apiErr.Error.Message != "" {
			message = apiErr.Error.Message
			if apiErr.Error.Code != 0 {
				errorCode = apiErr.Error.Code
			}
		}
	}

	details := string(resp.Body)
	if len(details) > 500 {
		details = details[:500] + "..."
	}

	// Create specific error types based on status code
	switch resp.StatusCode {
	case 401:
		return NewAuthenticationError(errorCode, message, details)
	case 403:
		return NewAuthenticationError(errorCode, message, details)
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
		if h.rateLimiter != nil {
			if resetTime.IsZero() {
				// If no reset time provided, estimate based on retry after
				resetTime = time.Now().Add(retryAfter)
			}
			h.rateLimiter.MarkRateLimited(resetTime)
		}

		return NewRateLimitError(errorCode, message, details, retryAfter)
	case 400, 422:
		return NewValidationError(errorCode, message, details, "")
	case 500, 502, 503, 504:
		return NewAPIError(errorCode, message, details, resp.RequestID)
	default:
		return NewAPIError(errorCode, message, details, resp.RequestID)
	}
}

// wrapNetworkError wraps network errors with appropriate error types
func (h *HTTPClient) wrapNetworkError(err error) error {
	// Check for timeout errors
	if timeoutErr, ok := err.(interface{ Timeout() bool }); ok && timeoutErr.Timeout() {
		return NewNetworkError(0, "Request timeout", err.Error(), true)
	}

	// Check for temporary errors
	if tempErr, ok := err.(interface{ Temporary() bool }); ok && tempErr.Temporary() {
		return NewNetworkError(0, "Temporary network error", err.Error(), true)
	}

	// Default to permanent network error
	return NewNetworkError(0, "Network error", err.Error(), false)
}

// isRetryableError determines if an error should trigger a retry
func (h *HTTPClient) isRetryableError(err error) bool {
	// Rate limit errors are retry-able
	if IsRateLimitError(err) {
		return true
	}

	// Temporary network errors are retry-able
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return netErr.Temporary
	}

	// Some API errors are retry-able (5xx status codes)
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code >= 500 && apiErr.Code < 600
	}

	return false
}

// shouldRetryStatus determines if a status code should trigger a retry
func (h *HTTPClient) shouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case 429: // Too Many Requests
		return true
	case 500, 502, 503, 504: // Server errors
		return true
	default:
		return false
	}
}

// logRequest logs the outgoing HTTP request
func (h *HTTPClient) logRequest(req *http.Request, body interface{}) {
	if h.logger == nil {
		return
	}

	fields := []interface{}{
		"method", req.Method,
		"url", req.URL.String(),
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

	h.logger.Debug("HTTP request", fields...)
}

// logResponse logs the HTTP response
func (h *HTTPClient) logResponse(resp *Response) {
	if h.logger == nil {
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
		h.logger.Error("HTTP response error", fields...)
	} else {
		h.logger.Debug("HTTP response", fields...)
	}
}

// logRetry logs retry attempts
func (h *HTTPClient) logRetry(attempt, maxRetries int, err error) {
	if h.logger == nil {
		return
	}

	h.logger.Warn("HTTP request retry",
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
