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

func TestContainerBuilderSetIsCarouselItem(t *testing.T) {
	t.Run("true sets param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetIsCarouselItem(true).Build()
		if params.Get("is_carousel_item") != "true" {
			t.Errorf("Expected is_carousel_item='true', got %q", params.Get("is_carousel_item"))
		}
	})

	t.Run("false does not set param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetIsCarouselItem(false).Build()
		if params.Get("is_carousel_item") != "" {
			t.Errorf("Expected empty is_carousel_item, got %q", params.Get("is_carousel_item"))
		}
	})
}

func TestContainerBuilderSetQuotePostID(t *testing.T) {
	t.Run("non-empty sets param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetQuotePostID("12345").Build()
		if params.Get("quote_post_id") != "12345" {
			t.Errorf("Expected quote_post_id='12345', got %q", params.Get("quote_post_id"))
		}
	})

	t.Run("empty does not set param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetQuotePostID("").Build()
		if params.Get("quote_post_id") != "" {
			t.Errorf("Expected empty quote_post_id, got %q", params.Get("quote_post_id"))
		}
	})
}

func TestContainerBuilderSetPollAttachment(t *testing.T) {
	t.Run("valid poll", func(t *testing.T) {
		builder := NewContainerBuilder()
		poll := &PollAttachment{
			OptionA: "Yes",
			OptionB: "No",
		}
		params := builder.SetPollAttachment(poll).Build()
		pollParam := params.Get("poll_attachment")
		if pollParam == "" {
			t.Fatal("Expected poll_attachment to be set")
		}
		// Should contain the options
		if !strings.Contains(pollParam, "Yes") || !strings.Contains(pollParam, "No") {
			t.Errorf("Expected poll_attachment to contain options, got %q", pollParam)
		}
	})

	t.Run("nil poll", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetPollAttachment(nil).Build()
		if params.Get("poll_attachment") != "" {
			t.Errorf("Expected empty poll_attachment for nil, got %q", params.Get("poll_attachment"))
		}
	})
}

func TestContainerBuilderSetTextEntities(t *testing.T) {
	t.Run("valid entities", func(t *testing.T) {
		builder := NewContainerBuilder()
		entities := []TextEntity{
			{EntityType: "SPOILER", Offset: 0, Length: 5},
			{EntityType: "SPOILER", Offset: 10, Length: 3},
		}
		params := builder.SetTextEntities(entities).Build()
		entitiesParam := params.Get("text_entities")
		if entitiesParam == "" {
			t.Fatal("Expected text_entities to be set")
		}
		if !strings.Contains(entitiesParam, "SPOILER") {
			t.Errorf("Expected text_entities to contain SPOILER, got %q", entitiesParam)
		}
	})

	t.Run("empty entities", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetTextEntities(nil).Build()
		if params.Get("text_entities") != "" {
			t.Errorf("Expected empty text_entities for nil, got %q", params.Get("text_entities"))
		}
	})
}

func TestContainerBuilderSetTextAttachment(t *testing.T) {
	t.Run("valid attachment", func(t *testing.T) {
		builder := NewContainerBuilder()
		attachment := &TextAttachment{
			Plaintext: "Long form content here",
		}
		params := builder.SetTextAttachment(attachment).Build()
		attachmentParam := params.Get("text_attachment")
		if attachmentParam == "" {
			t.Fatal("Expected text_attachment to be set")
		}
		if !strings.Contains(attachmentParam, "Long form content here") {
			t.Errorf("Expected text_attachment to contain text, got %q", attachmentParam)
		}
	})

	t.Run("nil attachment", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetTextAttachment(nil).Build()
		if params.Get("text_attachment") != "" {
			t.Errorf("Expected empty text_attachment for nil, got %q", params.Get("text_attachment"))
		}
	})
}

