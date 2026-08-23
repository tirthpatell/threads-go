package threads

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func baseTestConfig() *Config {
	return &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "https://example.com/callback",
		BaseURL:      "https://graph.threads.net",
	}
}

func TestNewClientDoesNotMutateCallerConfig(t *testing.T) {
	config := baseTestConfig()
	if _, err := NewClient(config); err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if len(config.Scopes) != 0 {
		t.Errorf("NewClient mutated caller Scopes: %v", config.Scopes)
	}
	if config.RetryConfig != nil {
		t.Errorf("NewClient mutated caller RetryConfig: %+v", config.RetryConfig)
	}
	if config.HTTPTimeout != 0 {
		t.Errorf("NewClient mutated caller HTTPTimeout: %v", config.HTTPTimeout)
	}
	if config.MaxResponseBodySize != 0 {
		t.Errorf("NewClient mutated caller MaxResponseBodySize: %d", config.MaxResponseBodySize)
	}
}

func TestUpdateConfigDoesNotMutateCallerConfig(t *testing.T) {
	client := newBareClient(t)

	t.Run("validation failure leaves caller config untouched", func(t *testing.T) {
		invalid := &Config{ClientSecret: "s", RedirectURI: "https://example.com/callback"}
		if err := client.UpdateConfig(invalid); err == nil {
			t.Fatal("expected error for missing ClientID")
		}
		if len(invalid.Scopes) != 0 || invalid.RetryConfig != nil || invalid.HTTPTimeout != 0 {
			t.Errorf("failed UpdateConfig mutated caller config: %+v", invalid)
		}
	})

	t.Run("success does not alias caller config", func(t *testing.T) {
		valid := baseTestConfig()
		if err := client.UpdateConfig(valid); err != nil {
			t.Fatalf("UpdateConfig: %v", err)
		}
		if len(valid.Scopes) != 0 {
			t.Errorf("UpdateConfig mutated caller Scopes: %v", valid.Scopes)
		}

		// Mutating the caller's config afterwards must not reach the client.
		valid.ClientID = "mutated"
		if got := client.GetConfig().ClientID; got != "test-client-id" {
			t.Errorf("client config aliased caller config, got ClientID %q", got)
		}
	})
}

func TestRetryConfigIsCopiedFromCallerConfig(t *testing.T) {
	retryConfig := &RetryConfig{
		MaxRetries:    0,
		InitialDelay:  time.Millisecond,
		MaxDelay:      time.Millisecond,
		BackoffFactor: 2.0,
	}
	config := baseTestConfig()
	config.RetryConfig = retryConfig

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// The caller still owns retryConfig; writing to it must not reach the
	// HTTP layer, which took its own copy.
	retryConfig.MaxRetries = 99

	client.httpClient.mu.RLock()
	got := client.httpClient.retryConfig.MaxRetries
	client.httpClient.mu.RUnlock()
	if got != 0 {
		t.Errorf("HTTP client aliased caller RetryConfig, MaxRetries = %d, want 0", got)
	}
}

func TestValidateBaseURL_InsecureOptOut(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		allowInsecure bool
		wantErr       bool
	}{
		{"https accepted", "https://graph.threads.net", false, false},
		{"loopback name accepted", "http://localhost:8080", false, false},
		{"loopback ip accepted", "http://127.0.0.1:8080", false, false},
		{"localhost subdomain accepted", "http://api.localhost:8080", false, false},
		{"plaintext gateway rejected by default", "http://gateway.internal:8080", false, true},
		{"private ip rejected by default", "http://10.0.0.5:8080", false, true},
		{"plaintext gateway allowed with opt-out", "http://gateway.internal:8080", true, false},
		{"private ip allowed with opt-out", "http://10.0.0.5:8080", true, false},
		{"query string still rejected", "https://graph.threads.net?a=b", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBaseURL(tt.baseURL, tt.allowInsecure)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateBaseURL(%q, %v) = %v, wantErr %v", tt.baseURL, tt.allowInsecure, err, tt.wantErr)
			}
		})
	}
}

