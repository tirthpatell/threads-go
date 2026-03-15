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

func TestExchangeCodeForToken_ServerError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid code","type":"OAuthException","code":100}}`))
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

	client, _ := NewClient(config)
	err := client.ExchangeCodeForToken(context.Background(), "bad_code")
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestExchangeCodeForToken_NoExpiresIn(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer","user_id":123}`))
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

	client, _ := NewClient(config)
	err := client.ExchangeCodeForToken(context.Background(), "code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ti := client.GetTokenInfo()
	if ti.AccessToken != "tok" {
		t.Errorf("expected tok, got %s", ti.AccessToken)
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

func TestDebugToken_NoToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)

	_, err := client.DebugToken(context.Background(), "")
	if err == nil {
		t.Fatal("expected error when no access token")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestDebugToken_EmptyInputToken(t *testing.T) {
	// When inputToken is empty, should use client's own access token
	client := testClient(t, jsonHandler(200, `{
		"data": {"type":"USER","is_valid":true,"expires_at":1735689600,"issued_at":1735603200,"user_id":"12345","scopes":["threads_basic"]}
	}`))

	resp, err := client.DebugToken(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Data.IsValid {
		t.Error("expected valid token")
	}
}

func TestDebugToken_ServerError(t *testing.T) {
	client := testClient(t, jsonHandler(401, `{"error":{"message":"invalid token","type":"OAuthException","code":190}}`))
	_, err := client.DebugToken(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestHandleTokenError_AuthError(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.handleTokenError(401, []byte(`{"error":{"message":"token expired","type":"auth_error","code":190}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestHandleTokenError_RateLimitError(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.handleTokenError(429, []byte(`{"error":{"message":"rate limited","type":"rate_limit","code":429}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestHandleTokenError_ValidationError(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.handleTokenError(400, []byte(`{"error":{"message":"bad request","type":"validation","code":400}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestHandleTokenError_GenericError(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.handleTokenError(500, []byte(`{"error":{"message":"server error","type":"server","code":500}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsAPIError(err) {
		t.Errorf("expected APIError, got %T", err)
	}
}

func TestHandleTokenError_UnparseableBody(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.handleTokenError(401, []byte(`not json at all`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError for unparseable body, got %T", err)
	}
}

func TestGetAccessToken(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	token := client.GetAccessToken()
	if token != "test-access-token" {
		t.Errorf("expected test-access-token, got %s", token)
	}
}

func TestLoadTokenFromStorage_Success(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()

	// Pre-store a valid token
	storage := &MemoryTokenStorage{}
	_ = storage.Store(&TokenInfo{
		AccessToken: "stored_token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      "12345",
		CreatedAt:   time.Now(),
	})
	config.TokenStorage = storage

	client, _ := NewClient(config)
	err := client.LoadTokenFromStorage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.GetAccessToken() != "stored_token" {
		t.Errorf("expected stored_token, got %s", client.GetAccessToken())
	}
}

func TestLoadTokenFromStorage_ExpiredToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()

	storage := &MemoryTokenStorage{}
	_ = storage.Store(&TokenInfo{
		AccessToken: "expired_token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(-time.Hour), // expired
		UserID:      "12345",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
	})
	config.TokenStorage = storage

	client, _ := NewClient(config)
	err := client.LoadTokenFromStorage()
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestLoadTokenFromStorage_NoToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()

	// Fresh storage with no token
	config.TokenStorage = &MemoryTokenStorage{}

	client, _ := NewClient(config)
	err := client.LoadTokenFromStorage()
	if err == nil {
		t.Fatal("expected error when no stored token")
	}
}

func TestGetTokenDebugInfo_WithToken(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	info := client.GetTokenDebugInfo()

	if hasToken, ok := info["has_token"].(bool); !ok || !hasToken {
		t.Error("expected has_token to be true")
	}
	if isAuth, ok := info["is_authenticated"].(bool); !ok || !isAuth {
		t.Error("expected is_authenticated to be true")
	}
	if _, ok := info["token_type"]; !ok {
		t.Error("expected token_type in debug info")
	}
	if _, ok := info["user_id"]; !ok {
		t.Error("expected user_id in debug info")
	}
	if _, ok := info["expires_at"]; !ok {
		t.Error("expected expires_at in debug info")
	}
	if _, ok := info["time_until_expiry"]; !ok {
		t.Error("expected time_until_expiry in debug info")
	}
	if _, ok := info["expires_in_hours"]; !ok {
		t.Error("expected expires_in_hours in debug info")
	}
	if _, ok := info["expires_in_days"]; !ok {
		t.Error("expected expires_in_days in debug info")
	}
}

func TestGetTokenDebugInfo_NoToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)
	info := client.GetTokenDebugInfo()

	if hasToken, ok := info["has_token"].(bool); !ok || hasToken {
		t.Error("expected has_token to be false")
	}
	if _, ok := info["token_type"]; ok {
		t.Error("expected no token_type without a token")
	}
}

func TestSetTokenFromDebugInfo_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	debugResp := &DebugTokenResponse{}
	debugResp.Data.IsValid = true
	debugResp.Data.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
	debugResp.Data.IssuedAt = time.Now().Unix()
	debugResp.Data.UserID = "99999"
	debugResp.Data.Scopes = []string{"threads_basic"}

	err := client.SetTokenFromDebugInfo("new_token_abc", debugResp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.GetAccessToken() != "new_token_abc" {
		t.Errorf("expected new_token_abc, got %s", client.GetAccessToken())
	}
	ti := client.GetTokenInfo()
	if ti.UserID != "99999" {
		t.Errorf("expected user ID 99999, got %s", ti.UserID)
	}
}

func TestSetTokenFromDebugInfo_NilResponse(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.SetTokenFromDebugInfo("token", nil)
	if err == nil {
		t.Fatal("expected error for nil debug response")
	}
}

func TestSetTokenFromDebugInfo_InvalidToken(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	debugResp := &DebugTokenResponse{}
	debugResp.Data.IsValid = false

	err := client.SetTokenFromDebugInfo("token", debugResp)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
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

func TestGetAuthURL_DefaultScopes(t *testing.T) {
	config := &Config{
		ClientID:     "my-app-id",
		ClientSecret: "secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)

	authURL := client.GetAuthURL(nil)
	if !strings.Contains(authURL, "threads_basic") {
		t.Error("expected default scope threads_basic in auth URL")
	}
}

func TestExchangeCodeForToken_WithLogger(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer","expires_in":3600,"user_id":123}`))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
		Logger:       &noopLogger{},
	}
	config.SetDefaults()
	config.BaseURL = server.URL

	client, _ := NewClient(config)
	err := client.ExchangeCodeForToken(context.Background(), "code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetLongLivedToken_NoToken(t *testing.T) {
	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
	}
	config.SetDefaults()
	client, _ := NewClient(config)

	err := client.GetLongLivedToken(context.Background())
	if err == nil {
		t.Fatal("expected error when no token")
	}
}

func TestGetLongLivedToken_ServerError(t *testing.T) {
	client := testClient(t, jsonHandler(401, `{"error":{"message":"invalid","type":"OAuthException","code":190}}`))
	err := client.GetLongLivedToken(context.Background())
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestGetLongLivedToken_NoExpiresIn(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"access_token":"ll_tok","token_type":"bearer"}`))
	err := client.GetLongLivedToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ti := client.GetTokenInfo()
	if ti.AccessToken != "ll_tok" {
		t.Errorf("expected ll_tok, got %s", ti.AccessToken)
	}
}

func TestGetLongLivedToken_WithLogger(t *testing.T) {
	handler := jsonHandler(200, `{"access_token":"ll","token_type":"bearer","expires_in":5184000}`)
	config := testClientConfig(t, handler)
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)
	err := client.GetLongLivedToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefreshToken_ServerError(t *testing.T) {
	client := testClient(t, jsonHandler(401, `{"error":{"message":"invalid","type":"OAuthException","code":190}}`))
	err := client.RefreshToken(context.Background())
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestRefreshToken_NoExpiresIn(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"access_token":"ref_tok","token_type":"bearer"}`))
	err := client.RefreshToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefreshToken_WithLogger(t *testing.T) {
	handler := jsonHandler(200, `{"access_token":"ref","token_type":"bearer","expires_in":5184000}`)
	config := testClientConfig(t, handler)
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)
	err := client.RefreshToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadTokenFromStorage_WithLogger(t *testing.T) {
	storage := &MemoryTokenStorage{}
	_ = storage.Store(&TokenInfo{
		AccessToken: "stored",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      "12345",
		CreatedAt:   time.Now(),
	})

	config := &Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		RedirectURI:  "https://example.com/callback",
		Logger:       &noopLogger{},
		TokenStorage: storage,
	}
	config.SetDefaults()

	client, _ := NewClient(config)
	err := client.LoadTokenFromStorage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDebugToken_WithLogger(t *testing.T) {
	handler := jsonHandler(200, `{"data":{"type":"USER","is_valid":true,"expires_at":1735689600,"issued_at":1735603200,"user_id":"12345","scopes":["threads_basic"]}}`)
	config := testClientConfig(t, handler)
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)
	_, err := client.DebugToken(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetTokenFromDebugInfo_WithLogger(t *testing.T) {
	handler := jsonHandler(200, `{}`)
	config := testClientConfig(t, handler)
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)

	debugResp := &DebugTokenResponse{}
	debugResp.Data.IsValid = true
	debugResp.Data.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
	debugResp.Data.IssuedAt = time.Now().Unix()
	debugResp.Data.UserID = "99999"

	err := client.SetTokenFromDebugInfo("tok", debugResp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
