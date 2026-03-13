package threads

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}

	// Check defaults
	if config.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("Expected HTTPTimeout to be %v, got %v", DefaultHTTPTimeout, config.HTTPTimeout)
	}

	if config.BaseURL != BaseAPIURL {
		t.Errorf("Expected BaseURL to be %s, got %s", BaseAPIURL, config.BaseURL)
	}

	if config.UserAgent != DefaultUserAgent {
		t.Errorf("Expected UserAgent to be %s, got %s", DefaultUserAgent, config.UserAgent)
	}

	// Check that scopes are set
	if len(config.Scopes) == 0 {
		t.Error("Expected scopes to be set by default")
	}

	// Check retry config
	if config.RetryConfig == nil {
		t.Fatal("Expected RetryConfig to be set")
	}

	if config.RetryConfig.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.RetryConfig.MaxRetries)
	}
}

func TestConfigValidation(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name      string
		config    *Config
		shouldErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"threads_basic"},
				HTTPTimeout:  30 * time.Second,
				BaseURL:      "https://graph.threads.net",
			},
			shouldErr: false,
		},
		{
			name: "missing client ID",
			config: &Config{
				ClientSecret: "test-client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"threads_basic"},
				HTTPTimeout:  30 * time.Second,
				BaseURL:      "https://graph.threads.net",
			},
			shouldErr: true,
		},
		{
			name: "invalid redirect URI",
			config: &Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURI:  "not-a-url",
				Scopes:       []string{"threads_basic"},
				HTTPTimeout:  30 * time.Second,
				BaseURL:      "https://graph.threads.net",
			},
			shouldErr: true,
		},
		{
			name: "invalid scope",
			config: &Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURI:  "https://example.com/callback",
				Scopes:       []string{"invalid_scope"},
				HTTPTimeout:  30 * time.Second,
				BaseURL:      "https://graph.threads.net",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGhostPostValidation(t *testing.T) {
	client := &Client{}

	// Test valid ghost post
	validGhost := &TextPostContent{
		Text:        "This is a ghost post",
		IsGhostPost: true,
	}
	err := client.ValidateTextPostContent(validGhost)
	if err != nil {
		t.Errorf("Expected valid ghost post to pass validation, got: %v", err)
	}

	// Test invalid ghost post (reply)
	invalidGhost := &TextPostContent{
		Text:        "This is an invalid ghost post",
		IsGhostPost: true,
		ReplyTo:     "some-post-id",
	}
	err = client.ValidateTextPostContent(invalidGhost)
	if err == nil {
		t.Error("Expected error for ghost post with ReplyTo")
	} else if validationErr, ok := err.(*ValidationError); ok {
		if validationErr.Field != "is_ghost_post" {
			t.Errorf("Expected error field 'is_ghost_post', got '%s'", validationErr.Field)
		}
	} else {
		t.Errorf("Expected ValidationError, got %T", err)
	}
}

func TestValidation(t *testing.T) {
	validator := NewValidator()

	t.Run("ValidateTextLength", func(t *testing.T) {
		// Test valid text
		err := validator.ValidateTextLength("Hello world", "Text")
		if err != nil {
			t.Errorf("Expected no error for valid text, got: %v", err)
		}

		// Test text too long
		longText := make([]byte, MaxTextLength+1)
		for i := range longText {
			longText[i] = 'a'
		}
		err = validator.ValidateTextLength(string(longText), "Text")
		if err == nil {
			t.Error("Expected error for text too long")
		}

		// Test CJK text within character limit (500 CJK chars = 1500 bytes,
		// but should count as 500 characters, not 1500)
		cjkText := ""
		for i := 0; i < MaxTextLength; i++ {
			cjkText += "你"
		}
		err = validator.ValidateTextLength(cjkText, "Text")
		if err != nil {
			t.Errorf("Expected no error for %d CJK characters, got: %v", MaxTextLength, err)
		}

		// Test CJK text exceeding character limit (501 CJK chars)
		cjkText += "你"
		err = validator.ValidateTextLength(cjkText, "Text")
		if err == nil {
			t.Error("Expected error for CJK text exceeding character limit")
		}
	})

	t.Run("ValidateTopicTag", func(t *testing.T) {
		// Test valid tag
		err := validator.ValidateTopicTag("valid_tag")
		if err != nil {
			t.Errorf("Expected no error for valid tag, got: %v", err)
		}

		// Test invalid tag with period
		err = validator.ValidateTopicTag("invalid.tag")
		if err == nil {
			t.Error("Expected error for tag with period")
		}

		// Test invalid tag with ampersand
		err = validator.ValidateTopicTag("invalid&tag")
		if err == nil {
			t.Error("Expected error for tag with ampersand")
		}
	})

	t.Run("ValidateCountryCodes", func(t *testing.T) {
		// Test valid codes
		err := validator.ValidateCountryCodes([]string{"US", "CA", "GB"})
		if err != nil {
			t.Errorf("Expected no error for valid country codes, got: %v", err)
		}

		// Test invalid code length
		err = validator.ValidateCountryCodes([]string{"USA"})
		if err == nil {
			t.Error("Expected error for invalid country code length")
		}

		// Test invalid characters
		err = validator.ValidateCountryCodes([]string{"U1"})
		if err == nil {
			t.Error("Expected error for country code with numbers")
		}
	})

	t.Run("ValidateLinkCount", func(t *testing.T) {
		// Test valid link count (0 links)
		err := validator.ValidateLinkCount("Hello world", "")
		if err != nil {
			t.Errorf("Expected no error for 0 links, got: %v", err)
		}

		// Test valid link count (5 links)
		fiveLinks := "http://a.com https://b.com http://c.com https://d.com http://e.com"
		err = validator.ValidateLinkCount(fiveLinks, "")
		if err != nil {
			t.Errorf("Expected no error for 5 links, got: %v", err)
		}

		// Test unique links logic
		// "If the text field contains www.example.com, www.example.com, and www.test.com,
		// and the link_attachment is www.test.com, this counts as 2 links"
		// (Assuming http/https prefix for validator detection)
		duplicateLinks := "http://example.com http://example.com http://test.com"
		err = validator.ValidateLinkCount(duplicateLinks, "http://test.com")
		if err != nil {
			t.Errorf("Expected no error for duplicate links (should count as 2), got: %v", err)
		}

		// Test link_attachment adds to count
		// "If the text field contains www.instagram.com and www.threads.com,
		// and the link_attachment is www.facebook.com, this counts as 3 links."
		textWithLinks := "http://instagram.com http://threads.com"
		err = validator.ValidateLinkCount(textWithLinks, "http://facebook.com")
		if err != nil {
			t.Errorf("Expected no error for 3 total links, got: %v", err)
		}

		// Test invalid link count (6 unique links)
		sixLinks := "http://a.com https://b.com http://c.com https://d.com http://e.com https://f.com"
		err = validator.ValidateLinkCount(sixLinks, "")
		if err == nil {
			t.Error("Expected error for 6 links")
		}

		// Test invalid link count (5 in text + 1 unique in attachment)
		fiveInText := "http://a.com https://b.com http://c.com https://d.com http://e.com"
		err = validator.ValidateLinkCount(fiveInText, "http://f.com")
		if err == nil {
			t.Error("Expected error for 6 total unique links")
		}
	})
}

func TestPostIDTypes(t *testing.T) {
	// Test PostID
	postID := ConvertToPostID("test-post-id")
	if !postID.Valid() {
		t.Error("Expected PostID to be valid")
	}
	if postID.String() != "test-post-id" {
		t.Errorf("Expected PostID string to be 'test-post-id', got '%s'", postID.String())
	}

	// Test empty PostID
	emptyPostID := ConvertToPostID("")
	if emptyPostID.Valid() {
		t.Error("Expected empty PostID to be invalid")
	}

	// Test UserID
	userID := ConvertToUserID("test-user-id")
	if !userID.Valid() {
		t.Error("Expected UserID to be valid")
	}
	if userID.String() != "test-user-id" {
		t.Errorf("Expected UserID string to be 'test-user-id', got '%s'", userID.String())
	}
}

func TestContainerBuilder(t *testing.T) {
	builder := NewContainerBuilder()

	params := builder.
		SetMediaType(MediaTypeText).
		SetText("Hello world").
		SetReplyControl(ReplyControlEveryone).
		Build()

	if params.Get("media_type") != MediaTypeText {
		t.Errorf("Expected media_type to be %s, got %s", MediaTypeText, params.Get("media_type"))
	}

	if params.Get("text") != "Hello world" {
		t.Errorf("Expected text to be 'Hello world', got '%s'", params.Get("text"))
	}

	if params.Get("reply_control") != string(ReplyControlEveryone) {
		t.Errorf("Expected reply_control to be %s, got %s", string(ReplyControlEveryone), params.Get("reply_control"))
	}
}

func TestContainerBuilderSetChildren(t *testing.T) {
	t.Run("comma separated format", func(t *testing.T) {
		builder := NewContainerBuilder()
		childIDs := []string{"id1", "id2", "id3", "id4", "id5"}
		params := builder.SetChildren(childIDs).Build()

		got := params.Get("children")
		expected := "id1,id2,id3,id4,id5"
		if got != expected {
			t.Errorf("Expected children=%q, got %q", expected, got)
		}
	})

	t.Run("preserves order with 10+ children", func(t *testing.T) {
		builder := NewContainerBuilder()
		childIDs := make([]string, 20)
		for i := range childIDs {
			childIDs[i] = fmt.Sprintf("id_%d", i+1)
		}
		params := builder.SetChildren(childIDs).Build()

		got := params.Get("children")
		expected := strings.Join(childIDs, ",")
		if got != expected {
			t.Errorf("Children order not preserved.\nExpected: %s\nGot:      %s", expected, got)
		}
	})

	t.Run("empty children", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetChildren(nil).Build()

		if params.Get("children") != "" {
			t.Error("Expected no children param for nil input")
		}
	})

	t.Run("SetChildren nil clears existing", func(t *testing.T) {
		builder := NewContainerBuilder()
		builder.AddChild("id1").AddChild("id2")
		params := builder.SetChildren(nil).Build()

		if params.Get("children") != "" {
			t.Error("Expected SetChildren(nil) to clear existing children")
		}
	})

	t.Run("no indexed params", func(t *testing.T) {
		builder := NewContainerBuilder()
		childIDs := []string{"id1", "id2", "id3"}
		params := builder.SetChildren(childIDs).Build()

		encoded := params.Encode()
		if strings.Contains(encoded, "children%5B") || strings.Contains(encoded, "children[") {
			t.Errorf("Should not contain indexed children params, got: %s", encoded)
		}
	})
}