func TestAllowInsecureBaseURLReachesHTTPLayer(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"12345","username":"testuser"}`))
	}))
	t.Cleanup(server.Close)

	// A plaintext, non-loopback host: the request URL stays insecure so
	// executeRequest's per-request check is what decides, while a custom
	// dialer quietly routes the connection to the test server.
	const insecureBaseURL = "http://gateway.internal:8080"
	serverAddr := strings.TrimPrefix(server.URL, "http://")
	redirectingClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, serverAddr)
			},
		},
	}

	// Without the opt-out the transport layer refuses the request outright,
	// even though the connection would have succeeded.
	rejecting := NewHTTPClient(&Config{BaseURL: insecureBaseURL}, nil)
	rejecting.mu.Lock()
	rejecting.client = redirectingClient
	rejecting.mu.Unlock()
	if _, err := rejecting.GET("/12345", nil, "test-access-token"); err == nil ||
		!strings.Contains(err.Error(), "invalid BaseURL") {
		t.Fatalf("expected the request to be rejected for an insecure BaseURL, got %v", err)
	}
	if requests != 0 {
		t.Fatalf("expected no request to reach the server, got %d", requests)
	}

	config := baseTestConfig()
	config.BaseURL = insecureBaseURL
	if _, err := NewClient(config); err == nil {
		t.Fatal("expected NewClient to reject a plaintext non-loopback BaseURL")
	}

	config = baseTestConfig()
	config.BaseURL = insecureBaseURL
	config.AllowInsecureBaseURL = true
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient with AllowInsecureBaseURL: %v", err)
	}
	if err := client.SetTokenInfo(&TokenInfo{
		AccessToken: "test-access-token",
		TokenType:   TokenTypeBearer,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      "12345",
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SetTokenInfo: %v", err)
	}
	client.httpClient.mu.Lock()
	client.httpClient.client = redirectingClient
	client.httpClient.mu.Unlock()

	if _, err := client.GetUser(context.Background(), UserID("12345")); err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if requests != 1 {
		t.Errorf("expected 1 request, got %d", requests)
	}
}

func TestSanitizeURLRedactsUserinfo(t *testing.T) {
	u, err := url.Parse("https://user:secret@graph.threads.net/me?access_token=abc&fields=id")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	got := sanitizeURL(u)
	if strings.Contains(got, "secret") || strings.Contains(got, "user:") {
		t.Errorf("sanitizeURL leaked userinfo: %s", got)
	}
	if strings.Contains(got, "abc") {
		t.Errorf("sanitizeURL leaked access_token: %s", got)
	}
	if u.User == nil {
		t.Error("sanitizeURL mutated the caller's URL")
	}

	// Userinfo must be dropped even when there is no query string to redact.
	noQuery, err := url.Parse("https://user:secret@graph.threads.net/me")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := sanitizeURL(noQuery); strings.Contains(got, "secret") {
		t.Errorf("sanitizeURL leaked userinfo with empty query: %s", got)
	}
}

func TestUnsafeTokenUserIDIsAuthenticationError(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	if err := client.SetTokenInfo(&TokenInfo{
		AccessToken: "test-access-token",
		TokenType:   TokenTypeBearer,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      "victim/threads_publish?creation_id=attacker",
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SetTokenInfo: %v", err)
	}

	if _, err := client.GetMe(context.Background()); err == nil || !IsAuthenticationError(err) {
		t.Fatalf("GetMe: expected AuthenticationError for unsafe token user ID, got %v", err)
	}
}

func TestConcurrentUpdateConfigAndConfigReaders(t *testing.T) {
	client := newBareClient(t)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			config := baseTestConfig()
			if i%2 == 0 {
				config.ClientID = "other-client-id"
			}
			if err := client.UpdateConfig(config); err != nil {
				t.Errorf("UpdateConfig: %v", err)
				return
			}
		}
	}()

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if _, _, err := client.GetAuthURL([]string{"threads_basic"}); err != nil {
					t.Errorf("GetAuthURL: %v", err)
					return
				}
				_ = client.GetAppAccessTokenShorthand()
				_ = client.GetConfig()
			}
		}()
	}

	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// TestCredentialsAndDestinationStayPaired guards the coupling between the
// client configuration and the transport snapshot: a request that reads
// credentials from one config must not be sent to a BaseURL installed by a
// concurrent UpdateConfig.
func TestCredentialsAndDestinationStayPaired(t *testing.T) {
	newTokenServer := func(wantSecret string, mismatches *int32) *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				atomic.AddInt32(mismatches, 1)
				return
			}
			if got := r.PostFormValue("client_secret"); got != wantSecret {
				atomic.AddInt32(mismatches, 1)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tok","token_type":"bearer","expires_in":3600,"user_id":12345}`))
		}))
		t.Cleanup(server.Close)
		return server
	}

	var mismatches int32
	serverA := newTokenServer("secret-a", &mismatches)
	serverB := newTokenServer("secret-b", &mismatches)

	configFor := func(secret, baseURL string) *Config {
		config := baseTestConfig()
		config.ClientSecret = secret
		config.BaseURL = baseURL
		return config
	}

	client, err := NewClient(configFor("secret-a", serverA.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var updater, exchangers sync.WaitGroup
	stop := make(chan struct{})

	updater.Add(1)
	go func() {
		defer updater.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			config := configFor("secret-a", serverA.URL)
			if i%2 == 1 {
				config = configFor("secret-b", serverB.URL)
			}
			if err := client.UpdateConfig(config); err != nil {
				t.Errorf("UpdateConfig: %v", err)
				return
			}
		}
	}()

	for i := 0; i < 4; i++ {
		exchangers.Add(1)
		go func() {
			defer exchangers.Done()
			for j := 0; j < 50; j++ {
				if err := client.ExchangeCodeForToken(context.Background(), "code", "state", "state"); err != nil {
					t.Errorf("ExchangeCodeForToken: %v", err)
					return
				}
			}
		}()
	}

	exchangers.Wait()
	close(stop)
	updater.Wait()

	if got := atomic.LoadInt32(&mismatches); got != 0 {
		t.Errorf("%d requests delivered a client_secret to the wrong destination", got)
	}
}

