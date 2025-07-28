package threads

import (
	"errors"
	"fmt"
	"time"
)

// BaseError represents a base error type for all Threads API errors.
// For error handling patterns, see: https://developers.facebook.com/docs/threads/troubleshooting
type BaseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *BaseError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("threads api error %d (%s): %s - %s", e.Code, e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("threads api error %d (%s): %s", e.Code, e.Type, e.Message)
}

// AuthenticationError represents authentication-related errors such as
// invalid tokens, expired tokens, or missing authentication credentials.
// Common HTTP status codes: 401 (Unauthorized), 403 (Forbidden).
type AuthenticationError struct {
	*BaseError
}

// NewAuthenticationError creates a new authentication error with the specified details.
// Use this when authentication fails, tokens are invalid/expired, or credentials are missing.
// The code parameter should typically be 401 or 403.
func NewAuthenticationError(code int, message, details string) *AuthenticationError {
	return &AuthenticationError{
		BaseError: &BaseError{
			Code:    code,
			Message: message,
			Type:    "authentication_error",
			Details: details,
		},
	}
}

// RateLimitError represents rate limiting errors when API quota is exceeded.
// Contains RetryAfter duration indicating when the client can retry the request.
// Common HTTP status code: 429 (Too Many Requests).
type RateLimitError struct {
	*BaseError
	RetryAfter time.Duration `json:"retry_after"`
}

// NewRateLimitError creates a new rate limit error with retry information.
// The retryAfter parameter indicates how long to wait before retrying.
// The code parameter should typically be 429.
func NewRateLimitError(code int, message, details string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		BaseError: &BaseError{
			Code:    code,
			Message: message,
			Type:    "rate_limit_error",
			Details: details,
		},
		RetryAfter: retryAfter,
	}
}

// ValidationError represents validation-related errors such as invalid parameters,
// missing required fields, or data that doesn't meet API requirements.
// The Field property indicates which field failed validation.
// Common HTTP status code: 400 (Bad Request).
type ValidationError struct {
	*BaseError
	Field string `json:"field,omitempty"`
}

// NewValidationError creates a new validation error with field information.
// The field parameter should indicate which input field failed validation.
// The code parameter should typically be 400.
func NewValidationError(code int, message, details, field string) *ValidationError {
	return &ValidationError{
		BaseError: &BaseError{
			Code:    code,
			Message: message,
			Type:    "validation_error",
			Details: details,
		},
		Field: field,
	}
}

// NetworkError represents network-related errors such as connection failures,
// timeouts, or DNS resolution issues. The Temporary field indicates if the
// error is likely transient and the request can be retried.
type NetworkError struct {
	*BaseError
	Temporary bool `json:"temporary"`
}

// NewNetworkError creates a new network error with temporary status.
// Set temporary to true for transient errors that may succeed on retry.
// The code may be 0 for client-side network failures.
func NewNetworkError(code int, message, details string, temporary bool) *NetworkError {
	return &NetworkError{
		BaseError: &BaseError{
			Code:    code,
			Message: message,
			Type:    "network_error",
			Details: details,
		},
		Temporary: temporary,
	}
}

// APIError represents general server-side API errors not covered by other error types.
// The RequestID can be used for debugging with Threads API support.
// Common HTTP status codes: 500 (Internal Server Error), 503 (Service Unavailable).
type APIError struct {
	*BaseError
	RequestID string `json:"request_id,omitempty"`
}

// NewAPIError creates a new API error with optional request ID.
// The requestID parameter helps with debugging and should be included when available
// from API response headers.
func NewAPIError(code int, message, details, requestID string) *APIError {
	return &APIError{
		BaseError: &BaseError{
			Code:    code,
			Message: message,
			Type:    "api_error",
			Details: details,
		},
		RequestID: requestID,
	}
}

// IsAuthenticationError checks if an error is an authentication error.
// This is useful for implementing retry logic or handling authentication failures.
// Returns true if the error is of type *AuthenticationError.
func IsAuthenticationError(err error) bool {
	var authenticationError *AuthenticationError
	ok := errors.As(err, &authenticationError)
	return ok
}

// IsRateLimitError checks if an error is a rate limit error.
// Use this to implement backoff strategies when rate limited.
// Returns true if the error is of type *RateLimitError.
func IsRateLimitError(err error) bool {
	var rateLimitError *RateLimitError
	ok := errors.As(err, &rateLimitError)
	return ok
}

// IsValidationError checks if an error is a validation error.
// This helps identify client-side input errors that need correction.
// Returns true if the error is of type *ValidationError.
func IsValidationError(err error) bool {
	var validationError *ValidationError
	ok := errors.As(err, &validationError)
	return ok
}

// IsNetworkError checks if an error is a network error.
// Network errors may be temporary and can often be retried.
// Returns true if the error is of type *NetworkError.
func IsNetworkError(err error) bool {
	var networkError *NetworkError
	ok := errors.As(err, &networkError)
	return ok
}

// IsAPIError checks if an error is a general API error.
// These are typically server-side issues that may require support.
// Returns true if the error is of type *APIError.
func IsAPIError(err error) bool {
	var APIError *APIError
	ok := errors.As(err, &APIError)
	return ok
}