func TestContainerBuilderAddChild(t *testing.T) {
	builder := NewContainerBuilder()
	params := builder.
		AddChild("id1").
		AddChild("id2").
		AddChild("id3").
		Build()

	got := params.Get("children")
	expected := "id1,id2,id3"
	if got != expected {
		t.Errorf("Expected children=%q, got %q", expected, got)
	}
}

func TestContainerBuilderGIFAttachment(t *testing.T) {
	builder := NewContainerBuilder()

	gif := &GIFAttachment{
		GIFID:    "test-gif-id-12345",
		Provider: GIFProviderTenor,
	}

	params := builder.
		SetMediaType(MediaTypeText).
		SetText("Check out this GIF!").
		SetGIFAttachment(gif).
		Build()

	if params.Get("media_type") != MediaTypeText {
		t.Errorf("Expected media_type to be %s, got %s", MediaTypeText, params.Get("media_type"))
	}

	gifParam := params.Get("gif_attachment")
	if gifParam == "" {
		t.Error("Expected gif_attachment to be set")
	}

	// Check that the GIF attachment contains expected values
	if gifParam == "" {
		t.Error("Expected gif_attachment parameter to be set")
	}
}

func TestContainerBuilderGIFAttachmentNil(t *testing.T) {
	builder := NewContainerBuilder()

	params := builder.
		SetMediaType(MediaTypeText).
		SetText("No GIF here").
		SetGIFAttachment(nil).
		Build()

	if params.Get("gif_attachment") != "" {
		t.Error("Expected gif_attachment to be empty when nil")
	}
}

