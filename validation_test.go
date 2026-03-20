package threads

import (
	"testing"
	"time"
)

func TestValidatePostContent(t *testing.T) {
	v := NewValidator()

	t.Run("nil content returns error", func(t *testing.T) {
		err := v.ValidatePostContent(nil, 0)
		if err == nil {
			t.Fatal("Expected error for nil content")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("non-nil content returns nil", func(t *testing.T) {
		err := v.ValidatePostContent("some content", 0)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}

func TestValidateTextAttachment(t *testing.T) {
	v := NewValidator()

	t.Run("nil is valid", func(t *testing.T) {
		err := v.ValidateTextAttachment(nil)
		if err != nil {
			t.Errorf("Expected no error for nil, got: %v", err)
		}
	})

	t.Run("empty plaintext returns error", func(t *testing.T) {
		err := v.ValidateTextAttachment(&TextAttachment{Plaintext: ""})
		if err == nil {
			t.Fatal("Expected error for empty plaintext")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("valid plaintext", func(t *testing.T) {
		err := v.ValidateTextAttachment(&TextAttachment{Plaintext: "Hello world"})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("plaintext too long", func(t *testing.T) {
		longText := make([]byte, MaxTextAttachmentLength+1)
		for i := range longText {
			longText[i] = 'a'
		}
		err := v.ValidateTextAttachment(&TextAttachment{Plaintext: string(longText)})
		if err == nil {
			t.Fatal("Expected error for plaintext too long")
		}
	})

	t.Run("valid with styling info", func(t *testing.T) {
		err := v.ValidateTextAttachment(&TextAttachment{
			Plaintext: "Hello world styled",
			TextWithStylingInfo: []TextStylingInfo{
				{Offset: 0, Length: 5, StylingInfo: []string{"bold"}},
				{Offset: 6, Length: 5, StylingInfo: []string{"italic"}},
			},
		})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("overlapping styling ranges returns error", func(t *testing.T) {
		err := v.ValidateTextAttachment(&TextAttachment{
			Plaintext: "Hello world styled",
			TextWithStylingInfo: []TextStylingInfo{
				{Offset: 0, Length: 8, StylingInfo: []string{"bold"}},
				{Offset: 5, Length: 5, StylingInfo: []string{"italic"}},
			},
		})
		if err == nil {
			t.Fatal("Expected error for overlapping styling ranges")
		}
	})
}

func TestValidateTextStylingRanges(t *testing.T) {
	v := NewValidator()

	t.Run("no overlap", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 5},
			{Offset: 5, Length: 5},
			{Offset: 10, Length: 5},
		})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("overlapping ranges", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 10},
			{Offset: 5, Length: 10},
		})
		if err == nil {
			t.Fatal("Expected error for overlapping ranges")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{})
		if err != nil {
			t.Errorf("Expected no error for empty slice, got: %v", err)
		}
	})

	t.Run("single range", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 5},
		})
		if err != nil {
			t.Errorf("Expected no error for single range, got: %v", err)
		}
	})
}

func TestValidateTextEntities(t *testing.T) {
	v := NewValidator()

	t.Run("empty slice is valid", func(t *testing.T) {
		err := v.ValidateTextEntities(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("valid spoiler entity", func(t *testing.T) {
		err := v.ValidateTextEntities([]TextEntity{
			{EntityType: "SPOILER", Offset: 0, Length: 5},
		})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("valid lowercase spoiler entity", func(t *testing.T) {
		err := v.ValidateTextEntities([]TextEntity{
			{EntityType: "spoiler", Offset: 0, Length: 5},
		})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("too many entities", func(t *testing.T) {
		entities := make([]TextEntity, MaxTextEntities+1)
		for i := range entities {
			entities[i] = TextEntity{EntityType: "SPOILER", Offset: i * 10, Length: 5}
		}
		err := v.ValidateTextEntities(entities)
		if err == nil {
			t.Fatal("Expected error for too many entities")
		}
	})

	t.Run("missing entity type", func(t *testing.T) {
		err := v.ValidateTextEntities([]TextEntity{
			{EntityType: "", Offset: 0, Length: 5},
		})
		if err == nil {
			t.Fatal("Expected error for missing entity type")
		}
	})

	t.Run("invalid entity type", func(t *testing.T) {
		err := v.ValidateTextEntities([]TextEntity{
			{EntityType: "BOLD", Offset: 0, Length: 5},
		})
		if err == nil {
			t.Fatal("Expected error for invalid entity type")
		}
	})

	t.Run("negative offset", func(t *testing.T) {
		err := v.ValidateTextEntities([]TextEntity{
			{EntityType: "SPOILER", Offset: -1, Length: 5},
		})
		if err == nil {
			t.Fatal("Expected error for negative offset")
		}
	})

	t.Run("zero length", func(t *testing.T) {
		err := v.ValidateTextEntities([]TextEntity{
			{EntityType: "SPOILER", Offset: 0, Length: 0},
		})
		if err == nil {
			t.Fatal("Expected error for zero length")
		}
	})

	t.Run("negative length", func(t *testing.T) {
		err := v.ValidateTextEntities([]TextEntity{
			{EntityType: "SPOILER", Offset: 0, Length: -1},
		})
		if err == nil {
			t.Fatal("Expected error for negative length")
		}
	})
}

func TestValidateCarouselChildren(t *testing.T) {
	v := NewValidator()

	t.Run("valid count", func(t *testing.T) {
		err := v.ValidateCarouselChildren(5)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("minimum valid count", func(t *testing.T) {
		err := v.ValidateCarouselChildren(MinCarouselItems)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("maximum valid count", func(t *testing.T) {
		err := v.ValidateCarouselChildren(MaxCarouselItems)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("too few children", func(t *testing.T) {
		err := v.ValidateCarouselChildren(1)
		if err == nil {
			t.Fatal("Expected error for too few children")
		}
	})

	t.Run("too many children", func(t *testing.T) {
		err := v.ValidateCarouselChildren(MaxCarouselItems + 1)
		if err == nil {
			t.Fatal("Expected error for too many children")
		}
	})
}

func TestValidateSearchOptions(t *testing.T) {
	v := NewValidator()

	t.Run("nil is valid", func(t *testing.T) {
		err := v.ValidateSearchOptions(nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("valid options", func(t *testing.T) {
		err := v.ValidateSearchOptions(&SearchOptions{
			Limit: 25,
			Since: MinSearchTimestamp,
		})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("limit too large", func(t *testing.T) {
		err := v.ValidateSearchOptions(&SearchOptions{
			Limit: MaxPostsPerRequest + 1,
		})
		if err == nil {
			t.Fatal("Expected error for limit too large")
		}
	})

	t.Run("since too old", func(t *testing.T) {
		err := v.ValidateSearchOptions(&SearchOptions{
			Since: 100, // way before MinSearchTimestamp
		})
		if err == nil {
			t.Fatal("Expected error for since too old")
		}
	})

	t.Run("since zero is valid", func(t *testing.T) {
		err := v.ValidateSearchOptions(&SearchOptions{
			Since: 0,
		})
		if err != nil {
			t.Errorf("Expected no error for since=0, got: %v", err)
		}
	})
}

func TestValidatePollAttachment(t *testing.T) {
	v := NewValidator()

	t.Run("nil is valid", func(t *testing.T) {
		err := v.ValidatePollAttachment(nil)
		if err != nil {
			t.Errorf("Expected no error for nil, got: %v", err)
		}
	})

	t.Run("valid two options", func(t *testing.T) {
		err := v.ValidatePollAttachment(&PollAttachment{
			OptionA: "Yes",
			OptionB: "No",
		})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("valid four options", func(t *testing.T) {
		err := v.ValidatePollAttachment(&PollAttachment{
			OptionA: "Option A",
			OptionB: "Option B",
			OptionC: "Option C",
			OptionD: "Option D",
		})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("missing option A returns error", func(t *testing.T) {
		err := v.ValidatePollAttachment(&PollAttachment{
			OptionA: "",
			OptionB: "No",
		})
		if err == nil {
			t.Fatal("Expected error for missing option A")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("missing option B returns error", func(t *testing.T) {
		err := v.ValidatePollAttachment(&PollAttachment{
			OptionA: "Yes",
			OptionB: "",
		})
		if err == nil {
			t.Fatal("Expected error for missing option B")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("option A too long returns error", func(t *testing.T) {
		longOption := string(make([]byte, MaxPollOptionLength+1))
		for i := range []byte(longOption) {
			longOption = longOption[:i] + "a" + longOption[i+1:]
		}
		// simpler approach:
		longStr := ""
		for i := 0; i < MaxPollOptionLength+1; i++ {
			longStr += "a"
		}
		err := v.ValidatePollAttachment(&PollAttachment{
			OptionA: longStr,
			OptionB: "No",
		})
		if err == nil {
			t.Fatal("Expected error for option A too long")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("option C too long returns error", func(t *testing.T) {
		longStr := ""
		for i := 0; i < MaxPollOptionLength+1; i++ {
			longStr += "a"
		}
		err := v.ValidatePollAttachment(&PollAttachment{
			OptionA: "Yes",
			OptionB: "No",
			OptionC: longStr,
		})
		if err == nil {
			t.Fatal("Expected error for option C too long")
		}
	})

	t.Run("option at max length is valid", func(t *testing.T) {
		maxStr := ""
		for i := 0; i < MaxPollOptionLength; i++ {
			maxStr += "a"
		}
		err := v.ValidatePollAttachment(&PollAttachment{
			OptionA: maxStr,
			OptionB: maxStr,
		})
		if err != nil {
			t.Errorf("Expected no error for max-length options, got: %v", err)
		}
	})
}

func TestValidateAltText(t *testing.T) {
	v := NewValidator()

	t.Run("empty is valid", func(t *testing.T) {
		err := v.ValidateAltText("")
		if err != nil {
			t.Errorf("Expected no error for empty alt text, got: %v", err)
		}
	})

	t.Run("valid alt text", func(t *testing.T) {
		err := v.ValidateAltText("A beautiful sunset over the ocean")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("alt text at max length is valid", func(t *testing.T) {
		maxStr := ""
		for i := 0; i < MaxAltTextLength; i++ {
			maxStr += "a"
		}
		err := v.ValidateAltText(maxStr)
		if err != nil {
			t.Errorf("Expected no error for max-length alt text, got: %v", err)
		}
	})

	t.Run("alt text too long returns error", func(t *testing.T) {
		longStr := ""
		for i := 0; i < MaxAltTextLength+1; i++ {
			longStr += "a"
		}
		err := v.ValidateAltText(longStr)
		if err == nil {
			t.Fatal("Expected error for alt text too long")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("unicode characters counted correctly", func(t *testing.T) {
		// Each Japanese character is 1 rune but 3 bytes; MaxAltTextLength runes should be valid
		runeStr := ""
		for i := 0; i < MaxAltTextLength; i++ {
			runeStr += "あ"
		}
		err := v.ValidateAltText(runeStr)
		if err != nil {
			t.Errorf("Expected no error for %d-rune alt text, got: %v", MaxAltTextLength, err)
		}
	})
}

func TestValidateTextStylingValues(t *testing.T) {
	v := NewValidator()

	t.Run("valid bold style", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 5, StylingInfo: []string{"bold"}},
		})
		if err != nil {
			t.Errorf("Expected no error for bold style, got: %v", err)
		}
	})

	t.Run("valid all styles", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 5, StylingInfo: []string{"bold", "italic"}},
			{Offset: 10, Length: 5, StylingInfo: []string{"highlight"}},
			{Offset: 20, Length: 5, StylingInfo: []string{"underline", "strikethrough"}},
		})
		if err != nil {
			t.Errorf("Expected no error for valid styles, got: %v", err)
		}
	})

	t.Run("invalid style value returns error", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 5, StylingInfo: []string{"bold", "superscript"}},
		})
		if err == nil {
			t.Fatal("Expected error for invalid style value 'superscript'")
		}
		if !IsValidationError(err) {
			t.Errorf("Expected ValidationError, got %T", err)
		}
	})

	t.Run("empty style slice is valid", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 5, StylingInfo: []string{}},
		})
		if err != nil {
			t.Errorf("Expected no error for empty style slice, got: %v", err)
		}
	})

	t.Run("unknown style returns error", func(t *testing.T) {
		err := v.validateTextStylingRanges([]TextStylingInfo{
			{Offset: 0, Length: 5, StylingInfo: []string{"BOLD"}}, // uppercase is invalid
		})
		if err == nil {
			t.Fatal("Expected error for uppercase style value")
		}
	})
}

func TestConfigValidatorValidateRequiredFields(t *testing.T) {
	cv := NewConfigValidator()

	t.Run("missing client secret", func(t *testing.T) {
		cfg := &Config{
			ClientID:    "id",
			RedirectURI: "https://example.com/callback",
			Scopes:      []string{"threads_basic"},
			HTTPTimeout: 30 * time.Second,
			BaseURL:     "https://graph.threads.net",
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for missing client secret")
		}
	})

	t.Run("missing redirect URI", func(t *testing.T) {
		cfg := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for missing redirect URI")
		}
	})
}

func TestConfigValidatorValidateHTTPSettings(t *testing.T) {
	cv := NewConfigValidator()

	t.Run("zero timeout", func(t *testing.T) {
		cfg := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  0,
			BaseURL:      "https://graph.threads.net",
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for zero HTTPTimeout")
		}
	})

	t.Run("empty base URL", func(t *testing.T) {
		cfg := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "",
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for empty BaseURL")
		}
	})

	t.Run("invalid base URL scheme", func(t *testing.T) {
		cfg := &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "ftp://example.com",
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for non-HTTP base URL")
		}
	})
}

func TestConfigValidatorValidateRetryConfig(t *testing.T) {
	cv := NewConfigValidator()

	baseConfig := func() *Config {
		return &Config{
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			Scopes:       []string{"threads_basic"},
			HTTPTimeout:  30 * time.Second,
			BaseURL:      "https://graph.threads.net",
		}
	}

	t.Run("nil retry config is valid", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RetryConfig = nil
		err := cv.Validate(cfg)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("negative max retries", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RetryConfig = &RetryConfig{
			MaxRetries:    -1,
			InitialDelay:  time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for negative MaxRetries")
		}
	})

	t.Run("zero initial delay", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RetryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  0,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for zero InitialDelay")
		}
	})

	t.Run("zero max delay", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RetryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      0,
			BackoffFactor: 2.0,
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for zero MaxDelay")
		}
	})

	t.Run("zero backoff factor", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RetryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 0,
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for zero BackoffFactor")
		}
	})

	t.Run("initial delay greater than max delay", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RetryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  time.Minute,
			MaxDelay:      time.Second,
			BackoffFactor: 2.0,
		}
		err := cv.Validate(cfg)
		if err == nil {
			t.Fatal("Expected error for InitialDelay > MaxDelay")
		}
	})

	t.Run("valid retry config", func(t *testing.T) {
		cfg := baseConfig()
		cfg.RetryConfig = &RetryConfig{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		}
		err := cv.Validate(cfg)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}
