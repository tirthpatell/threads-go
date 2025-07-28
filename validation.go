package threads

import (
	"fmt"
	"strings"
)

// Validator provides common validation methods
type Validator struct{}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// ValidatePostContent performs common validation for all post types
func (v *Validator) ValidatePostContent(content interface{}, _ int) error {
	if content == nil {
		return NewValidationError(400, "Content cannot be nil", "Post content is required", "content")
	}
	return nil
}

// ValidateTextLength validates text doesn't exceed maximum length
func (v *Validator) ValidateTextLength(text string, fieldName string) error {
	if len(text) > MaxTextLength {
		return NewValidationError(400,
			fmt.Sprintf("%s too long", fieldName),
			fmt.Sprintf("%s is limited to %d characters", fieldName, MaxTextLength),
			strings.ToLower(fieldName))
	}
	return nil
}

// ValidateMediaURL validates media URLs for basic format and accessibility
func (v *Validator) ValidateMediaURL(mediaURL, mediaType string) error {
	if mediaURL == "" {
		return NewValidationError(400,
			"Media URL cannot be empty",
			fmt.Sprintf("%s URL is required", mediaType),
			"media_url")
	}

	// Basic URL format validation
	if !strings.HasPrefix(mediaURL, "http://") && !strings.HasPrefix(mediaURL, "https://") {
		return NewValidationError(400,
			"Invalid media URL format",
			"Media URL must start with http:// or https://",
			"media_url")
	}

	return nil
}

// ValidateTopicTag validates a topic tag according to Threads API rules
func (v *Validator) ValidateTopicTag(tag string) error {
	if tag == "" {
		return nil // Empty tag is valid (optional)
	}

	// Check for forbidden characters
	if strings.Contains(tag, ".") {
		return NewValidationError(400,
			"Invalid topic tag",
			"Topic tags cannot contain periods (.)",
			"topic_tag")
	}

	if strings.Contains(tag, "&") {
		return NewValidationError(400,
			"Invalid topic tag",
			"Topic tags cannot contain ampersands (&)",
			"topic_tag")
	}

	return nil
}

// ValidateCountryCodes validates ISO 3166-1 alpha-2 country codes
func (v *Validator) ValidateCountryCodes(codes []string) error {
	if len(codes) == 0 {
		return nil // Empty list is valid
	}

	for _, code := range codes {
		if len(code) != 2 {
			return NewValidationError(400,
				"Invalid country code",
				fmt.Sprintf("Country code '%s' must be 2 characters (ISO 3166-1 alpha-2)", code),
				"country_codes")
		}

		// Convert to uppercase for consistency
		code = strings.ToUpper(code)

		// Basic validation - should be alphabetic
		for _, char := range code {
			if char < 'A' || char > 'Z' {
				return NewValidationError(400,
					"Invalid country code",
					fmt.Sprintf("Country code '%s' must contain only letters", code),
					"country_codes")
			}
		}
	}

	return nil
}

// ValidateCarouselChildren validates carousel children count
func (v *Validator) ValidateCarouselChildren(childrenCount int) error {
	if childrenCount < MinCarouselItems {
		return NewValidationError(400,
			"Insufficient children",
			fmt.Sprintf("Carousel must have at least %d children", MinCarouselItems),
			"children")
	}

	if childrenCount > MaxCarouselItems {
		return NewValidationError(400,
			"Too many children",
			fmt.Sprintf("Carousel cannot have more than %d children", MaxCarouselItems),
			"children")
	}

	return nil
}

// ValidatePaginationOptions validates pagination parameters
func (v *Validator) ValidatePaginationOptions(opts *PaginationOptions) error {
	if opts == nil {
		return nil
	}

	if opts.Limit > MaxPostsPerRequest {
		return NewValidationError(400,
			"Limit too large",
			fmt.Sprintf("Maximum limit is %d posts per request", MaxPostsPerRequest),
			"limit")
	}

	return nil
}

// ValidateSearchOptions validates search parameters
func (v *Validator) ValidateSearchOptions(opts *SearchOptions) error {
	if opts == nil {
		return nil
	}

	if opts.Limit > MaxPostsPerRequest {
		return NewValidationError(400,
			"Limit too large",
			fmt.Sprintf("Maximum limit is %d posts per request", MaxPostsPerRequest),
			"limit")
	}

	if opts.Since > 0 && opts.Since < MinSearchTimestamp {
		return NewValidationError(400,
			"Invalid since timestamp",
			fmt.Sprintf("Since timestamp must be greater than or equal to %d", MinSearchTimestamp),
			"since")
	}

	return nil
}

// ConfigValidator validates client configuration
type ConfigValidator struct{}

// NewConfigValidator creates a new config validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// Validate validates the entire configuration
func (cv *ConfigValidator) Validate(c *Config) error {
	validators := []func(*Config) error{
		cv.validateRequiredFields,
		cv.validateRedirectURI,
		cv.validateScopes,
		cv.validateHTTPSettings,
		cv.validateRetryConfig,
	}

	for _, validator := range validators {
		if err := validator(c); err != nil {
			return err
		}
	}
	return nil
}

// validateRequiredFields checks all required fields are present
func (cv *ConfigValidator) validateRequiredFields(c *Config) error {
	if c.ClientID == "" {
		return fmt.Errorf("ClientID is required")
	}

	if c.ClientSecret == "" {
		return fmt.Errorf("ClientSecret is required")
	}

	if c.RedirectURI == "" {
		return fmt.Errorf("RedirectURI is required")
	}

	return nil
}

// validateRedirectURI validates the redirect URI format
func (cv *ConfigValidator) validateRedirectURI(c *Config) error {
	if !strings.HasPrefix(c.RedirectURI, "http://") && !strings.HasPrefix(c.RedirectURI, "https://") {
		return fmt.Errorf("RedirectURI must be a valid HTTP or HTTPS URL")
	}
	return nil
}

// validateScopes validates the configured scopes
func (cv *ConfigValidator) validateScopes(c *Config) error {
	if len(c.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}

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

	return nil
}

// validateHTTPSettings validates HTTP-related configuration
func (cv *ConfigValidator) validateHTTPSettings(c *Config) error {
	if c.HTTPTimeout <= 0 {
		return fmt.Errorf("HTTPTimeout must be positive")
	}

	if c.BaseURL == "" {
		return fmt.Errorf("BaseURL is required")
	}

	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("BaseURL must be a valid HTTP or HTTPS URL")
	}

	return nil
}

// validateRetryConfig validates retry configuration
func (cv *ConfigValidator) validateRetryConfig(c *Config) error {
	if c.RetryConfig == nil {
		return nil
	}

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

	return nil
}
