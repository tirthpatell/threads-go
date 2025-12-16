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

	// Validate link count (including link_attachment)
	if err := validator.ValidateLinkCount(content.Text, content.LinkAttachment); err != nil {
		return err
	}

	// Validate text entities (spoilers) if present
	if err := validator.ValidateTextEntities(content.TextEntities); err != nil {
		return err
	}

	// Validate text attachment if present
	if err := validator.ValidateTextAttachment(content.TextAttachment); err != nil {
		return err
	}

	// Validate GIF attachment if present
	if err := validator.ValidateGIFAttachment(content.GIFAttachment); err != nil {
		return err
	}

	// Text attachment can only be used with TEXT-only posts
	if content.TextAttachment != nil {
		// Cannot be used with polls
		if content.PollAttachment != nil {
			return NewValidationError(400,
				"Text attachment incompatible with poll",
				"Text attachments cannot be used with polls",
				"text_attachment")
		}

		// If main post has link_attachment, text attachment cannot have link_attachment_url
		if content.LinkAttachment != "" && content.TextAttachment.LinkAttachmentURL != "" {
			return NewValidationError(400,
				"Duplicate link attachments",
				"If the main post has a link_attachment, the text attachment cannot have a link_attachment_url",
				"text_attachment.link_attachment_url")
		}
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

	// Validate link count
	if err := validator.ValidateLinkCount(content.Text, ""); err != nil {
		return err
	}

	// Validate text entities (spoilers) if present
	if err := validator.ValidateTextEntities(content.TextEntities); err != nil {
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

	// Validate link count
	if err := validator.ValidateLinkCount(content.Text, ""); err != nil {
		return err
	}

	// Validate text entities (spoilers) if present
	if err := validator.ValidateTextEntities(content.TextEntities); err != nil {
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

	// Validate link count
	if err := validator.ValidateLinkCount(content.Text, ""); err != nil {
		return err
	}

	// Validate text entities (spoilers) if present
	if err := validator.ValidateTextEntities(content.TextEntities); err != nil {
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