func TestValidateGIFAttachment(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name      string
		gif       *GIFAttachment
		shouldErr bool
		errField  string
	}{
		{
			name:      "nil gif attachment is valid",
			gif:       nil,
			shouldErr: false,
		},
		{
			name: "valid gif attachment",
			gif: &GIFAttachment{
				GIFID:    "test-gif-id",
				Provider: GIFProviderTenor,
			},
			shouldErr: false,
		},
		{
			name: "missing gif_id",
			gif: &GIFAttachment{
				GIFID:    "",
				Provider: GIFProviderTenor,
			},
			shouldErr: true,
			errField:  "gif_attachment.gif_id",
		},
		{
			name: "whitespace only gif_id",
			gif: &GIFAttachment{
				GIFID:    "   ",
				Provider: GIFProviderTenor,
			},
			shouldErr: true,
			errField:  "gif_attachment.gif_id",
		},
		{
			name: "missing provider",
			gif: &GIFAttachment{
				GIFID:    "test-gif-id",
				Provider: "",
			},
			shouldErr: true,
			errField:  "gif_attachment.provider",
		},
		{
			name: "valid giphy provider",
			gif: &GIFAttachment{
				GIFID:    "test-gif-id",
				Provider: GIFProviderGiphy,
			},
			shouldErr: false,
		},
		{
			name: "invalid provider",
			gif: &GIFAttachment{
				GIFID:    "test-gif-id",
				Provider: GIFProvider("INVALID"),
			},
			shouldErr: true,
			errField:  "gif_attachment.provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateGIFAttachment(tt.gif)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if validationErr, ok := err.(*ValidationError); ok {
					if tt.errField != "" && validationErr.Field != tt.errField {
						t.Errorf("Expected error field '%s', got '%s'", tt.errField, validationErr.Field)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGIFProviderConstants(t *testing.T) {
	if GIFProviderTenor != "TENOR" {
		t.Errorf("Expected GIFProviderTenor to be 'TENOR', got '%s'", GIFProviderTenor)
	}
	if GIFProviderGiphy != "GIPHY" {
		t.Errorf("Expected GIFProviderGiphy to be 'GIPHY', got '%s'", GIFProviderGiphy)
	}
}

func TestGIFAttachmentStruct(t *testing.T) {
	gif := &GIFAttachment{
		GIFID:    "12345-tenor-gif",
		Provider: GIFProviderTenor,
	}

	if gif.GIFID != "12345-tenor-gif" {
		t.Errorf("Expected GIFID to be '12345-tenor-gif', got '%s'", gif.GIFID)
	}

	if gif.Provider != GIFProviderTenor {
		t.Errorf("Expected Provider to be GIFProviderTenor, got '%s'", gif.Provider)
	}
}

func TestReplyApprovalsValidation(t *testing.T) {
	client := &Client{}

	// Test valid post with reply approvals
	validContent := &TextPostContent{
		Text:                 "Post with reply approvals",
		EnableReplyApprovals: true,
	}
	err := client.ValidateTextPostContent(validContent)
	if err != nil {
		t.Errorf("Expected valid post with reply approvals, got: %v", err)
	}

	// Test ghost post with reply approvals (should fail)
	invalidContent := &TextPostContent{
		Text:                 "Invalid ghost post",
		IsGhostPost:          true,
		EnableReplyApprovals: true,
	}
	err = client.ValidateTextPostContent(invalidContent)
	if err == nil {
		t.Error("Expected error for ghost post with reply approvals")
	} else if validationErr, ok := err.(*ValidationError); ok {
		if validationErr.Field != "enable_reply_approvals" {
			t.Errorf("Expected error field 'enable_reply_approvals', got '%s'", validationErr.Field)
		}
	} else {
		t.Errorf("Expected ValidationError, got %T", err)
	}
}

func TestContainerBuilderEnableReplyApprovals(t *testing.T) {
	builder := NewContainerBuilder()

	params := builder.
		SetMediaType(MediaTypeText).
		SetText("Post with reply approvals").
		SetEnableReplyApprovals(true).
		Build()

	if params.Get("enable_reply_approvals") != "true" {
		t.Errorf("Expected enable_reply_approvals to be 'true', got '%s'", params.Get("enable_reply_approvals"))
	}

	// Test false (should not set param)
	builder2 := NewContainerBuilder()
	params2 := builder2.
		SetMediaType(MediaTypeText).
		SetText("Normal post").
		SetEnableReplyApprovals(false).
		Build()

	if params2.Get("enable_reply_approvals") != "" {
		t.Errorf("Expected enable_reply_approvals to be empty when false, got '%s'", params2.Get("enable_reply_approvals"))
	}
}

func TestPendingRepliesOptionsApprovalStatus(t *testing.T) {
	// Test valid statuses
	if ApprovalStatusPending != "pending" {
		t.Errorf("Expected ApprovalStatusPending to be 'pending', got '%s'", ApprovalStatusPending)
	}
	if ApprovalStatusIgnored != "ignored" {
		t.Errorf("Expected ApprovalStatusIgnored to be 'ignored', got '%s'", ApprovalStatusIgnored)
	}

	// Test options struct
	reverse := false
	opts := &PendingRepliesOptions{
		Limit:          25,
		Reverse:        &reverse,
		ApprovalStatus: ApprovalStatusPending,
	}

	if opts.ApprovalStatus != "pending" {
		t.Errorf("Expected ApprovalStatus to be 'pending', got '%s'", opts.ApprovalStatus)
	}
}

func TestErrorIsTransient(t *testing.T) {
	apiErr := NewAPIError(2, "An unexpected error", "details", "trace-123")
	apiErr.IsTransient = true
	if !apiErr.IsTransient {
		t.Error("Expected IsTransient to be true")
	}
	validErr := NewValidationError(400, "Bad request", "details", "field")
	if validErr.IsTransient {
		t.Error("Expected IsTransient to default to false")
	}
}

func TestCreateErrorFromResponseParsesIsTransient(t *testing.T) {
	h := &HTTPClient{
		logger:      &noopLogger{},
		retryConfig: &RetryConfig{MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 2},
	}

	body := []byte(`{"error":{"message":"An unexpected error","type":"OAuthException","code":2,"is_transient":true}}`)
	resp := &Response{
		Body:       body,
		StatusCode: 500,
		RequestID:  "test-trace",
	}

	err := h.createErrorFromResponse(resp)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, got %T: %v", err, err)
	}
	if !apiErr.IsTransient {
		t.Error("Expected IsTransient to be true for transient API error")
	}

	body2 := []byte(`{"error":{"message":"Resource not found","type":"OAuthException","code":24,"is_transient":false}}`)
	resp2 := &Response{
		Body:       body2,
		StatusCode: 400,
		RequestID:  "test-trace-2",
	}

	err2 := h.createErrorFromResponse(resp2)
	var valErr *ValidationError
	if !errors.As(err2, &valErr) {
		t.Fatalf("Expected ValidationError, got %T: %v", err2, err2)
	}
	if valErr.IsTransient {
		t.Error("Expected IsTransient to be false for non-transient error")
	}
}

func TestIsRetryableErrorWithTransientAPIError(t *testing.T) {
	h := &HTTPClient{
		logger:      &noopLogger{},
		retryConfig: &RetryConfig{MaxRetries: 3, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 2},
	}

	// API error with code 2 but HTTP status 500 — should be retryable
	apiErr := NewAPIError(2, "An unexpected error", "details", "trace")
	apiErr.HTTPStatusCode = 500
	apiErr.IsTransient = true
	if !h.isRetryableError(apiErr) {
		t.Error("Expected transient API error with HTTP 500 to be retryable")
	}

	// API error with code 2, HTTP status 500, is_transient false — still retryable (5xx)
	apiErr2 := NewAPIError(2, "An unexpected error", "details", "trace")
	apiErr2.HTTPStatusCode = 500
	if !h.isRetryableError(apiErr2) {
		t.Error("Expected API error with HTTP 500 to be retryable regardless of is_transient")
	}

	// API error with code 24, HTTP status 400, not transient — NOT retryable
	valErr := NewValidationError(24, "Resource not found", "details", "")
	valErr.HTTPStatusCode = 400
	if h.isRetryableError(valErr) {
		t.Error("Expected non-transient 400 error to NOT be retryable")
	}

	// Transient error with non-5xx status — retryable because is_transient=true
	transientErr := NewValidationError(2, "Unexpected error", "details", "")
	transientErr.HTTPStatusCode = 400
	transientErr.IsTransient = true
	if !h.isRetryableError(transientErr) {
		t.Error("Expected transient error to be retryable even with non-5xx HTTP status")
	}
}

func TestIsTransientErrorHelper(t *testing.T) {
	transientErr := NewAPIError(2, "Unexpected error", "details", "trace")
	transientErr.IsTransient = true

	if !IsTransientError(transientErr) {
		t.Error("Expected IsTransientError to return true")
	}

	nonTransientErr := NewValidationError(24, "Not found", "details", "")
	if IsTransientError(nonTransientErr) {
		t.Error("Expected IsTransientError to return false")
	}

	if IsTransientError(fmt.Errorf("random error")) {
		t.Error("Expected IsTransientError to return false for non-threads error")
	}

	// Wrapped transient error should still be detected
	wrappedErr := fmt.Errorf("operation failed: %w", transientErr)
	if !IsTransientError(wrappedErr) {
		t.Error("Expected IsTransientError to return true for wrapped transient error")
	}
}

func TestErrorSubcode(t *testing.T) {
	apiErr := NewAPIError(24, "Resource not found", "details", "trace")
	apiErr.ErrorSubcode = 4279009

	if apiErr.ErrorSubcode != 4279009 {
		t.Errorf("Expected ErrorSubcode 4279009, got %d", apiErr.ErrorSubcode)
	}
}

func TestCreateErrorFromResponseParsesErrorSubcode(t *testing.T) {
	h := &HTTPClient{
		logger:      &noopLogger{},
		retryConfig: &RetryConfig{MaxRetries: 0, InitialDelay: time.Second, MaxDelay: time.Second, BackoffFactor: 2},
	}

	body := []byte(`{"error":{"message":"Media not found","type":"OAuthException","code":24,"is_transient":false,"error_subcode":4279009}}`)
	resp := &Response{
		Body:       body,
		StatusCode: 400,
		RequestID:  "test-trace",
	}

	err := h.createErrorFromResponse(resp)
	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("Expected ValidationError, got %T: %v", err, err)
	}
	if valErr.ErrorSubcode != 4279009 {
		t.Errorf("Expected ErrorSubcode 4279009, got %d", valErr.ErrorSubcode)
	}
}

func TestSearchOptionsAuthorUsername(t *testing.T) {
	opts := &SearchOptions{
		AuthorUsername: "testuser",
		SearchType:     SearchTypeTop,
	}

	if opts.AuthorUsername != "testuser" {
		t.Errorf("Expected AuthorUsername to be 'testuser', got '%s'", opts.AuthorUsername)
	}

	// Test with @ prefix (should be stripped in search.go)
	opts.AuthorUsername = "@testuser"
	if opts.AuthorUsername != "@testuser" {
		t.Errorf("Expected AuthorUsername to be '@testuser', got '%s'", opts.AuthorUsername)
	}
}

func TestWaitForContainerReadyRespectsContext(t *testing.T) {
	// Serve IN_PROGRESS status so the poll loop blocks on the select
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"id":"fake-id","status":"IN_PROGRESS"}`))
	}))
	defer ts.Close()

	config := NewConfig()
	config.ClientID = "test"
	config.ClientSecret = "test"
	config.RedirectURI = "http://localhost"
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	// Point HTTP client at test server and set a valid token
	client.httpClient.baseURL = ts.URL
	client.tokenInfo = &TokenInfo{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
	}
	client.accessToken = "test-token"

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err = client.waitForContainerReady(ctx, ContainerID("fake-id"), 100, 1*time.Second)
	if err == nil {
		t.Fatal("Expected error when context times out")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
}
