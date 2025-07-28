package threads

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TokenResponse represents the response from token exchange endpoint.
// This structure is returned when exchanging an authorization code for an access token.
// The access token can then be used to authenticate API requests.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

// LongLivedTokenResponse represents the response from long-lived token conversion endpoint.
// Long-lived tokens typically last for 60 days compared to short-lived tokens which last 1 hour.
// Convert short-lived tokens to long-lived tokens for better user experience.
type LongLivedTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// generateState generates a random state parameter for OAuth security
func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthURL generates the authorization URL for OAuth 2.0 flow.
// Users should be redirected to this URL to grant permissions to your app.
// If scopes are not provided, defaults to threads_basic and threads_content_publish.
// Returns the complete authorization URL including all necessary parameters.
func (c *Client) GetAuthURL(scopes []string) string {
	if len(scopes) == 0 {
		scopes = []string{"threads_basic", "threads_content_publish"}
	}

	state, err := generateState()
	if err != nil {
		// If we can't generate state, use a simple timestamp-based fallback
		state = fmt.Sprintf("state_%d", time.Now().Unix())
	}

	params := url.Values{
		"client_id":     {c.config.ClientID},
		"redirect_uri":  {c.config.RedirectURI},
		"scope":         {strings.Join(scopes, " ")}, // Use space-separated scopes
		"response_type": {"code"},
		"state":         {state},
	}

	authURL := fmt.Sprintf("https://www.threads.net/oauth/authorize?%s", params.Encode())
	return authURL
}

