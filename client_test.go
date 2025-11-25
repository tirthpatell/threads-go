package threads

import (
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
			name: "invalid provider",
			gif: &GIFAttachment{
				GIFID:    "test-gif-id",
				Provider: GIFProvider("GIPHY"),
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
	// Verify the TENOR constant is correctly defined
	if GIFProviderTenor != "TENOR" {
		t.Errorf("Expected GIFProviderTenor to be 'TENOR', got '%s'", GIFProviderTenor)
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
