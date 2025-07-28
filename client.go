// Package threads provides a comprehensive Go client for the Threads API by Meta.
//
// The Threads API allows developers to build applications that interact with
// Threads, Meta's social media platform. This package provides a clean, idiomatic
// Go interface for all Threads API endpoints including authentication, posts,
// users, replies, insights, and more.
//
// # Quick Start
//
// For users with an existing access token:
//
//	client, err := threads.NewClientWithToken("your-access-token", &threads.Config{
//		ClientID:     "your-client-id",
//		ClientSecret: "your-client-secret",
//		RedirectURI:  "your-redirect-uri",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create a post
//	post, err := client.CreateTextPost(ctx, &threads.TextPostContent{
//		Text: "Hello from Go!",
//	})
//
// For OAuth 2.0 authentication flow:
//
//	config := &threads.Config{
//		ClientID:     "your-client-id",
//		ClientSecret: "your-client-secret",
//		RedirectURI:  "your-redirect-uri",
//		Scopes:       []string{"threads_basic", "threads_content_publish"},
//	}
//
//	client, err := threads.NewClient(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Get authorization URL
//	authURL := client.GetAuthURL(config.Scopes)
//	// Direct user to authURL, then exchange code for token
//	err = client.ExchangeCodeForToken("auth-code-from-callback")
//
// For complete API documentation: https://developers.facebook.com/docs/threads
package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client provides access to the Threads API with thread-safe operations.
// It implements the ClientInterface and all its composed interfaces.
type Client struct {
	config       *Config
	httpClient   *HTTPClient
	rateLimiter  *RateLimiter
	baseURL      string
	accessToken  string
	tokenInfo    *TokenInfo
	tokenStorage TokenStorage
	mu           sync.RWMutex // Protects token-related fields
}

// Config holds configuration settings for the Threads API client.
// Required fields: ClientID, ClientSecret, RedirectURI.
// All other fields have sensible defaults.
type Config struct {
	// ClientID is your Threads app's client ID (required).
	// This is provided when you create a Threads app in the Meta Developer Console.
	ClientID string

	// ClientSecret is your Threads app's client secret (required).
	// This is provided when you create a Threads app in the Meta Developer Console.
	// Keep this secret and never expose it in client-side code.
	ClientSecret string

	// RedirectURI is the URI where users will be redirected after authorization (required).
	// This must match exactly with the redirect URI configured in your app settings.
	// Must be a valid HTTP or HTTPS URL.
	RedirectURI string

	// Scopes defines the permissions your app is requesting (required).
	// Available scopes include:
	// - threads_basic: Basic profile access
	// - threads_content_publish: Create and publish posts
	// - threads_manage_insights: Access analytics data
	// - threads_manage_replies: Manage replies and conversations
	// - threads_read_replies: Read replies to posts
	// - threads_manage_mentions: Manage mentions
	// - threads_keyword_search: Search functionality
	// - threads_delete: Delete posts
	// - threads_location_tagging: Location services
	// - threads_profile_discovery: Public profile lookup
	Scopes []string

	// HTTPTimeout sets the timeout for HTTP requests (optional).
	// Default: 30 seconds. Set to 0 for no timeout (not recommended).
	HTTPTimeout time.Duration

	// RetryConfig configures retry behavior for failed requests (optional).
	// If nil, default retry configuration will be used.
	RetryConfig *RetryConfig

	// Logger provides structured logging for debugging and monitoring (optional).
	// If nil, no logging will be performed. Implement the Logger interface
	// to provide custom logging behavior.
	Logger Logger

	// TokenStorage provides persistent token storage (optional).
	// If nil, tokens will be stored in memory only and lost when the client
	// is destroyed. Implement the TokenStorage interface for persistence.
	TokenStorage TokenStorage

	// BaseURL is the base URL for the Threads API (optional).
	// Default: "https://graph.threads.net". Only change this for testing
	// or if using a proxy/gateway.
	BaseURL string

	// UserAgent is the User-Agent header sent with requests (optional).
	// Default: "threads-go/1.0.0". Customize this to identify your application.
	UserAgent string

	// Debug enables debug mode with verbose logging (optional).
	// Default: false. When true, detailed request/response information
	// will be logged if a Logger is provided.
	Debug bool
}