func TestContainerBuilderSetLinkAttachment(t *testing.T) {
	t.Run("non-empty link", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetLinkAttachment("https://example.com").Build()
		if params.Get("link_attachment") != "https://example.com" {
			t.Errorf("Expected link_attachment='https://example.com', got %q", params.Get("link_attachment"))
		}
	})

	t.Run("empty link", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetLinkAttachment("").Build()
		if params.Get("link_attachment") != "" {
			t.Errorf("Expected empty link_attachment, got %q", params.Get("link_attachment"))
		}
	})
}

func TestContainerBuilderSetAllowlistedCountryCodes(t *testing.T) {
	t.Run("with codes", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetAllowlistedCountryCodes([]string{"US", "CA", "GB"}).Build()
		codes := params["allowlisted_country_codes"]
		if len(codes) != 3 {
			t.Errorf("Expected 3 country codes, got %d", len(codes))
		}
	})

	t.Run("empty codes", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetAllowlistedCountryCodes(nil).Build()
		codes := params["allowlisted_country_codes"]
		if len(codes) != 0 {
			t.Errorf("Expected 0 country codes, got %d", len(codes))
		}
	})
}

func TestContainerBuilderSetAltText(t *testing.T) {
	t.Run("non-empty alt text", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetAltText("A photo of a sunset").Build()
		if params.Get("alt_text") != "A photo of a sunset" {
			t.Errorf("Expected alt_text='A photo of a sunset', got %q", params.Get("alt_text"))
		}
	})

	t.Run("empty alt text", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetAltText("").Build()
		if params.Get("alt_text") != "" {
			t.Errorf("Expected empty alt_text, got %q", params.Get("alt_text"))
		}
	})
}

func TestContainerBuilderSetReplyTo(t *testing.T) {
	t.Run("non-empty reply to", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetReplyTo("post-123").Build()
		if params.Get("reply_to_id") != "post-123" {
			t.Errorf("Expected reply_to_id='post-123', got %q", params.Get("reply_to_id"))
		}
	})

	t.Run("empty reply to", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetReplyTo("").Build()
		if params.Get("reply_to_id") != "" {
			t.Errorf("Expected empty reply_to_id, got %q", params.Get("reply_to_id"))
		}
	})
}

func TestContainerBuilderSetTopicTag(t *testing.T) {
	t.Run("non-empty topic tag", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetTopicTag("golang").Build()
		if params.Get("topic_tag") != "golang" {
			t.Errorf("Expected topic_tag='golang', got %q", params.Get("topic_tag"))
		}
	})

	t.Run("empty topic tag", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetTopicTag("").Build()
		if params.Get("topic_tag") != "" {
			t.Errorf("Expected empty topic_tag, got %q", params.Get("topic_tag"))
		}
	})
}

func TestContainerBuilderSetLocationID(t *testing.T) {
	t.Run("non-empty location ID", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetLocationID("loc-456").Build()
		if params.Get("location_id") != "loc-456" {
			t.Errorf("Expected location_id='loc-456', got %q", params.Get("location_id"))
		}
	})

	t.Run("empty location ID", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetLocationID("").Build()
		if params.Get("location_id") != "" {
			t.Errorf("Expected empty location_id, got %q", params.Get("location_id"))
		}
	})
}

func TestContainerBuilderSetIsSpoilerMedia(t *testing.T) {
	t.Run("true sets param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetIsSpoilerMedia(true).Build()
		if params.Get("is_spoiler_media") != "true" {
			t.Errorf("Expected is_spoiler_media='true', got %q", params.Get("is_spoiler_media"))
		}
	})

	t.Run("false does not set param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetIsSpoilerMedia(false).Build()
		if params.Get("is_spoiler_media") != "" {
			t.Errorf("Expected empty is_spoiler_media, got %q", params.Get("is_spoiler_media"))
		}
	})
}

