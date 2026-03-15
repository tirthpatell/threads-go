package threads

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewNetworkError(t *testing.T) {
	err := NewNetworkError(0, "connection refused", "details", true)
	if err == nil {
		t.Fatal("Expected non-nil error")
	}
	if err.Code != 0 {
		t.Errorf("Expected code 0, got %d", err.Code)
	}
	if err.Message != "connection refused" {
		t.Errorf("Expected message 'connection refused', got %q", err.Message)
	}
	if !err.Temporary {
		t.Error("Expected Temporary to be true")
	}
	if err.Cause != nil {
		t.Error("Expected Cause to be nil for NewNetworkError")
	}
}

func TestNewNetworkError_NonTemporary(t *testing.T) {
	err := NewNetworkError(0, "DNS failure", "could not resolve", false)
	if err.Temporary {
		t.Error("Expected Temporary to be false")
	}
}

func TestIsNetworkError(t *testing.T) {
	t.Run("returns true for NetworkError", func(t *testing.T) {
		err := NewNetworkError(0, "timeout", "details", true)
		if !IsNetworkError(err) {
			t.Error("Expected IsNetworkError to return true")
		}
	})

	t.Run("returns true for wrapped NetworkError", func(t *testing.T) {
		netErr := NewNetworkError(0, "timeout", "details", true)
		wrapped := fmt.Errorf("wrapper: %w", netErr)
		if !IsNetworkError(wrapped) {
			t.Error("Expected IsNetworkError to return true for wrapped NetworkError")
		}
	})

	t.Run("returns false for non-NetworkError", func(t *testing.T) {
		err := NewAPIError(500, "server error", "details", "req-123")
		if IsNetworkError(err) {
			t.Error("Expected IsNetworkError to return false for APIError")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		if IsNetworkError(nil) {
			t.Error("Expected IsNetworkError to return false for nil")
		}
	})
}

func TestNewRateLimitError(t *testing.T) {
	retryAfter := 30 * time.Second
	err := NewRateLimitError(429, "too many requests", "rate limited", retryAfter)

	if err == nil {
		t.Fatal("Expected non-nil error")
	}
	if err.Code != 429 {
		t.Errorf("Expected code 429, got %d", err.Code)
	}
	if err.Message != "too many requests" {
		t.Errorf("Expected message 'too many requests', got %q", err.Message)
	}
	if err.RetryAfter != retryAfter {
		t.Errorf("Expected RetryAfter %v, got %v", retryAfter, err.RetryAfter)
	}
	if err.Type != "rate_limit_error" {
		t.Errorf("Expected type 'rate_limit_error', got %q", err.Type)
	}
}

func TestBaseErrorError(t *testing.T) {
	t.Run("with details", func(t *testing.T) {
		err := &BaseError{
			Code:    400,
			Message: "Bad request",
			Type:    "validation_error",
			Details: "field X is invalid",
		}
		errStr := err.Error()
		if errStr == "" {
			t.Fatal("Expected non-empty error string")
		}
		// Should contain details
		if !strings.Contains(errStr, "field X is invalid") {
			t.Errorf("Expected error string to contain details, got: %s", errStr)
		}
	})

	t.Run("without details", func(t *testing.T) {
		err := &BaseError{
			Code:    400,
			Message: "Bad request",
			Type:    "validation_error",
		}
		errStr := err.Error()
		if errStr == "" {
			t.Fatal("Expected non-empty error string")
		}
		// Should not contain " - " separator since there are no details
		if strings.Contains(errStr, " - ") {
			t.Errorf("Expected error string without details separator, got: %s", errStr)
		}
	})
}

func TestExtractBaseError(t *testing.T) {
	t.Run("AuthenticationError", func(t *testing.T) {
		err := NewAuthenticationError(401, "unauthorized", "details")
		base := extractBaseError(err)
		if base == nil {
			t.Fatal("Expected non-nil BaseError")
		}
		if base.Code != 401 {
			t.Errorf("Expected code 401, got %d", base.Code)
		}
	})

	t.Run("RateLimitError", func(t *testing.T) {
		err := NewRateLimitError(429, "rate limited", "details", time.Minute)
		base := extractBaseError(err)
		if base == nil {
			t.Fatal("Expected non-nil BaseError")
		}
		if base.Code != 429 {
			t.Errorf("Expected code 429, got %d", base.Code)
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		err := NewValidationError(400, "invalid", "details", "field")
		base := extractBaseError(err)
		if base == nil {
			t.Fatal("Expected non-nil BaseError")
		}
		if base.Code != 400 {
			t.Errorf("Expected code 400, got %d", base.Code)
		}
	})

	t.Run("NetworkError", func(t *testing.T) {
		err := NewNetworkError(0, "timeout", "details", true)
		base := extractBaseError(err)
		if base == nil {
			t.Fatal("Expected non-nil BaseError")
		}
		if base.Type != "network_error" {
			t.Errorf("Expected type 'network_error', got %q", base.Type)
		}
	})

	t.Run("APIError", func(t *testing.T) {
		err := NewAPIError(500, "server error", "details", "req-123")
		base := extractBaseError(err)
		if base == nil {
			t.Fatal("Expected non-nil BaseError")
		}
		if base.Code != 500 {
			t.Errorf("Expected code 500, got %d", base.Code)
		}
	})

	t.Run("unknown error type returns nil", func(t *testing.T) {
		err := errors.New("plain error")
		base := extractBaseError(err)
		if base != nil {
			t.Errorf("Expected nil for unknown error type, got %v", base)
		}
	})
}