// RetryConfig defines retry behavior for failed requests with exponential backoff.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3).
	// Set to 0 to disable retries. Higher values provide more resilience
	// but may increase latency for failing requests.
	MaxRetries int

	// InitialDelay is the delay before the first retry attempt (default: 1 second).
	// This delay is multiplied by BackoffFactor for subsequent retries.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retry attempts (default: 30 seconds).
	// This prevents exponential backoff from creating excessively long delays.
	MaxDelay time.Duration

	// BackoffFactor is the multiplier for exponential backoff (default: 2.0).
	// Each retry delay is calculated as: min(InitialDelay * BackoffFactor^attempt, MaxDelay)
	BackoffFactor float64
}

// Logger interface for structured logging.
type Logger interface {
	// Debug logs debug-level messages with optional structured fields.
	// Used for detailed tracing and development debugging.
	Debug(msg string, fields ...any)

	// Info logs informational messages with optional structured fields.
	// Used for general operational information.
	Info(msg string, fields ...any)

	// Warn logs warning messages with optional structured fields.
	// Used for potentially problematic situations that don't prevent operation.
	Warn(msg string, fields ...any)

	// Error logs error messages with optional structured fields.
	// Used for error conditions that may affect functionality.
	Error(msg string, fields ...any)
}

// TokenStorage interface for storing and retrieving tokens.
// The default MemoryTokenStorage loses tokens when the application terminates.
type TokenStorage interface {
	// Store saves a token to persistent storage.
	// Should return an error if the token cannot be saved.
	Store(token *TokenInfo) error

	// Load retrieves a token from persistent storage.
	// Should return an error if no token is found or cannot be loaded.
	Load() (*TokenInfo, error)

	// Delete removes a token from persistent storage.
	// Should return an error if the token cannot be deleted.
	Delete() error
}

// TokenInfo holds information about the current token
type TokenInfo struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// MemoryTokenStorage provides in-memory token storage (default)
type MemoryTokenStorage struct {
	token *TokenInfo
}

// Store saves the token in memory
func (m *MemoryTokenStorage) Store(token *TokenInfo) error {
	m.token = token
	return nil
}

// Load retrieves the token from memory
func (m *MemoryTokenStorage) Load() (*TokenInfo, error) {
	if m.token == nil {
		return nil, NewAuthenticationError(401, "No token stored", "Token not found in memory storage")
	}
	return m.token, nil
}

// Delete removes the token from memory
func (m *MemoryTokenStorage) Delete() error {
	m.token = nil
	return nil
}

// NewConfig creates a new configuration with sensible defaults.
func NewConfig() *Config {
	return &Config{
		Scopes: []string{"threads_basic",
			"threads_content_publish",
			"threads_manage_replies",
			"threads_manage_insights",
			"threads_read_replies",
			"threads_manage_mentions",
			"threads_keyword_search",
			"threads_delete",
			"threads_location_tagging",
			"threads_profile_discovery"},
		HTTPTimeout: 30 * time.Second,
		RetryConfig: &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
		BaseURL:   "https://graph.threads.net",
		UserAgent: "threads-go/1.0.0",
		Debug:     false,
	}
}