func TestContainerBuilderSetIsGhostPost(t *testing.T) {
	t.Run("true sets param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetIsGhostPost(true).Build()
		if params.Get("is_ghost_post") != "true" {
			t.Errorf("Expected is_ghost_post='true', got %q", params.Get("is_ghost_post"))
		}
	})

	t.Run("false does not set param", func(t *testing.T) {
		builder := NewContainerBuilder()
		params := builder.SetIsGhostPost(false).Build()
		if params.Get("is_ghost_post") != "" {
			t.Errorf("Expected empty is_ghost_post, got %q", params.Get("is_ghost_post"))
		}
	})
}

func TestContainerBuilderAddChildEmpty(t *testing.T) {
	builder := NewContainerBuilder()
	params := builder.AddChild("").Build()
	if params.Get("children") != "" {
		t.Errorf("Expected empty children for empty childID, got %q", params.Get("children"))
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

// clearThreadsEnv sets all THREADS_* env vars to empty via t.Setenv (auto-restored on cleanup).
func clearThreadsEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"THREADS_CLIENT_ID", "THREADS_CLIENT_SECRET", "THREADS_REDIRECT_URI",
		"THREADS_SCOPES", "THREADS_HTTP_TIMEOUT", "THREADS_BASE_URL",
		"THREADS_USER_AGENT", "THREADS_DEBUG", "THREADS_MAX_RETRIES",
		"THREADS_INITIAL_DELAY", "THREADS_MAX_DELAY", "THREADS_BACKOFF_FACTOR",
	} {
		t.Setenv(key, "")
	}
}

func TestNewConfigFromEnv(t *testing.T) {
	clearThreadsEnv(t)

	t.Run("missing THREADS_CLIENT_ID", func(t *testing.T) {
		_, err := NewConfigFromEnv()
		if err == nil {
			t.Fatal("Expected error for missing THREADS_CLIENT_ID")
		}
	})

	t.Run("missing THREADS_CLIENT_SECRET", func(t *testing.T) {
		t.Setenv("THREADS_CLIENT_ID", "test-id")
		_, err := NewConfigFromEnv()
		if err == nil {
			t.Fatal("Expected error for missing THREADS_CLIENT_SECRET")
		}
	})

	t.Run("missing THREADS_REDIRECT_URI", func(t *testing.T) {
		t.Setenv("THREADS_CLIENT_ID", "test-id")
		t.Setenv("THREADS_CLIENT_SECRET", "test-secret")
		_, err := NewConfigFromEnv()
		if err == nil {
			t.Fatal("Expected error for missing THREADS_REDIRECT_URI")
		}
	})

	t.Run("all required vars set", func(t *testing.T) {
		t.Setenv("THREADS_CLIENT_ID", "env-client-id")
		t.Setenv("THREADS_CLIENT_SECRET", "env-client-secret")
		t.Setenv("THREADS_REDIRECT_URI", "https://example.com/callback")
		config, err := NewConfigFromEnv()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if config.ClientID != "env-client-id" {
			t.Errorf("Expected ClientID 'env-client-id', got %q", config.ClientID)
		}
		if config.ClientSecret != "env-client-secret" {
			t.Errorf("Expected ClientSecret 'env-client-secret', got %q", config.ClientSecret)
		}
	})

	t.Run("optional env vars", func(t *testing.T) {
		t.Setenv("THREADS_CLIENT_ID", "env-client-id")
		t.Setenv("THREADS_CLIENT_SECRET", "env-client-secret")
		t.Setenv("THREADS_REDIRECT_URI", "https://example.com/callback")
		t.Setenv("THREADS_SCOPES", "threads_basic, threads_content_publish")
		t.Setenv("THREADS_HTTP_TIMEOUT", "60s")
		t.Setenv("THREADS_BASE_URL", "https://custom.example.com")
		t.Setenv("THREADS_USER_AGENT", "custom-agent/1.0")
		t.Setenv("THREADS_DEBUG", "true")
		t.Setenv("THREADS_MAX_RETRIES", "5")
		t.Setenv("THREADS_INITIAL_DELAY", "2s")
		t.Setenv("THREADS_MAX_DELAY", "60s")
		t.Setenv("THREADS_BACKOFF_FACTOR", "3.0")

		config, err := NewConfigFromEnv()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(config.Scopes) != 2 {
			t.Errorf("Expected 2 scopes, got %d", len(config.Scopes))
		}
		if config.Scopes[0] != "threads_basic" {
			t.Errorf("Expected first scope 'threads_basic', got %q", config.Scopes[0])
		}
		if config.Scopes[1] != "threads_content_publish" {
			t.Errorf("Expected second scope 'threads_content_publish', got %q", config.Scopes[1])
		}
		if config.HTTPTimeout != 60*time.Second {
			t.Errorf("Expected HTTPTimeout 60s, got %v", config.HTTPTimeout)
		}
		if config.BaseURL != "https://custom.example.com" {
			t.Errorf("Expected BaseURL 'https://custom.example.com', got %q", config.BaseURL)
		}
		if config.UserAgent != "custom-agent/1.0" {
			t.Errorf("Expected UserAgent 'custom-agent/1.0', got %q", config.UserAgent)
		}
		if !config.Debug {
			t.Error("Expected Debug to be true")
		}
		if config.RetryConfig.MaxRetries != 5 {
			t.Errorf("Expected MaxRetries 5, got %d", config.RetryConfig.MaxRetries)
		}
		if config.RetryConfig.InitialDelay != 2*time.Second {
			t.Errorf("Expected InitialDelay 2s, got %v", config.RetryConfig.InitialDelay)
		}
		if config.RetryConfig.MaxDelay != 60*time.Second {
			t.Errorf("Expected MaxDelay 60s, got %v", config.RetryConfig.MaxDelay)
		}
		if config.RetryConfig.BackoffFactor != 3.0 {
			t.Errorf("Expected BackoffFactor 3.0, got %v", config.RetryConfig.BackoffFactor)
		}
	})
}

