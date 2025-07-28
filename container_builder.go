package threads

import (
	"encoding/json"
	"net/url"
	"strings"
)

// ContainerBuilder helps build container creation parameters
type ContainerBuilder struct {
	params url.Values
}

// NewContainerBuilder creates a new container builder
func NewContainerBuilder() *ContainerBuilder {
	return &ContainerBuilder{
		params: url.Values{},
	}
}

// SetMediaType sets the media type
func (b *ContainerBuilder) SetMediaType(mediaType string) *ContainerBuilder {
	b.params.Set("media_type", mediaType)
	return b
}

// SetText sets the text content
func (b *ContainerBuilder) SetText(text string) *ContainerBuilder {
	if strings.TrimSpace(text) != "" {
		b.params.Set("text", text)
	}
	return b
}

// SetImageURL sets the image URL for image posts
func (b *ContainerBuilder) SetImageURL(imageURL string) *ContainerBuilder {
	if imageURL != "" {
		b.params.Set("image_url", imageURL)
	}
	return b
}

// SetVideoURL sets the video URL for video posts
func (b *ContainerBuilder) SetVideoURL(videoURL string) *ContainerBuilder {
	if videoURL != "" {
		b.params.Set("video_url", videoURL)
	}
	return b
}

// SetAltText sets the alt text for media
func (b *ContainerBuilder) SetAltText(altText string) *ContainerBuilder {
	if altText != "" {
		b.params.Set("alt_text", altText)
	}
	return b
}

// SetReplyControl sets who can reply to the post
func (b *ContainerBuilder) SetReplyControl(replyControl ReplyControl) *ContainerBuilder {
	if replyControl != "" {
		b.params.Set("reply_control", string(replyControl))
	}
	return b
}

// SetReplyTo sets the ID of the post being replied to
func (b *ContainerBuilder) SetReplyTo(replyToID string) *ContainerBuilder {
	if replyToID != "" {
		b.params.Set("reply_to_id", replyToID)
	}
	return b
}

// SetTopicTag sets the topic tag
func (b *ContainerBuilder) SetTopicTag(tag string) *ContainerBuilder {
	if tag != "" {
		b.params.Set("topic_tag", tag)
	}
	return b
}

// SetLocationID sets the location ID
func (b *ContainerBuilder) SetLocationID(locationID string) *ContainerBuilder {
	if locationID != "" {
		b.params.Set("location_id", locationID)
	}
	return b
}

// SetQuotePostID sets the quoted post ID
func (b *ContainerBuilder) SetQuotePostID(quotePostID string) *ContainerBuilder {
	if quotePostID != "" {
		b.params.Set("quote_post_id", quotePostID)
	}
	return b
}

// SetLinkAttachment sets the link attachment
func (b *ContainerBuilder) SetLinkAttachment(linkURL string) *ContainerBuilder {
	if linkURL != "" {
		b.params.Set("link_attachment", linkURL)
	}
	return b
}

// SetPollAttachment sets the poll attachment
func (b *ContainerBuilder) SetPollAttachment(poll *PollAttachment) *ContainerBuilder {
	if poll != nil {
		pollJSON, err := json.Marshal(poll)
		if err == nil {
			b.params.Set("poll_attachment", string(pollJSON))
		}
	}
	return b
}

// SetAllowlistedCountryCodes sets geo-gating country codes
func (b *ContainerBuilder) SetAllowlistedCountryCodes(codes []string) *ContainerBuilder {
	for _, code := range codes {
		b.params.Add("allowlisted_country_codes", code)
	}
	return b
}

// AddChild adds a child container ID (for carousel posts)
func (b *ContainerBuilder) AddChild(childID string) *ContainerBuilder {
	b.params.Add("children", childID)
	return b
}

// SetChildren sets all children container IDs at once (for carousel posts)
func (b *ContainerBuilder) SetChildren(childIDs []string) *ContainerBuilder {
	for i, childID := range childIDs {
		b.params.Add("children", childID)
		// Also add as indexed parameter for API compatibility
		b.params.Set(b.childIndexKey(i), childID)
	}
	return b
}

// SetAutoPublishText sets whether to auto-publish text posts
func (b *ContainerBuilder) SetAutoPublishText(autoPublish bool) *ContainerBuilder {
	if autoPublish {
		b.params.Set("auto_publish_text", "true")
	}
	return b
}

// SetIsCarouselItem marks this as a carousel item
func (b *ContainerBuilder) SetIsCarouselItem(isCarouselItem bool) *ContainerBuilder {
	if isCarouselItem {
		b.params.Set("is_carousel_item", "true")
	}
	return b
}

// Build returns the built parameters
func (b *ContainerBuilder) Build() url.Values {
	return b.params
}

// childIndexKey generates the indexed child parameter key
func (b *ContainerBuilder) childIndexKey(index int) string {
	return "children[" + b.toString(index) + "]"
}

// toString converts an interface to string
func (b *ContainerBuilder) toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		// Convert int to string manually
		if val == 0 {
			return "0"
		}
		sign := ""
		if val < 0 {
			sign = "-"
			val = -val
		}
		result := ""
		for val > 0 {
			result = string(rune('0'+val%10)) + result
			val /= 10
		}
		return sign + result
	default:
		return ""
	}
}