// NewConfigFromEnv creates a new configuration from environment variables.
// Required: THREADS_CLIENT_ID, THREADS_CLIENT_SECRET, THREADS_REDIRECT_URI.
func NewConfigFromEnv() (*Config, error) {
	config := NewConfig()

	// Required environment variables
	clientID := os.Getenv("THREADS_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("THREADS_CLIENT_ID environment variable is required")
	}
	config.ClientID = clientID

	clientSecret := os.Getenv("THREADS_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("THREADS_CLIENT_SECRET environment variable is required")
	}
	config.ClientSecret = clientSecret

	redirectURI := os.Getenv("THREADS_REDIRECT_URI")
	if redirectURI == "" {
		return nil, fmt.Errorf("THREADS_REDIRECT_URI environment variable is required")
	}
	config.RedirectURI = redirectURI

	// Optional environment variables
	if scopes := os.Getenv("THREADS_SCOPES"); scopes != "" {
		config.Scopes = strings.Split(scopes, ",")
		// Trim whitespace from each scope
		for i, scope := range config.Scopes {
			config.Scopes[i] = strings.TrimSpace(scope)
		}
	}

	if timeout := os.Getenv("THREADS_HTTP_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.HTTPTimeout = duration
		}
	}

	if baseURL := os.Getenv("THREADS_BASE_URL"); baseURL != "" {
		config.BaseURL = baseURL
	}

	if userAgent := os.Getenv("THREADS_USER_AGENT"); userAgent != "" {
		config.UserAgent = userAgent
	}

	if debug := os.Getenv("THREADS_DEBUG"); debug != "" {
		if debugBool, err := strconv.ParseBool(debug); err == nil {
			config.Debug = debugBool
		}
	}

	// Retry configuration from environment
	if maxRetries := os.Getenv("THREADS_MAX_RETRIES"); maxRetries != "" {
		if retries, err := strconv.Atoi(maxRetries); err == nil && retries >= 0 {
			config.RetryConfig.MaxRetries = retries
		}
	}

	if initialDelay := os.Getenv("THREADS_INITIAL_DELAY"); initialDelay != "" {
		if delay, err := time.ParseDuration(initialDelay); err == nil {
			config.RetryConfig.InitialDelay = delay
		}
	}

	if maxDelay := os.Getenv("THREADS_MAX_DELAY"); maxDelay != "" {
		if delay, err := time.ParseDuration(maxDelay); err == nil {
			config.RetryConfig.MaxDelay = delay
		}
	}

	if backoffFactor := os.Getenv("THREADS_BACKOFF_FACTOR"); backoffFactor != "" {
		if factor, err := strconv.ParseFloat(backoffFactor, 64); err == nil && factor > 0 {
			config.RetryConfig.BackoffFactor = factor
		}
	}

	return config, nil
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("ClientID is required")
	}

	if c.ClientSecret == "" {
		return fmt.Errorf("ClientSecret is required")
	}

	if c.RedirectURI == "" {
		return fmt.Errorf("RedirectURI is required")
	}

	// Validate redirect URI format
	if !strings.HasPrefix(c.RedirectURI, "http://") && !strings.HasPrefix(c.RedirectURI, "https://") {
		return fmt.Errorf("RedirectURI must be a valid HTTP or HTTPS URL")
	}

	if len(c.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}

	// Validate scopes
	validScopes := map[string]bool{
		"threads_basic":             true,
		"threads_content_publish":   true,
		"threads_manage_insights":   true,
		"threads_manage_replies":    true,
		"threads_read_replies":      true,
		"threads_manage_mentions":   true,
		"threads_keyword_search":    true,
		"threads_delete":            true,
		"threads_location_tagging":  true,
		"threads_profile_discovery": true,
	}

	for _, scope := range c.Scopes {
		if !validScopes[scope] {
			return fmt.Errorf("invalid scope: %s", scope)
		}
	}

	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("HTTPTimeout must be positive")
	}

	if c.RetryConfig != nil {
		if c.RetryConfig.MaxRetries < 0 {
			return fmt.Errorf("RetryConfig.MaxRetries must be non-negative")
		}

		if c.RetryConfig.InitialDelay <= 0 {
			return fmt.Errorf("RetryConfig.InitialDelay must be positive")
		}

		if c.RetryConfig.MaxDelay <= 0 {
			return fmt.Errorf("RetryConfig.MaxDelay must be positive")
		}

		if c.RetryConfig.BackoffFactor <= 0 {
			return fmt.Errorf("RetryConfig.BackoffFactor must be positive")
		}

		if c.RetryConfig.InitialDelay > c.RetryConfig.MaxDelay {
			return fmt.Errorf("RetryConfig.InitialDelay cannot be greater than MaxDelay")
		}
	}

	if c.BaseURL == "" {
		return fmt.Errorf("BaseURL is required")
	}

	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("BaseURL must be a valid HTTP or HTTPS URL")
	}

	return nil
}

// SetDefaults sets default values for any unset configuration options
func (c *Config) SetDefaults() {
	if len(c.Scopes) == 0 {
		c.Scopes = []string{"threads_basic", "threads_content_publish", "threads_manage_insights", "threads_manage_replies", "threads_read_replies"}
	}

	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = 30 * time.Second
	}

	if c.RetryConfig == nil {
		c.RetryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		}
	}

	if c.BaseURL == "" {
		c.BaseURL = "https://graph.threads.net"
	}

	if c.UserAgent == "" {
		c.UserAgent = "threads-go/1.0.0"
	}
}