func TestNewClientFromEnv(t *testing.T) {
	clearThreadsEnv(t)

	t.Run("fails without env vars", func(t *testing.T) {
		_, err := NewClientFromEnv()
		if err == nil {
			t.Fatal("Expected error when env vars are missing")
		}
	})

	t.Run("succeeds with required env vars", func(t *testing.T) {
		t.Setenv("THREADS_CLIENT_ID", "client-id")
		t.Setenv("THREADS_CLIENT_SECRET", "client-secret")
		t.Setenv("THREADS_REDIRECT_URI", "https://example.com/callback")
		client, err := NewClientFromEnv()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
	})
}

func TestNewClientWithToken(t *testing.T) {
	t.Run("empty token returns error", func(t *testing.T) {
		_, err := NewClientWithToken("", &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
		})
		if err == nil {
			t.Fatal("Expected error for empty token")
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		_, err := NewClientWithToken("some-token", nil)
		if err == nil {
			t.Fatal("Expected error for nil config")
		}
	})

	t.Run("debug token fails returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			w.Write([]byte(`{"error":{"message":"Invalid token","type":"OAuthException","code":190}}`))
		}))
		defer ts.Close()

		config := &Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURI:  "https://example.com/callback",
			BaseURL:      ts.URL,
		}

		_, err := NewClientWithToken("invalid-token", config)
		if err == nil {
			t.Fatal("Expected error for invalid token")
		}
	})

	t.Run("valid token with mock server", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{
				"data": {
					"app_id": "123",
					"type": "USER",
					"application": "TestApp",
					"is_valid": true,
					"issued_at": 1700000000,
					"expires_at": 1900000000,
					"scopes": ["threads_basic"],
					"user_id": "456"
				}
			}`))
		}))
		defer ts.Close()

		config := &Config{
			ClientID:     "test-id",
			ClientSecret: "test-secret",
			RedirectURI:  "https://example.com/callback",
			BaseURL:      ts.URL,
		}

		client, err := NewClientWithToken("valid-token", config)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if client == nil {
			t.Fatal("Expected non-nil client")
		}
		if !client.IsAuthenticated() {
			t.Error("Expected client to be authenticated")
		}
	})
}

func TestValidateToken(t *testing.T) {
	t.Run("not authenticated", func(t *testing.T) {
		client := newBareClient(t)
		err := client.ValidateToken()
		if err == nil {
			t.Fatal("Expected error for unauthenticated client")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		client := newBareClient(t)
		client.tokenInfo = &TokenInfo{
			AccessToken: "expired-token",
			ExpiresAt:   time.Now().Add(-time.Hour),
		}
		client.accessToken = "expired-token"
		err := client.ValidateToken()
		if err == nil {
			t.Fatal("Expected error for expired token")
		}
	})

	t.Run("valid token with mock", func(t *testing.T) {
		client := testClient(t, jsonHandler(200, `{"id":"12345"}`))

		err := client.ValidateToken()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
	})
}

func TestGetConfig(t *testing.T) {
	client := newBareClient(t)

	retrieved := client.GetConfig()
	if retrieved == nil {
		t.Fatal("Expected non-nil config")
	}
	if retrieved.ClientID == "" {
		t.Error("Expected non-empty ClientID")
	}

	// Verify it's a copy (modifying shouldn't affect client)
	original := client.config.ClientID
	retrieved.ClientID = "modified"
	if client.config.ClientID != original {
		t.Error("GetConfig should return a copy, not a reference")
	}
}

func TestUpdateConfig(t *testing.T) {
	client := newBareClient(t)

	t.Run("nil config returns error", func(t *testing.T) {
		err := client.UpdateConfig(nil)
		if err == nil {
			t.Fatal("Expected error for nil config")
		}
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		err := client.UpdateConfig(&Config{})
		if err == nil {
			t.Fatal("Expected error for empty config")
		}
	})

	t.Run("valid config updates client", func(t *testing.T) {
		newConfig := &Config{
			ClientID:     "new-id",
			ClientSecret: "new-secret",
			RedirectURI:  "https://new.example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  60 * time.Second,
			BaseURL:      "https://new-api.example.com",
			RetryConfig: &RetryConfig{
				MaxRetries:    5,
				InitialDelay:  time.Second,
				MaxDelay:      30 * time.Second,
				BackoffFactor: 2.0,
			},
		}
		err := client.UpdateConfig(newConfig)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if client.baseURL != "https://new-api.example.com" {
			t.Errorf("Expected baseURL to be updated, got %q", client.baseURL)
		}
	})
}

func TestClone(t *testing.T) {
	client := newBareClient(t)

	cloned, err := client.Clone()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cloned == nil {
		t.Fatal("Expected non-nil cloned client")
	}
	if cloned == client {
		t.Error("Expected Clone to return a different instance")
	}
}

func TestCloneWithConfig(t *testing.T) {
	client := newBareClient(t)

	newConfig := &Config{
		ClientID:     "new-id",
		ClientSecret: "new-secret",
		RedirectURI:  "https://new.example.com/callback",
		Scopes:       []string{"threads_basic"},
		HTTPTimeout:  60 * time.Second,
		BaseURL:      "https://new-api.example.com",
	}

	cloned, err := client.CloneWithConfig(newConfig)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cloned == nil {
		t.Fatal("Expected non-nil cloned client")
	}
}

func TestGetRateLimitStatus(t *testing.T) {
	client := newBareClient(t)

	status := client.GetRateLimitStatus()
	if status.Limit <= 0 {
		t.Errorf("Expected positive limit, got %d", status.Limit)
	}
}

func TestIsNearRateLimit(t *testing.T) {
	client := newBareClient(t)

	// With fresh rate limiter, should not be near limit
	if client.IsNearRateLimit(0.9) {
		t.Error("Expected not near rate limit with fresh limiter")
	}
}

func TestIsRateLimited(t *testing.T) {
	client := newBareClient(t)

	if client.IsRateLimited() {
		t.Error("Expected not rate limited with fresh limiter")
	}
}

func TestDisableAndEnableRateLimiting(t *testing.T) {
	client := newBareClient(t)

	// Disable
	client.DisableRateLimiting()
	if client.rateLimiter != nil {
		t.Error("Expected rateLimiter to be nil after disabling")
	}

	// Enable
	client.EnableRateLimiting()
	if client.rateLimiter == nil {
		t.Error("Expected rateLimiter to be non-nil after enabling")
	}

	// Enable again (should be no-op since already enabled)
	client.EnableRateLimiting()
	if client.rateLimiter == nil {
		t.Error("Expected rateLimiter to still be non-nil")
	}
}

func TestWaitForRateLimit(t *testing.T) {
	client := newBareClient(t)

	// Should return immediately when not rate limited
	ctx := context.Background()
	err := client.WaitForRateLimit(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestTestAPICall(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
		client := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("Expected GET, got %s", r.Method)
			}
			if r.URL.Query().Get("fields") != "id" {
				t.Errorf("Expected fields=id, got %s", r.URL.Query().Get("fields"))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"id":"12345"}`))
		}))

		resp, err := client.TestAPICall("GET", "/v1.0/me", map[string]string{"fields": "id"})
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("POST request", func(t *testing.T) {
		client := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"id":"12345"}`))
		}))

		resp, err := client.TestAPICall("POST", "/v1.0/me/threads", map[string]string{})
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp == nil {
			t.Fatal("Expected non-nil response")
		}
	})

	t.Run("unsupported method falls back to GET", func(t *testing.T) {
		client := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("Expected GET for default, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"id":"12345"}`))
		}))

		resp, err := client.TestAPICall("PATCH", "/v1.0/me", nil)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if resp == nil {
			t.Fatal("Expected non-nil response")
		}
	})
}

