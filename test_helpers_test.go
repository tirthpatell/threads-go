package threads

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// testClient creates a *Client whose HTTP requests go to the given handler.
func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	config := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	config.BaseURL = server.URL

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("testClient: %v", err)
	}

	// Set a valid token so methods that require auth work
	err = client.SetTokenInfo(&TokenInfo{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      "12345",
		CreatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatalf("testClient SetTokenInfo: %v", err)
	}

	return client
}

// jsonHandler returns an http.HandlerFunc that responds with the given status and JSON body.
func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// newTestHTTPClient creates an HTTPClient pointed at a test server with the given handler and retry config.
func newTestHTTPClient(t *testing.T, handler http.Handler, retryConfig *RetryConfig) *HTTPClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	config := &Config{
		HTTPTimeout: 5 * time.Second,
		Logger:      &noopLogger{},
		RetryConfig: retryConfig,
		BaseURL:     server.URL,
	}
	return NewHTTPClient(config, nil)
}

// testClientConfig creates a Config pointed at a test server.
func testClientConfig(t *testing.T, handler http.Handler) *Config {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	config := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	config.BaseURL = server.URL
	return config
}

// testClientWithConfig creates a *Client using the provided config.
func testClientWithConfig(t *testing.T, config *Config) *Client {
	t.Helper()
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("testClientWithConfig: %v", err)
	}

	err = client.SetTokenInfo(&TokenInfo{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      "12345",
		CreatedAt:   time.Now(),
	})
	if err != nil {
		t.Fatalf("testClientWithConfig SetTokenInfo: %v", err)
	}

	return client
}

// newBareClient creates a minimal *Client without an HTTP server for testing pure logic.
func newBareClient(t *testing.T) *Client {
	t.Helper()
	config := NewConfig()
	config.ClientID = "test-client-id"
	config.ClientSecret = "test-client-secret"
	config.RedirectURI = "https://example.com/callback"
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("newBareClient: %v", err)
	}
	return client
}

// testClientNoAuth creates a *Client without a token set (unauthenticated).
func testClientNoAuth(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	config := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	config.BaseURL = server.URL

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("testClientNoAuth: %v", err)
	}
	return client
}

// noopLogger is a no-op Logger implementation for tests.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, fields ...any) {}
func (n *noopLogger) Info(msg string, fields ...any)  {}
func (n *noopLogger) Warn(msg string, fields ...any)  {}
func (n *noopLogger) Error(msg string, fields ...any) {}
