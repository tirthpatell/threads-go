package threads

import (
	"fmt"
	"strings"
)

// ValidateTextPostContent validates text post content according to Threads API limits
func (c *Client) ValidateTextPostContent(content *TextPostContent) error {
	validator := NewValidator()

	if content == nil {
		return NewValidationError(400, "Content cannot be nil", "Text post content is required", "content")
	}

	// Validate text length (500-character limit)
	if err := validator.ValidateTextLength(content.Text, "Text"); err != nil {
		return err
	}

	// Validate topic tag if present
	if content.TopicTag != "" {
		if err := validator.ValidateTopicTag(content.TopicTag); err != nil {
			return err
		}
	}

	// Validate country codes if present
	if len(content.AllowlistedCountryCodes) > 0 {
		if err := validator.ValidateCountryCodes(content.AllowlistedCountryCodes); err != nil {
			return err
		}
	}

	return nil
}

// ValidateImagePostContent validates image post content according to Threads API limits
func (c *Client) ValidateImagePostContent(content *ImagePostContent) error {
	validator := NewValidator()

	if content == nil {
		return NewValidationError(400, "Content cannot be nil", "Image post content is required", "content")
	}

	// Validate text length if present (500-character limit)
	if err := validator.ValidateTextLength(content.Text, "Text"); err != nil {
		return err
	}

	// Validate image URL
	if err := validator.ValidateMediaURL(content.ImageURL, "image"); err != nil {
		return err
	}

	// Validate topic tag if present
	if content.TopicTag != "" {
		if err := validator.ValidateTopicTag(content.TopicTag); err != nil {
			return err
		}
	}

	// Validate country codes if present
	if len(content.AllowlistedCountryCodes) > 0 {
		if err := validator.ValidateCountryCodes(content.AllowlistedCountryCodes); err != nil {
			return err
		}
	}

	return nil
}

// ValidateVideoPostContent validates video post content according to Threads API limits
func (c *Client) ValidateVideoPostContent(content *VideoPostContent) error {
	validator := NewValidator()

	if content == nil {
		return NewValidationError(400, "Content cannot be nil", "Video post content is required", "content")
	}

	// Validate text length if present (500-character limit)
	if err := validator.ValidateTextLength(content.Text, "Text"); err != nil {
		return err
	}

	// Validate video URL
	if err := validator.ValidateMediaURL(content.VideoURL, "video"); err != nil {
		return err
	}

	// Validate topic tag if present
	if content.TopicTag != "" {
		if err := validator.ValidateTopicTag(content.TopicTag); err != nil {
			return err
		}
	}

	// Validate country codes if present
	if len(content.AllowlistedCountryCodes) > 0 {
		if err := validator.ValidateCountryCodes(content.AllowlistedCountryCodes); err != nil {
			return err
		}
	}

	return nil
}

// ValidateCarouselPostContent validates carousel post content according to Threads API limits
func (c *Client) ValidateCarouselPostContent(content *CarouselPostContent) error {
	validator := NewValidator()

	if content == nil {
		return NewValidationError(400, "Content cannot be nil", "Carousel post content is required", "content")
	}

	// Validate text length if present (500-character limit)
	if err := validator.ValidateTextLength(content.Text, "Text"); err != nil {
		return err
	}

	// Validate children count (2-20 limit)
	if err := validator.ValidateCarouselChildren(len(content.Children)); err != nil {
		return err
	}

	// Validate topic tag if present
	if content.TopicTag != "" {
		if err := validator.ValidateTopicTag(content.TopicTag); err != nil {
			return err
		}
	}

	// Validate country codes if present
	if len(content.AllowlistedCountryCodes) > 0 {
		if err := validator.ValidateCountryCodes(content.AllowlistedCountryCodes); err != nil {
			return err
		}
	}

	return nil
}

// ValidateCarouselChildren validates carousel children containers
func (c *Client) ValidateCarouselChildren(childrenIDs []string) error {
	validator := NewValidator()

	// Validate that children IDs are provided
	if len(childrenIDs) == 0 {
		return NewValidationError(400, "Children required", "Carousel must have at least one child container", "children")
	}

	// Validate the count using the existing validator method
	if err := validator.ValidateCarouselChildren(len(childrenIDs)); err != nil {
		return err
	}

	// Validate each child ID is not empty
	for i, childID := range childrenIDs {
		if strings.TrimSpace(childID) == "" {
			return NewValidationError(400, "Invalid child ID", fmt.Sprintf("Child ID at index %d cannot be empty", i), "children")
		}
	}

	return nil
}

// ValidateTopicTag validates a topic tag format
func (c *Client) ValidateTopicTag(tag string) error {
	validator := NewValidator()
	return validator.ValidateTopicTag(tag)
}

// ValidateCountryCodes validates country codes
func (c *Client) ValidateCountryCodes(codes []string) error {
	validator := NewValidator()
	return validator.ValidateCountryCodes(codes)
}