// NewClient creates a new Threads API client with the provided configuration.
// The client is thread-safe and can be used concurrently from multiple goroutines.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Set defaults for any missing configuration
	config.SetDefaults()

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Use memory storage as default if none provided
	tokenStorage := config.TokenStorage
	if tokenStorage == nil {
		tokenStorage = &MemoryTokenStorage{}
	}

	// Create rate limiter
	rateLimiterConfig := &RateLimiterConfig{
		InitialLimit:      100, // Default limit, will be updated from API responses
		BackoffMultiplier: 2.0,
		MaxBackoff:        5 * time.Minute,
		QueueSize:         100,
		Logger:            config.Logger,
	}
	rateLimiter := NewRateLimiter(rateLimiterConfig)

	// Create HTTP client
	httpClient := NewHTTPClient(config, rateLimiter)

	client := &Client{
		config:       config,
		httpClient:   httpClient,
		rateLimiter:  rateLimiter,
		baseURL:      config.BaseURL,
		tokenStorage: tokenStorage,
	}

	// Try to load existing token from storage
	if tokenInfo, err := tokenStorage.Load(); err == nil {
		client.tokenInfo = tokenInfo
		client.accessToken = tokenInfo.AccessToken
	}

	return client, nil
}

// NewClientFromEnv creates a new Threads API client using environment variables.
// This is a convenience function that combines NewConfigFromEnv and NewClient.
func NewClientFromEnv() (*Client, error) {
	config, err := NewConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create config from environment: %w", err)
	}

	return NewClient(config)
}

// NewClientWithToken creates a new Threads API client with an existing access token.
// The function validates the token by calling the debug_token endpoint.
func NewClientWithToken(accessToken string, config *Config) (*Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token cannot be empty")
	}

	// Create the client first
	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Set a temporary token to enable the debug call
	tempTokenInfo := &TokenInfo{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(time.Hour), // Temporary, will be updated
		CreatedAt:   time.Now(),
	}

	if err := client.SetTokenInfo(tempTokenInfo); err != nil {
		return nil, fmt.Errorf("failed to set temporary token: %w", err)
	}

	// Validate and get accurate token information
	debugResp, err := client.DebugToken(context.Background(), accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	// Set accurate token information from debug response
	if err := client.SetTokenFromDebugInfo(accessToken, debugResp); err != nil {
		return nil, fmt.Errorf("failed to set token info: %w", err)
	}

	return client, nil
}

// SetTokenInfo sets the token information in a thread-safe manner
func (c *Client) SetTokenInfo(tokenInfo *TokenInfo) error {
	if tokenInfo == nil {
		return fmt.Errorf("tokenInfo cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.tokenInfo = tokenInfo
	c.accessToken = tokenInfo.AccessToken

	// Store the token using the configured storage
	if err := c.tokenStorage.Store(tokenInfo); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// GetTokenInfo returns the current token information in a thread-safe manner
func (c *Client) GetTokenInfo() *TokenInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.tokenInfo == nil {
		return nil
	}

	// Return a copy to prevent external modification
	tokenCopy := *c.tokenInfo
	return &tokenCopy
}

// IsAuthenticated returns true if the client has a valid access token
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.accessToken != "" && c.tokenInfo != nil
}

// IsTokenExpired returns true if the current token is expired
func (c *Client) IsTokenExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.tokenInfo == nil {
		return true
	}

	return time.Now().After(c.tokenInfo.ExpiresAt)
}

// IsTokenExpiringSoon returns true if the token expires within the given duration
func (c *Client) IsTokenExpiringSoon(within time.Duration) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.tokenInfo == nil {
		return true
	}

	return time.Now().Add(within).After(c.tokenInfo.ExpiresAt)
}

// ValidateToken validates the current token by making a test API call
func (c *Client) ValidateToken() error {
	if !c.IsAuthenticated() {
		return NewAuthenticationError(401, "No token available", "Client is not authenticated")
	}

	if c.IsTokenExpired() {
		return NewAuthenticationError(401, "Token expired", "The access token has expired")
	}

	// Make a simple API call to validate the token
	_, err := c.TestAPICall("GET", "/v1.0/me", map[string]string{
		"fields": "id",
	})

	return err
}