func TestSafeJSONUnmarshal(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		var result map[string]string
		err := safeJSONUnmarshal([]byte{}, &result, "test", "req-1")
		if err == nil {
			t.Fatal("Expected error for empty data")
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		var result map[string]string
		err := safeJSONUnmarshal([]byte("   "), &result, "test", "req-1")
		if err == nil {
			t.Fatal("Expected error for whitespace-only data")
		}
	})

	t.Run("non-JSON response", func(t *testing.T) {
		var result map[string]string
		err := safeJSONUnmarshal([]byte("Not JSON data"), &result, "test", "req-1")
		if err == nil {
			t.Fatal("Expected error for non-JSON data")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var result map[string]string
		err := safeJSONUnmarshal([]byte(`{"broken`), &result, "test", "req-1")
		if err == nil {
			t.Fatal("Expected error for invalid JSON")
		}
	})

	t.Run("valid JSON object", func(t *testing.T) {
		var result map[string]string
		err := safeJSONUnmarshal([]byte(`{"key":"value"}`), &result, "test", "req-1")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if result["key"] != "value" {
			t.Errorf("Expected key=value, got %q", result["key"])
		}
	})

	t.Run("valid JSON array", func(t *testing.T) {
		var result []int
		err := safeJSONUnmarshal([]byte(`[1,2,3]`), &result, "test", "req-1")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(result))
		}
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("invalid redirect URI format", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "ftp://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for non-HTTP redirect URI")
		}
	})

	t.Run("empty scopes", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for empty scopes")
		}
	})

	t.Run("invalid scope", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"invalid_scope"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for invalid scope")
		}
	})

	t.Run("negative HTTP timeout", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  -1 * time.Second,
			BaseURL:      "https://graph.threads.net",
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for negative HTTPTimeout")
		}
	})

	t.Run("retry config negative max retries", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
			RetryConfig: &RetryConfig{
				MaxRetries:    -1,
				InitialDelay:  time.Second,
				MaxDelay:      30 * time.Second,
				BackoffFactor: 2.0,
			},
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for negative max retries")
		}
	})

	t.Run("retry config zero initial delay", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
			RetryConfig: &RetryConfig{
				MaxRetries:    3,
				InitialDelay:  0,
				MaxDelay:      30 * time.Second,
				BackoffFactor: 2.0,
			},
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for zero initial delay")
		}
	})

	t.Run("retry config zero max delay", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
			RetryConfig: &RetryConfig{
				MaxRetries:    3,
				InitialDelay:  time.Second,
				MaxDelay:      0,
				BackoffFactor: 2.0,
			},
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for zero max delay")
		}
	})

	t.Run("retry config zero backoff factor", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
			RetryConfig: &RetryConfig{
				MaxRetries:    3,
				InitialDelay:  time.Second,
				MaxDelay:      30 * time.Second,
				BackoffFactor: 0,
			},
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for zero backoff factor")
		}
	})

	t.Run("retry config initial delay greater than max delay", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
			RetryConfig: &RetryConfig{
				MaxRetries:    3,
				InitialDelay:  time.Minute,
				MaxDelay:      time.Second,
				BackoffFactor: 2.0,
			},
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for InitialDelay > MaxDelay")
		}
	})

	t.Run("empty BaseURL", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "",
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for empty BaseURL")
		}
	})

	t.Run("invalid BaseURL scheme", func(t *testing.T) {
		config := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "ftp://graph.threads.net",
		}
		err := config.Validate()
		if err == nil {
			t.Fatal("Expected error for non-HTTP BaseURL")
		}
	})
}

