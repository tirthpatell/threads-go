package threads

import (
	"encoding/json"
	"fmt"
)

// getUserID extracts user ID from token info
func (c *Client) getUserID() string {
	if c.tokenInfo != nil && c.tokenInfo.UserID != "" {
		return c.tokenInfo.UserID
	}

	// If user ID is not in token info, we might need to call /me endpoint
	// For now, return empty string to trigger an error
	return ""
}

// handleAPIError processes API error responses
func (c *Client) handleAPIError(resp *Response) error {
	var apiErr struct {
		Error struct {
			Message     string `json:"message"`
			Type        string `json:"type"`
			Code        int    `json:"code"`
			IsTransient bool   `json:"is_transient"`
			ErrorData   struct {
				Details string `json:"details"`
			} `json:"error_data"`
		} `json:"error"`
	}

	// Try to parse structured error response
	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &apiErr); err == nil && apiErr.Error.Message != "" {
			message := apiErr.Error.Message
			details := apiErr.Error.ErrorData.Details
			errorCode := apiErr.Error.Code
			isTransient := apiErr.Error.IsTransient
			if errorCode == 0 {
				errorCode = resp.StatusCode
			}

			// Return appropriate error type based on status code
			var resultErr error
			switch resp.StatusCode {
			case 401, 403:
				resultErr = NewAuthenticationError(errorCode, message, details)
			case 429:
				retryAfter := resp.RateLimit.RetryAfter
				resultErr = NewRateLimitError(errorCode, message, details, retryAfter)
			case 400, 422:
				resultErr = NewValidationError(errorCode, message, details, "")
			default:
				resultErr = NewAPIError(errorCode, message, details, resp.RequestID)
			}

			// Set IsTransient and HTTPStatusCode on the base error
			if base := extractBaseError(resultErr); base != nil {
				base.IsTransient = isTransient
				base.HTTPStatusCode = resp.StatusCode
			}

			return resultErr
		}
	}

	// Fallback to generic error
	message := fmt.Sprintf("API request failed with status %d", resp.StatusCode)
	details := string(resp.Body)
	if len(details) > 500 {
		details = details[:500] + "..."
	}

	fallbackErr := NewAPIError(resp.StatusCode, message, details, resp.RequestID)
	fallbackErr.HTTPStatusCode = resp.StatusCode
	return fallbackErr
}