// EnsureValidToken ensures the client has a valid, non-expired token
// It will attempt to refresh the token if it's expired or expiring soon
func (c *Client) EnsureValidToken(ctx context.Context) error {
	if !c.IsAuthenticated() {
		return NewAuthenticationError(401, "No token available", "Client is not authenticated")
	}

	// If token is expired or expiring within 1 hour, try to refresh
	if c.IsTokenExpired() || c.IsTokenExpiringSoon(time.Hour) {
		if err := c.RefreshToken(ctx); err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	return nil
}

// ClearToken removes the current token from the client and storage
func (c *Client) ClearToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.accessToken = ""
	c.tokenInfo = nil

	// Clear from storage
	if err := c.tokenStorage.Delete(); err != nil {
		return fmt.Errorf("failed to clear token from storage: %w", err)
	}

	return nil
}

// GetConfig returns a copy of the client configuration
func (c *Client) GetConfig() *Config {
	// Return a copy to prevent external modification
	configCopy := *c.config
	return &configCopy
}

// UpdateConfig updates the client configuration with validation
// Note: This does not affect already established connections
func (c *Client) UpdateConfig(newConfig *Config) error {
	if newConfig == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate the new configuration
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults for any missing configuration
	newConfig.SetDefaults()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.config = newConfig
	c.baseURL = newConfig.BaseURL

	return nil
}

// Clone creates a new client instance with the same configuration
// but separate token storage and state
func (c *Client) Clone() (*Client, error) {
	config := c.GetConfig()
	return NewClient(config)
}

// CloneWithConfig creates a new client instance with a different configuration
func (c *Client) CloneWithConfig(newConfig *Config) (*Client, error) {
	return NewClient(newConfig)
}

// safeJSONUnmarshal safely unmarshal's JSON with proper error handling for empty responses
func safeJSONUnmarshal(data []byte, v any, context string, requestID string) error {
	// Check for empty response
	if len(data) == 0 {
		return NewAPIError(200, "Empty response", fmt.Sprintf("Received empty response for %s", context), requestID)
	}

	// Check for whitespace-only response
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return NewAPIError(200, "Empty response", fmt.Sprintf("Received whitespace-only response for %s", context), requestID)
	}

	// Check for non-JSON response (common error responses)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return NewAPIError(200, "Invalid JSON response", fmt.Sprintf("Received non-JSON response for %s: %s", context, string(data)), requestID)
	}

	// Attempt to unmarshal
	if err := json.Unmarshal(data, v); err != nil {
		return NewAPIError(200, "Failed to parse JSON response", fmt.Sprintf("JSON parsing failed for %s: %s", context, err.Error()), requestID)
	}

	return nil
}

// GetRateLimitStatus returns the current rate limit status
func (c *Client) GetRateLimitStatus() RateLimitStatus {
	return c.rateLimiter.GetStatus()
}

// IsNearRateLimit returns true if the client is close to hitting rate limits
func (c *Client) IsNearRateLimit(threshold float64) bool {
	return c.rateLimiter.IsNearLimit(threshold)
}

// IsRateLimited returns true if the client is currently rate limited by the API
func (c *Client) IsRateLimited() bool {
	return c.rateLimiter.IsRateLimited()
}

// DisableRateLimiting disables the rate limiter entirely
// Use with caution - this will allow unlimited requests to the API
func (c *Client) DisableRateLimiting() {
	c.rateLimiter = nil
}

// EnableRateLimiting re-enables rate limiting with a new rate limiter
func (c *Client) EnableRateLimiting() {
	if c.rateLimiter == nil {
		rateLimiterConfig := &RateLimiterConfig{
			InitialLimit:      100,
			BackoffMultiplier: 2.0,
			MaxBackoff:        5 * time.Minute,
			QueueSize:         100,
			Logger:            c.config.Logger,
		}
		c.rateLimiter = NewRateLimiter(rateLimiterConfig)
	}
}

// WaitForRateLimit blocks until it's safe to make another request
func (c *Client) WaitForRateLimit(ctx context.Context) error {
	return c.rateLimiter.Wait(ctx)
}

// TestAPICall makes a test API call (for testing purposes only)
func (c *Client) TestAPICall(method, path string, params map[string]string) (*Response, error) {
	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	queryParams := url.Values{}
	for key, value := range params {
		queryParams.Set(key, value)
	}

	switch method {
	case "GET":
		return c.httpClient.GET(path, queryParams, token)
	case "POST":
		return c.httpClient.POST(path, queryParams, token)
	default:
		return c.httpClient.GET(path, queryParams, token)
	}
}

// Compile-time check to ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)