func TestSetTokenInfoNil(t *testing.T) {
	config := NewConfig()
	config.ClientID = "test"
	config.ClientSecret = "secret"
	config.RedirectURI = "http://localhost"
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.SetTokenInfo(nil)
	if err == nil {
		t.Fatal("Expected error for nil tokenInfo")
	}
}

func TestGetTokenInfoNilToken(t *testing.T) {
	config := NewConfig()
	config.ClientID = "test"
	config.ClientSecret = "secret"
	config.RedirectURI = "http://localhost"
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	info := client.GetTokenInfo()
	if info != nil {
		t.Error("Expected nil for client with no token")
	}
}

func TestMemoryTokenStorageLoad(t *testing.T) {
	storage := &MemoryTokenStorage{}

	// Load from empty storage
	_, err := storage.Load()
	if err == nil {
		t.Fatal("Expected error for empty storage")
	}

	// Store and load
	token := &TokenInfo{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	if err := storage.Store(token); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.AccessToken != "test-token" {
		t.Errorf("Expected 'test-token', got %q", loaded.AccessToken)
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

func TestHandleAPIError(t *testing.T) {
	client := newBareClient(t)

	tests := []struct {
		name       string
		resp       *Response
		checkError func(t *testing.T, err error)
	}{
		{
			name: "401 auth error",
			resp: &Response{
				StatusCode: 401,
				Body:       []byte(`{"error":{"message":"Invalid token","type":"OAuthException","code":190}}`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsAuthenticationError(err) {
					t.Errorf("Expected authentication error, got %T", err)
				}
			},
		},
		{
			name: "403 auth error",
			resp: &Response{
				StatusCode: 403,
				Body:       []byte(`{"error":{"message":"Permission denied","type":"OAuthException","code":200}}`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsAuthenticationError(err) {
					t.Errorf("Expected authentication error, got %T", err)
				}
			},
		},
		{
			name: "429 rate limit error",
			resp: &Response{
				StatusCode: 429,
				Body:       []byte(`{"error":{"message":"Rate limited","type":"OAuthException","code":32}}`),
				RateLimit:  &RateLimitInfo{RetryAfter: 60 * time.Second},
			},
			checkError: func(t *testing.T, err error) {
				if !IsRateLimitError(err) {
					t.Errorf("Expected rate limit error, got %T", err)
				}
			},
		},
		{
			name: "429 rate limit error without rate limit info",
			resp: &Response{
				StatusCode: 429,
				Body:       []byte(`{"error":{"message":"Rate limited","type":"OAuthException","code":32}}`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsRateLimitError(err) {
					t.Errorf("Expected rate limit error, got %T", err)
				}
			},
		},
		{
			name: "400 validation error",
			resp: &Response{
				StatusCode: 400,
				Body:       []byte(`{"error":{"message":"Invalid param","type":"OAuthException","code":100}}`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsValidationError(err) {
					t.Errorf("Expected validation error, got %T", err)
				}
			},
		},
		{
			name: "422 validation error",
			resp: &Response{
				StatusCode: 422,
				Body:       []byte(`{"error":{"message":"Unprocessable","type":"OAuthException","code":422}}`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsValidationError(err) {
					t.Errorf("Expected validation error, got %T", err)
				}
			},
		},
		{
			name: "500 generic API error",
			resp: &Response{
				StatusCode: 500,
				Body:       []byte(`{"error":{"message":"Server error","type":"ServerException","code":2,"is_transient":true,"error_subcode":1234,"error_data":{"details":"internal"}}}`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsAPIError(err) {
					t.Errorf("Expected API error, got %T", err)
				}
				if !IsTransientError(err) {
					t.Error("Expected transient error")
				}
			},
		},
		{
			name: "error with zero error code uses status code",
			resp: &Response{
				StatusCode: 503,
				Body:       []byte(`{"error":{"message":"Service unavailable","type":"ServerException","code":0}}`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsAPIError(err) {
					t.Errorf("Expected API error, got %T", err)
				}
			},
		},
		{
			name: "empty body fallback",
			resp: &Response{
				StatusCode: 502,
				Body:       []byte{},
				RequestID:  "req-123",
			},
			checkError: func(t *testing.T, err error) {
				if !IsAPIError(err) {
					t.Errorf("Expected API error, got %T", err)
				}
				if !strings.Contains(err.Error(), "502") {
					t.Errorf("Expected status code in error message, got: %s", err.Error())
				}
			},
		},
		{
			name: "malformed JSON fallback",
			resp: &Response{
				StatusCode: 500,
				Body:       []byte(`not json at all`),
			},
			checkError: func(t *testing.T, err error) {
				if !IsAPIError(err) {
					t.Errorf("Expected API error, got %T", err)
				}
			},
		},
		{
			name: "long body gets truncated in fallback",
			resp: &Response{
				StatusCode: 500,
				Body:       []byte(strings.Repeat("x", 600)),
			},
			checkError: func(t *testing.T, err error) {
				if !IsAPIError(err) {
					t.Errorf("Expected API error, got %T", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.handleAPIError(tt.resp)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			tt.checkError(t, err)
		})
	}
}

func TestEnsureValidToken_RefreshPath(t *testing.T) {
	// Test: token valid, no refresh needed
	t.Run("token valid", func(t *testing.T) {
		client := newBareClient(t)
		client.accessToken = "test-token"
		client.tokenInfo = &TokenInfo{
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		err := client.EnsureValidToken(context.Background())
		if err != nil {
			t.Errorf("Expected no error for valid token, got: %v", err)
		}
	})

	// Test: token expired, refresh fails
	t.Run("expired token refresh fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
			fmt.Fprintln(w, `{"error":{"message":"refresh failed"}}`)
		}))
		defer server.Close()

		config := NewConfig()
		config.ClientID = "test-id"
		config.ClientSecret = "test-secret"
		config.RedirectURI = "https://example.com/callback"
		config.BaseURL = server.URL
		client, err := NewClient(config)
		if err != nil {
			t.Fatal(err)
		}
		client.accessToken = "test-token"
		client.tokenInfo = &TokenInfo{
			ExpiresAt: time.Now().Add(-1 * time.Hour), // expired
			CreatedAt: time.Now().Add(-25 * time.Hour),
		}

		err = client.EnsureValidToken(context.Background())
		if err == nil {
			t.Fatal("Expected error for expired token with failed refresh")
		}
		if !strings.Contains(err.Error(), "failed to refresh token") {
			t.Errorf("Expected 'failed to refresh token' in error, got: %v", err)
		}
	})
}
