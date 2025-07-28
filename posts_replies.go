package threads

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// CreateReply creates a reply to a specific post or reply
func (c *Client) CreateReply(ctx context.Context, content *PostContent) (*Post, error) {
	if content == nil {
		return nil, NewValidationError(400, "Content cannot be nil", "PostContent is required", "content")
	}

	if strings.TrimSpace(content.ReplyTo) == "" {
		return nil, NewValidationError(400, "Reply target is required", "Must specify reply_to_id", "reply_to")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build request parameters based on media type
	mediaType := content.MediaType
	if mediaType == "" {
		mediaType = MediaTypeText // Default to TEXT for replies
	}

	params := url.Values{
		"media_type":  {mediaType},
		"reply_to_id": {content.ReplyTo},
	}

	// Add text if provided
	if strings.TrimSpace(content.Text) != "" {
		params.Set("text", content.Text)
	}

	// Create container first
	containerID, err := c.createContainer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply container: %w", err)
	}

	// Wait recommended 10 seconds before publishing reply
	if c.config.Logger != nil {
		c.config.Logger.Info("Reply container created, waiting before publishing", "container_id", containerID)
	}

	// Use context timeout or fixed delay
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(ReplyPublishDelay):
	}

	// Publish the container
	post, err := c.publishContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to publish reply: %w", err)
	}

	return post, nil
}

// ReplyToPost creates a reply to a specific post
func (c *Client) ReplyToPost(ctx context.Context, postID PostID, content *PostContent) (*Post, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "Cannot reply without specifying the post to reply to", "post_id")
	}

	if content == nil {
		return nil, NewValidationError(400, "Content cannot be nil", "PostContent is required", "content")
	}

	// Set the reply-to field
	content.ReplyTo = postID.String()

	// Use CreateReply to handle the actual reply creation
	return c.CreateReply(ctx, content)
}
