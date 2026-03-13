package threads

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExchangeCodeForToken_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"access_token": "new_token_123",
			"token_type": "bearer",
			"expires_in": 3600,
			"user_id": 99999
		}`))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	config.BaseURL = server.URL

	client, err := NewClient(config)
	if err != nil {
		t.Fatal(err)
	}

	err = client.ExchangeCodeForToken(context.Background(), "auth_code_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !client.IsAuthenticated() {
		t.Error("expected client to be authenticated")
	}
	tokenInfo := client.GetTokenInfo()
	if tokenInfo.AccessToken != "new_token_123" {
		t.Errorf("expected new_token_123, got %s", tokenInfo.AccessToken)
	}
	if tokenInfo.UserID != "99999" {
		t.Errorf("expected user ID 99999, got %s", tokenInfo.UserID)
	}
}

func TestExchangeCodeForToken_EmptyCode(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)

	err := client.ExchangeCodeForToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty code")
	}
}

func TestGetLongLivedToken_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"access_token": "long_lived_token",
		"token_type": "bearer",
		"expires_in": 5184000
	}`))

	err := client.GetLongLivedToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tokenInfo := client.GetTokenInfo()
	if tokenInfo.AccessToken != "long_lived_token" {
		t.Errorf("expected long_lived_token, got %s", tokenInfo.AccessToken)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"access_token": "refreshed_token",
		"token_type": "bearer",
		"expires_in": 5184000
	}`))

	err := client.RefreshToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tokenInfo := client.GetTokenInfo()
	if tokenInfo.AccessToken != "refreshed_token" {
		t.Errorf("expected refreshed_token, got %s", tokenInfo.AccessToken)
	}
}

func TestRefreshToken_NoToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)

	err := client.RefreshToken(context.Background())
	if err == nil {
		t.Fatal("expected error when no token")
	}
}

func TestDebugToken_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": {
			"type": "USER",
			"application": "Test App",
			"is_valid": true,
			"expires_at": 1735689600,
			"issued_at": 1735603200,
			"user_id": "12345",
			"scopes": ["threads_basic"]
		}
	}`))

	resp, err := client.DebugToken(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Data.IsValid {
		t.Error("expected valid token")
	}
	if resp.Data.UserID != "12345" {
		t.Errorf("expected user ID 12345, got %s", resp.Data.UserID)
	}
}

func TestGetAuthURL_ContainsRequiredParams(t *testing.T) {
	config := &Config{
		ClientID:     "my-app-id",
		ClientSecret: "secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)

	authURL := client.GetAuthURL([]string{"threads_basic"})
	if authURL == "" {
		t.Fatal("expected non-empty auth URL")
	}
	for _, param := range []string{"client_id=my-app-id", "response_type=code", "scope=threads_basic"} {
		if !strings.Contains(authURL, param) {
			t.Errorf("expected auth URL to contain %q, got %s", param, authURL)
		}
	}
}

func TestTokenExpiration(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)

	_ = client.SetTokenInfo(&TokenInfo{
		AccessToken: "expired",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(-time.Hour),
		UserID:      "12345",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
	})

	if !client.IsTokenExpired() {
		t.Error("expected token to be expired")
	}
	if !client.IsTokenExpiringSoon(time.Hour) {
		t.Error("expected token to be expiring soon")
	}
}