func TestSanitizeHeadersRedactsCredentialCarryingHeaders(t *testing.T) {
	h := &HTTPClient{}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer super-secret")
	headers.Set("Cookie", "session=super-secret")
	headers.Set("Proxy-Authorization", "Basic super-secret")
	headers.Set("X-Api-Key", "super-secret")
	headers.Set("X-Access-Token", "super-secret")
	headers.Set("X-App-Secret", "super-secret")
	headers.Set("Content-Type", "application/json")
	headers.Set("User-Agent", "threads-go/test")

	sanitized := h.sanitizeHeaders(headers)
	for name, value := range sanitized {
		if strings.Contains(value, "super-secret") {
			t.Errorf("header %q leaked its value: %s", name, value)
		}
	}
	if sanitized["Content-Type"] != "application/json" {
		t.Errorf("Content-Type should not be redacted, got %q", sanitized["Content-Type"])
	}
	if sanitized["User-Agent"] != "threads-go/test" {
		t.Errorf("User-Agent should not be redacted, got %q", sanitized["User-Agent"])
	}
}

// TestRefreshTokenUsesCoupledSnapshot mirrors the credential-pairing test for
// the refresh endpoint, which carries a bearer token rather than app creds.
func TestRefreshTokenUsesCoupledSnapshot(t *testing.T) {
	newRefreshServer := func(name string, seen *int32) *httptest.Server {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("access_token") != "" {
				atomic.AddInt32(seen, 1)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"tok-` + name + `","token_type":"bearer","expires_in":3600}`))
		}))
		t.Cleanup(server.Close)
		return server
	}

	var seenA, seenB int32
	serverA := newRefreshServer("a", &seenA)
	serverB := newRefreshServer("b", &seenB)

	configFor := func(baseURL string) *Config {
		config := baseTestConfig()
		config.BaseURL = baseURL
		return config
	}

	client, err := NewClient(configFor(serverA.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.SetTokenInfo(&TokenInfo{
		AccessToken: "test-access-token",
		TokenType:   TokenTypeBearer,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      "12345",
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("SetTokenInfo: %v", err)
	}

	var updater, refreshers sync.WaitGroup
	stop := make(chan struct{})

	updater.Add(1)
	go func() {
		defer updater.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			baseURL := serverA.URL
			if i%2 == 1 {
				baseURL = serverB.URL
			}
			if err := client.UpdateConfig(configFor(baseURL)); err != nil {
				t.Errorf("UpdateConfig: %v", err)
				return
			}
		}
	}()

	for i := 0; i < 4; i++ {
		refreshers.Add(1)
		go func() {
			defer refreshers.Done()
			for j := 0; j < 50; j++ {
				if err := client.RefreshToken(context.Background()); err != nil {
					t.Errorf("RefreshToken: %v", err)
					return
				}
			}
		}()
	}

	refreshers.Wait()
	close(stop)
	updater.Wait()

	if total := atomic.LoadInt32(&seenA) + atomic.LoadInt32(&seenB); total != 200 {
		t.Errorf("expected 200 refresh requests, got %d", total)
	}
}