// ExchangeCodeForToken exchanges an authorization code for an access token.
// This should be called after the user authorizes your app, and you receive the code
// from the redirect URI callback. The resulting token is automatically stored
// in the client and token storage.
func (c *Client) ExchangeCodeForToken(ctx context.Context, code string) error {
	if code == "" {
		return NewValidationError(400, "Authorization code is required", "Code parameter cannot be empty", "code")
	}

	data := url.Values{
		"client_id":     {c.config.ClientID},
		"client_secret": {c.config.ClientSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {c.config.RedirectURI},
		"code":          {code},
	}

	resp, err := c.httpClient.POST("/oauth/access_token", data, "")
	if err != nil {
		return NewNetworkError(0, "Failed to exchange code for token", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		return c.handleTokenError(resp.StatusCode, resp.Body)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(resp.Body, &tokenResp); err != nil {
		return NewAPIError(resp.StatusCode, "Failed to parse token response", err.Error(), "")
	}

	// Create token info with expiration
	now := time.Now()
	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	} else {
		// Fallback if API doesn't provide expires_in (shouldn't happen but just in case)
		expiresAt = now.Add(time.Hour) // Short-lived tokens typically expire in 1 hour
	}

	tokenInfo := &TokenInfo{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresAt:   expiresAt,
		UserID:      tokenResp.UserID,
		CreatedAt:   now,
	}

	// Store the token using thread-safe method
	if err := c.SetTokenInfo(tokenInfo); err != nil {
		if c.config.Logger != nil {
			c.config.Logger.Warn("Failed to store token", "error", err.Error())
		}
	}

	// Log successful authentication if logger is available
	if c.config.Logger != nil {
		c.config.Logger.Info("Successfully exchanged authorization code for access token",
			"user_id", tokenResp.UserID,
			"token_type", tokenResp.TokenType,
			"expires_at", expiresAt)
	}

	return nil
}

// GetLongLivedToken converts a short-lived token to a long-lived token.
// Short-lived tokens expire in 1 hour while long-lived tokens last for 60 days.
// This method requires an existing valid short-lived token in the client.
// The long-lived token automatically replaces the short-lived token in storage.
func (c *Client) GetLongLivedToken(ctx context.Context) error {
	c.mu.RLock()
	currentToken := c.accessToken
	c.mu.RUnlock()

	if currentToken == "" {
		return NewAuthenticationError(401, "No access token available", "Must exchange authorization code for token first")
	}

	params := url.Values{
		"grant_type":    {"th_exchange_token"},
		"client_secret": {c.config.ClientSecret},
		"access_token":  {currentToken},
	}

	resp, err := c.httpClient.GET("/access_token", params, currentToken)
	if err != nil {
		return NewNetworkError(0, "Failed to get long-lived token", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		return c.handleTokenError(resp.StatusCode, resp.Body)
	}

	var tokenResp LongLivedTokenResponse
	if err := json.Unmarshal(resp.Body, &tokenResp); err != nil {
		return NewAPIError(resp.StatusCode, "Failed to parse long-lived token response", err.Error(), "")
	}

	// Update token info with long-lived token
	now := time.Now()
	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	} else {
		// Fallback if API doesn't provide expires_in (shouldn't happen for long-lived tokens)
		expiresAt = now.Add(60 * 24 * time.Hour) // 60 days default
	}

	// Create new token info with long-lived token
	c.mu.RLock()
	var userID string
	if c.tokenInfo != nil {
		userID = c.tokenInfo.UserID
	}
	c.mu.RUnlock()

	tokenInfo := &TokenInfo{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresAt:   expiresAt,
		UserID:      userID,
		CreatedAt:   now,
	}

	// Store the token using thread-safe method
	if err := c.SetTokenInfo(tokenInfo); err != nil {
		if c.config.Logger != nil {
			c.config.Logger.Warn("Failed to store long-lived token", "error", err.Error())
		}
	}

	// Log successful long-lived token conversion if logger is available
	if c.config.Logger != nil {
		c.config.Logger.Info("Successfully converted to long-lived token",
			"expires_in_seconds", tokenResp.ExpiresIn,
			"expires_at", expiresAt,
			"token_type", tokenResp.TokenType)
	}

	return nil
}

// RefreshToken refreshes the current access token before it expires.
// This extends the validity of your existing token without requiring user re-authorization.
// The refreshed token automatically replaces the current token in storage.
// Note: Only long-lived tokens can be refreshed.
func (c *Client) RefreshToken(ctx context.Context) error {
	c.mu.RLock()
	currentToken := c.accessToken
	c.mu.RUnlock()

	if currentToken == "" {
		return NewAuthenticationError(401, "No access token to refresh", "Must have an existing token to refresh")
	}

	params := url.Values{
		"grant_type":   {"th_refresh_token"},
		"access_token": {currentToken},
	}

	resp, err := c.httpClient.GET("/refresh_access_token", params, "")
	if err != nil {
		return NewNetworkError(0, "Failed to refresh token", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		return c.handleTokenError(resp.StatusCode, resp.Body)
	}

	var tokenResp LongLivedTokenResponse
	if err := json.Unmarshal(resp.Body, &tokenResp); err != nil {
		return NewAPIError(resp.StatusCode, "Failed to parse token refresh response", err.Error(), "")
	}

	// Update token info with refreshed token
	now := time.Now()
	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	} else {
		// Fallback if API doesn't provide expires_in (shouldn't happen for refreshed tokens)
		expiresAt = now.Add(60 * 24 * time.Hour) // 60 days default
	}

	// Create new token info with refreshed token
	c.mu.RLock()
	var userID string
	if c.tokenInfo != nil {
		userID = c.tokenInfo.UserID
	}
	c.mu.RUnlock()

	tokenInfo := &TokenInfo{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresAt:   expiresAt,
		UserID:      userID,
		CreatedAt:   now,
	}

	// Store the token using thread-safe method
	if err := c.SetTokenInfo(tokenInfo); err != nil {
		if c.config.Logger != nil {
			c.config.Logger.Warn("Failed to store refreshed token", "error", err.Error())
		}
	}

	// Log successful token refresh if logger is available
	if c.config.Logger != nil {
		c.config.Logger.Info("Successfully refreshed access token",
			"expires_in_seconds", tokenResp.ExpiresIn,
			"expires_at", expiresAt,
			"token_type", tokenResp.TokenType)
	}

	return nil
}

// handleTokenError processes token-related API errors
func (c *Client) handleTokenError(statusCode int, body []byte) error {
	var errorResp struct {
		Error struct {
			Message   string `json:"message"`
			Type      string `json:"type"`
			Code      int    `json:"code"`
			ErrorData struct {
				Details string `json:"details"`
			} `json:"error_data"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		// If we can't parse the error response, return a generic error
		return NewAuthenticationError(statusCode, "Authentication failed", string(body))
	}

	message := errorResp.Error.Message
	details := errorResp.Error.ErrorData.Details
	errorType := errorResp.Error.Type

	// Map common error types to appropriate error categories
	switch {
	case statusCode == 401 || strings.Contains(errorType, "auth") || strings.Contains(message, "token"):
		return NewAuthenticationError(statusCode, message, details)
	case statusCode == 429 || strings.Contains(errorType, "rate"):
		// Try to extract retry-after from headers if available
		return NewRateLimitError(statusCode, message, details, 60*time.Second)
	case statusCode >= 400 && statusCode < 500:
		return NewValidationError(statusCode, message, details, "")
	default:
		return NewAPIError(statusCode, message, details, "")
	}
}

// GetAccessToken returns the current access token in a thread-safe manner.
// This method is primarily intended for debugging and testing purposes.
// For production use, the client handles token management automatically.
func (c *Client) GetAccessToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// getAccessTokenSafe returns the current access token in a thread-safe manner
// This is an internal method for use by other client methods
func (c *Client) getAccessTokenSafe() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.accessToken
}

// LoadTokenFromStorage attempts to load a previously stored token from the configured storage.
// If a valid token is found, it's automatically set as the active token for the client.
// Returns an error if no token is found, if the token is expired, or if loading fails.
func (c *Client) LoadTokenFromStorage() error {
	tokenInfo, err := c.tokenStorage.Load()
	if err != nil {
		return err
	}

	// Check if loaded token is expired
	if time.Now().After(tokenInfo.ExpiresAt) {
		// Token is expired, clear it
		err := c.tokenStorage.Delete()
		if err != nil {
			return err
		}
		return NewAuthenticationError(401, "Stored token expired", "Token found in storage but expired")
	}

	// Use thread-safe method to set token
	if err := c.SetTokenInfo(tokenInfo); err != nil {
		return err
	}

	if c.config.Logger != nil {
		c.config.Logger.Info("Successfully loaded token from storage",
			"expires_at", tokenInfo.ExpiresAt,
			"user_id", tokenInfo.UserID)
	}

	return nil
}

// GetTokenDebugInfo returns detailed token information for debugging purposes.
// This method provides comprehensive information about the current token state including
// expiration times, validity checks, and calculated values useful for troubleshooting.
// The returned map contains various fields like has_token, is_authenticated, expires_at, etc.
func (c *Client) GetTokenDebugInfo() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	debugInfo := map[string]interface{}{
		"has_token":        c.accessToken != "",
		"is_authenticated": c.IsAuthenticated(),
		"is_expired":       c.IsTokenExpired(),
	}

	if c.tokenInfo != nil {
		now := time.Now()
		timeUntilExpiry := c.tokenInfo.ExpiresAt.Sub(now)

		debugInfo["token_type"] = c.tokenInfo.TokenType
		debugInfo["user_id"] = c.tokenInfo.UserID
		debugInfo["created_at"] = c.tokenInfo.CreatedAt
		debugInfo["expires_at"] = c.tokenInfo.ExpiresAt
		debugInfo["time_until_expiry"] = timeUntilExpiry.String()
		debugInfo["expires_in_hours"] = timeUntilExpiry.Hours()
		debugInfo["expires_in_days"] = timeUntilExpiry.Hours() / 24
		debugInfo["expiring_soon_1h"] = c.IsTokenExpiringSoon(time.Hour)
		debugInfo["expiring_soon_24h"] = c.IsTokenExpiringSoon(24 * time.Hour)
		debugInfo["expiring_soon_7d"] = c.IsTokenExpiringSoon(7 * 24 * time.Hour)
	}

	return debugInfo
}

// DebugTokenResponse represents the response from the debug_token endpoint
type DebugTokenResponse struct {
	Data struct {
		Type                string   `json:"type"`
		Application         string   `json:"application"`
		DataAccessExpiresAt int64    `json:"data_access_expires_at"`
		ExpiresAt           int64    `json:"expires_at"`
		IsValid             bool     `json:"is_valid"`
		IssuedAt            int64    `json:"issued_at"`
		Scopes              []string `json:"scopes"`
		UserID              string   `json:"user_id"`
	} `json:"data"`
}

// DebugToken calls the debug_token endpoint to get detailed token information.
// If inputToken is empty, it will debug the client's current access token.
// This method is useful for validating token status, checking expiration times,
// and retrieving token metadata like scopes and user information.
func (c *Client) DebugToken(ctx context.Context, inputToken string) (*DebugTokenResponse, error) {
	c.mu.RLock()
	accessToken := c.accessToken
	c.mu.RUnlock()

	if accessToken == "" {
		return nil, NewAuthenticationError(401, "No access token available", "Client must be authenticated to debug tokens")
	}

	if inputToken == "" {
		inputToken = accessToken
	}

	params := url.Values{
		"input_token":  {inputToken},
		"access_token": {accessToken},
	}

	resp, err := c.httpClient.GET("/debug_token", params, accessToken)
	if err != nil {
		return nil, NewNetworkError(0, "Failed to debug token", err.Error(), true)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleTokenError(resp.StatusCode, resp.Body)
	}

	var debugResp DebugTokenResponse
	if err := json.Unmarshal(resp.Body, &debugResp); err != nil {
		return nil, NewAPIError(resp.StatusCode, "Failed to parse debug token response", err.Error(), "")
	}

	if c.config.Logger != nil {
		c.config.Logger.Debug("Debug token response received",
			"is_valid", debugResp.Data.IsValid,
			"expires_at", debugResp.Data.ExpiresAt,
			"issued_at", debugResp.Data.IssuedAt,
			"user_id", debugResp.Data.UserID,
			"scopes", debugResp.Data.Scopes)
	}

	return &debugResp, nil
}

// SetTokenFromDebugInfo creates and sets token info from debug token response.
// This method takes the response from the debug_token endpoint and creates a properly
// configured TokenInfo struct with accurate expiration times based on the API response.
// This is useful for setting up tokens when you have the debug information available.
func (c *Client) SetTokenFromDebugInfo(accessToken string, debugResp *DebugTokenResponse) error {
	if debugResp == nil {
		return fmt.Errorf("debug response cannot be nil")
	}

	if !debugResp.Data.IsValid {
		return NewAuthenticationError(401, "Token is not valid", "Debug token endpoint reports token as invalid")
	}

	// Convert Unix timestamps to time.Time
	expiresAt := time.Unix(debugResp.Data.ExpiresAt, 0)
	issuedAt := time.Unix(debugResp.Data.IssuedAt, 0)

	tokenInfo := &TokenInfo{
		AccessToken: accessToken,
		TokenType:   "Bearer", // Threads API uses Bearer tokens
		ExpiresAt:   expiresAt,
		UserID:      debugResp.Data.UserID,
		CreatedAt:   issuedAt, // Use the issued_at from the API
	}

	// Store the token using thread-safe method
	if err := c.SetTokenInfo(tokenInfo); err != nil {
		return fmt.Errorf("failed to store token info: %w", err)
	}

	if c.config.Logger != nil {
		lifetime := expiresAt.Sub(issuedAt)
		c.config.Logger.Info("Token info set from debug response",
			"user_id", debugResp.Data.UserID,
			"expires_at", expiresAt,
			"issued_at", issuedAt,
			"lifetime_hours", lifetime.Hours(),
			"lifetime_days", lifetime.Hours()/24,
			"scopes", debugResp.Data.Scopes)
	}

	return nil
}
