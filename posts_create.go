package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// CreateTextPost creates a new text post on Threads
func (c *Client) CreateTextPost(ctx context.Context, content *TextPostContent) (*Post, error) {
	// Validate content according to API limits
	if err := c.ValidateTextPostContent(content); err != nil {
		return nil, err
	}

	if strings.TrimSpace(content.Text) == "" {
		return nil, NewValidationError(400, "Text content is required", ErrEmptyPostID, "text")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Handle auto_publish_text flow differently
	if content.AutoPublishText {
		return c.createAndPublishTextPostDirectly(ctx, content)
	}

	// Standard container creation and publishing flow
	containerID, err := c.createTextContainer(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create text container: %w", err)
	}

	// Wait for container to be ready
	if err := c.waitForContainerReady(ctx, ContainerID(containerID), DefaultContainerPollMaxAttempts, DefaultContainerPollInterval); err != nil {
		return nil, fmt.Errorf("container not ready for publishing: %w", err)
	}

	// Publish the container
	post, err := c.publishContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to publish text post: %w", err)
	}

	return post, nil
}

// CreateImagePost creates a new image post on Threads
func (c *Client) CreateImagePost(ctx context.Context, content *ImagePostContent) (*Post, error) {
	// Validate content according to API limits
	if err := c.ValidateImagePostContent(content); err != nil {
		return nil, err
	}

	if strings.TrimSpace(content.ImageURL) == "" {
		return nil, NewValidationError(400, "Image URL is required", "Post must have an image URL", "image_url")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Create container first
	containerID, err := c.createImageContainer(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create image container: %w", err)
	}

	// Wait for container to be ready
	if err := c.waitForContainerReady(ctx, ContainerID(containerID), DefaultContainerPollMaxAttempts, DefaultContainerPollInterval); err != nil {
		return nil, fmt.Errorf("container not ready for publishing: %w", err)
	}

	// Publish the container
	post, err := c.publishContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to publish image post: %w", err)
	}

	return post, nil
}

// CreateVideoPost creates a new video post on Threads
func (c *Client) CreateVideoPost(ctx context.Context, content *VideoPostContent) (*Post, error) {
	// Validate content according to API limits
	if err := c.ValidateVideoPostContent(content); err != nil {
		return nil, err
	}

	if strings.TrimSpace(content.VideoURL) == "" {
		return nil, NewValidationError(400, "Video URL is required", "Post must have a video URL", "video_url")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Create container first
	containerID, err := c.createVideoContainer(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create video container: %w", err)
	}

	// Wait for container to be ready
	if err := c.waitForContainerReady(ctx, ContainerID(containerID), DefaultContainerPollMaxAttempts, DefaultContainerPollInterval); err != nil {
		return nil, fmt.Errorf("container not ready for publishing: %w", err)
	}

	// Publish the container
	post, err := c.publishContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to publish video post: %w", err)
	}

	return post, nil
}

// CreateCarouselPost creates a new carousel post on Threads
func (c *Client) CreateCarouselPost(ctx context.Context, content *CarouselPostContent) (*Post, error) {
	// Validate content according to API limits
	if err := c.ValidateCarouselPostContent(content); err != nil {
		return nil, err
	}

	if len(content.Children) == 0 {
		return nil, NewValidationError(400, "Children containers are required", "Carousel post must have at least one child container", "children")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Create container first
	containerID, err := c.createCarouselContainer(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create carousel container: %w", err)
	}

	// Wait for container to be ready
	if err := c.waitForContainerReady(ctx, ContainerID(containerID), DefaultContainerPollMaxAttempts, DefaultContainerPollInterval); err != nil {
		return nil, fmt.Errorf("container not ready for publishing: %w", err)
	}

	// Publish the container
	post, err := c.publishContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to publish carousel post: %w", err)
	}

	return post, nil
}

// CreateQuotePost creates a new quote post on Threads
// CreateQuotePost creates a quote post using any supported content type with a quoted post ID.
// This method acts as a router, directing to the appropriate creation method based on content type.
//
// Supported content types:
//   - *TextPostContent: Creates a text quote post
//   - *ImagePostContent: Creates an image quote post
//   - *VideoPostContent: Creates a video quote post
//   - *CarouselPostContent: Creates a carousel quote post
//
// The quotedPostID parameter specifies which post to quote.
func (c *Client) CreateQuotePost(ctx context.Context, content interface{}, quotedPostID string) (*Post, error) {
	if strings.TrimSpace(quotedPostID) == "" {
		return nil, NewValidationError(400, "Quoted post ID is required", "Quote post must reference an existing post", "quoted_post_id")
	}

	switch v := content.(type) {
	case *TextPostContent:
		// Set the quoted post ID and delegate to text post creation
		v.QuotedPostID = quotedPostID
		return c.CreateTextPost(ctx, v)

	case *ImagePostContent:
		// Set the quoted post ID and delegate to image post creation
		v.QuotedPostID = quotedPostID
		return c.CreateImagePost(ctx, v)

	case *VideoPostContent:
		// Set the quoted post ID and delegate to video post creation
		v.QuotedPostID = quotedPostID
		return c.CreateVideoPost(ctx, v)

	case *CarouselPostContent:
		// Set the quoted post ID and delegate to carousel post creation
		v.QuotedPostID = quotedPostID
		return c.CreateCarouselPost(ctx, v)

	default:
		return nil, fmt.Errorf("unsupported content type for quote post: %T", content)
	}
}

// RepostPost reposts an existing post on Threads using the direct repost endpoint
func (c *Client) RepostPost(ctx context.Context, postID PostID) (*Post, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "Cannot repost without a post ID", "post_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Use the direct repost endpoint
	path := fmt.Sprintf("/%s/repost", postID.String())
	resp, err := c.httpClient.POST(path, nil, c.getAccessTokenSafe())
	if err != nil {
		return nil, fmt.Errorf("failed to create repost: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response to get repost ID
	var repostResp struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(resp.Body, &repostResp); err != nil {
		return nil, NewAPIError(resp.StatusCode, "Failed to parse repost response", err.Error(), resp.RequestID)
	}

	if repostResp.ID == "" {
		return nil, NewAPIError(resp.StatusCode, "Repost ID not returned", "API response missing repost ID", resp.RequestID)
	}

	// Fetch the created repost details
	return c.GetPost(ctx, ConvertToPostID(repostResp.ID))
}

// CreateMediaContainer creates a media container for use in carousel posts
func (c *Client) CreateMediaContainer(ctx context.Context, mediaType, mediaURL, altText string) (ContainerID, error) {
	if mediaType == "" {
		return "", NewValidationError(400, "Media type is required", "Must specify IMAGE or VIDEO", "media_type")
	}

	if mediaURL == "" {
		return "", NewValidationError(400, "Media URL is required", "Must provide a valid media URL", "media_url")
	}

	// Validate media URL
	validator := NewValidator()
	if err := validator.ValidateMediaURL(mediaURL, strings.ToLower(mediaType)); err != nil {
		return "", err
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return "", err
	}

	// Build container using builder pattern
	builder := NewContainerBuilder().
		SetMediaType(strings.ToUpper(mediaType)).
		SetIsCarouselItem(true).
		SetAltText(altText)

	// Set the appropriate URL parameter based on media type
	switch strings.ToUpper(mediaType) {
	case MediaTypeImage:
		builder.SetImageURL(mediaURL)
	case MediaTypeVideo:
		builder.SetVideoURL(mediaURL)
	default:
		return "", NewValidationError(400, "Invalid media type", "Media type must be IMAGE or VIDEO", "media_type")
	}

	containerID, err := c.createContainer(ctx, builder.Build())
	if err != nil {
		return "", err
	}

	return ConvertToContainerID(containerID), nil
}

// createTextContainer creates a container for text content
func (c *Client) createTextContainer(ctx context.Context, content *TextPostContent) (string, error) {
	builder := NewContainerBuilder().
		SetMediaType(MediaTypeText).
		SetText(content.Text).
		SetLinkAttachment(content.LinkAttachment).
		SetPollAttachment(content.PollAttachment).
		SetReplyControl(content.ReplyControl).
		SetReplyTo(content.ReplyTo).
		SetTopicTag(content.TopicTag).
		SetAllowlistedCountryCodes(content.AllowlistedCountryCodes).
		SetLocationID(content.LocationID).
		SetTextEntities(content.TextEntities).
		SetTextAttachment(content.TextAttachment)

	// Add quoted post ID if this is a quote post
	if content.QuotedPostID != "" {
		builder.SetQuotePostID(content.QuotedPostID)
	}

	return c.createContainer(ctx, builder.Build())
}

// createImageContainer creates a container for image content
func (c *Client) createImageContainer(ctx context.Context, content *ImagePostContent) (string, error) {
	builder := NewContainerBuilder().
		SetMediaType(MediaTypeImage).
		SetImageURL(content.ImageURL).
		SetText(content.Text).
		SetAltText(content.AltText).
		SetReplyControl(content.ReplyControl).
		SetReplyTo(content.ReplyTo).
		SetTopicTag(content.TopicTag).
		SetAllowlistedCountryCodes(content.AllowlistedCountryCodes).
		SetLocationID(content.LocationID).
		SetTextEntities(content.TextEntities).
		SetIsSpoilerMedia(content.IsSpoilerMedia)

	// Add quoted post ID if this is a quote post
	if content.QuotedPostID != "" {
		builder.SetQuotePostID(content.QuotedPostID)
	}

	return c.createContainer(ctx, builder.Build())
}

// createVideoContainer creates a container for video content
func (c *Client) createVideoContainer(ctx context.Context, content *VideoPostContent) (string, error) {
	builder := NewContainerBuilder().
		SetMediaType(MediaTypeVideo).
		SetVideoURL(content.VideoURL).
		SetText(content.Text).
		SetAltText(content.AltText).
		SetReplyControl(content.ReplyControl).
		SetReplyTo(content.ReplyTo).
		SetTopicTag(content.TopicTag).
		SetAllowlistedCountryCodes(content.AllowlistedCountryCodes).
		SetLocationID(content.LocationID).
		SetTextEntities(content.TextEntities).
		SetIsSpoilerMedia(content.IsSpoilerMedia)

	// Add quoted post ID if this is a quote post
	if content.QuotedPostID != "" {
		builder.SetQuotePostID(content.QuotedPostID)
	}

	containerID, err := c.createContainer(ctx, builder.Build())
	if err != nil {
		return "", err
	}

	// Videos need processing time - wait for container to be ready
	if c.config.Logger != nil {
		c.config.Logger.Info("Video container created, waiting for processing", "container_id", containerID)
	}

	// Wait for video processing to complete
	if err := c.waitForContainerProcessing(ctx, containerID); err != nil {
		return "", fmt.Errorf("video processing failed: %w", err)
	}

	return containerID, nil
}

// createCarouselContainer creates a container for carousel content
func (c *Client) createCarouselContainer(ctx context.Context, content *CarouselPostContent) (string, error) {
	builder := NewContainerBuilder().
		SetMediaType(MediaTypeCarousel).
		SetText(content.Text).
		SetChildren(content.Children).
		SetReplyControl(content.ReplyControl).
		SetReplyTo(content.ReplyTo).
		SetTopicTag(content.TopicTag).
		SetAllowlistedCountryCodes(content.AllowlistedCountryCodes).
		SetLocationID(content.LocationID).
		SetTextEntities(content.TextEntities).
		SetIsSpoilerMedia(content.IsSpoilerMedia)

	// Add quoted post ID if this is a quote post
	if content.QuotedPostID != "" {
		builder.SetQuotePostID(content.QuotedPostID)
	}

	return c.createContainer(ctx, builder.Build())
}

// createAndPublishTextPostDirectly creates and publishes a text post directly when auto_publish_text is true
func (c *Client) createAndPublishTextPostDirectly(ctx context.Context, content *TextPostContent) (*Post, error) {
	builder := NewContainerBuilder().
		SetMediaType(MediaTypeText).
		SetText(content.Text).
		SetAutoPublishText(true).
		SetLinkAttachment(content.LinkAttachment).
		SetPollAttachment(content.PollAttachment).
		SetReplyControl(content.ReplyControl).
		SetReplyTo(content.ReplyTo).
		SetTopicTag(content.TopicTag).
		SetAllowlistedCountryCodes(content.AllowlistedCountryCodes).
		SetLocationID(content.LocationID).
		SetTextEntities(content.TextEntities).
		SetTextAttachment(content.TextAttachment)

	// Get user ID from token info
	userID := c.getUserID()
	if userID == "" {
		return nil, NewAuthenticationError(401, "User ID not available", "Cannot determine user ID from token")
	}

	// Make API call to create and publish post directly
	path := fmt.Sprintf("/%s/threads", userID)
	resp, err := c.httpClient.POST(path, builder.Build(), c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response - when auto_publish_text is true, the API returns the post ID directly
	var post Post
	if err := safeJSONUnmarshal(resp.Body, &post, "direct publish response", resp.RequestID); err != nil {
		return nil, err
	}

	// Validate that we got a valid post ID
	if post.ID == "" {
		return nil, NewAPIError(resp.StatusCode, "Post ID not returned", "API response missing post ID", resp.RequestID)
	}

	// Fetch the created post details
	return c.GetPost(ctx, ConvertToPostID(post.ID))
}

// createContainer is a helper method to create containers with given parameters
func (c *Client) createContainer(_ context.Context, params url.Values) (string, error) {
	// Get user ID from token info
	userID := c.getUserID()
	if userID == "" {
		return "", NewAuthenticationError(401, "User ID not available", "Cannot determine user ID from token")
	}

	// Make API call to create container
	path := fmt.Sprintf("/%s/threads", userID)
	resp, err := c.httpClient.POST(path, params, c.getAccessTokenSafe())
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", c.handleAPIError(resp)
	}

	// Parse response to get container ID
	var containerResp struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(resp.Body, &containerResp); err != nil {
		return "", NewAPIError(resp.StatusCode, "Failed to parse container response", err.Error(), resp.RequestID)
	}

	if containerResp.ID == "" {
		return "", NewAPIError(resp.StatusCode, "Container ID not returned", "API response missing container ID", resp.RequestID)
	}

	return containerResp.ID, nil
}

// publishContainer publishes a created container
func (c *Client) publishContainer(ctx context.Context, containerID string) (*Post, error) {
	if containerID == "" {
		return nil, NewValidationError(400, ErrEmptyContainerID, "Cannot publish without container ID", "container_id")
	}

	// Get user ID from token info
	userID := c.getUserID()
	if userID == "" {
		return nil, NewAuthenticationError(401, "User ID not available", "Cannot determine user ID from token")
	}

	// Build request parameters
	params := url.Values{
		"creation_id": {containerID},
	}

	// Make API call to publish container
	path := fmt.Sprintf("/%s/threads_publish", userID)
	resp, err := c.httpClient.POST(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response to get post ID
	var publishResp struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(resp.Body, &publishResp); err != nil {
		return nil, NewAPIError(resp.StatusCode, "Failed to parse publish response", err.Error(), resp.RequestID)
	}

	if publishResp.ID == "" {
		return nil, NewAPIError(resp.StatusCode, "Post ID not returned", "API response missing post ID", resp.RequestID)
	}

	// Fetch the created post details
	return c.GetPost(ctx, ConvertToPostID(publishResp.ID))
}

// waitForContainerProcessing waits for a video container to finish processing
func (c *Client) waitForContainerProcessing(ctx context.Context, containerID string) error {
	if c.config.Logger != nil {
		c.config.Logger.Info("Waiting for video container processing", "container_id", containerID)
	}

	for attempt := 1; attempt <= VideoProcessingMaxAttempts; attempt++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check container status
		params := url.Values{
			"fields": {ContainerStatusFields},
		}

		path := fmt.Sprintf("/%s", containerID)
		resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())

		if err != nil {
			if c.config.Logger != nil {
				c.config.Logger.Warn("Failed to check container status", "container_id", containerID, "attempt", attempt, "error", err.Error())
			}

			if attempt < VideoProcessingMaxAttempts {
				time.Sleep(VideoProcessingPollInterval)
				continue
			}
			return fmt.Errorf("container status check failed after %d attempts: %w", VideoProcessingMaxAttempts, err)
		}

		if resp.StatusCode != 200 {
			if c.config.Logger != nil {
				c.config.Logger.Warn("Container status check returned non-200", "container_id", containerID, "status_code", resp.StatusCode, "attempt", attempt)
			}

			if attempt < VideoProcessingMaxAttempts {
				time.Sleep(VideoProcessingPollInterval)
				continue
			}
			return NewAPIError(resp.StatusCode, "Container status check failed", string(resp.Body), "")
		}

		// Parse response to check status
		var statusResp struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			ErrorMessage string `json:"error_message,omitempty"`
		}

		if err := json.Unmarshal(resp.Body, &statusResp); err != nil {
			if c.config.Logger != nil {
				c.config.Logger.Warn("Failed to parse container status response", "container_id", containerID, "attempt", attempt, "error", err.Error())
			}

			if attempt < VideoProcessingMaxAttempts {
				time.Sleep(VideoProcessingPollInterval)
				continue
			}
			return fmt.Errorf("failed to parse container status response: %w", err)
		}

		if c.config.Logger != nil {
			c.config.Logger.Info("Container status check", "container_id", containerID, "status", statusResp.Status, "attempt", attempt)
		}

		// Check status
		switch statusResp.Status {
		case ContainerStatusFinished:
			if c.config.Logger != nil {
				c.config.Logger.Info("Video container processing completed", "container_id", containerID, "attempts", attempt)
			}
			return nil

		case ContainerStatusPublished:
			if c.config.Logger != nil {
				c.config.Logger.Info("Video container already published", "container_id", containerID, "attempts", attempt)
			}
			return nil

		case ContainerStatusInProgress:
			if c.config.Logger != nil {
				c.config.Logger.Info("Video container still processing", "container_id", containerID, "attempt", attempt)
			}
			if attempt < VideoProcessingMaxAttempts {
				time.Sleep(VideoProcessingPollInterval)
				continue
			}
			return NewAPIError(408, "Video processing timeout", fmt.Sprintf("Container %s is still processing after %d minutes", containerID, VideoProcessingMaxAttempts), "")

		case ContainerStatusError:
			errorMsg := "Unknown error"
			if statusResp.ErrorMessage != "" {
				errorMsg = statusResp.ErrorMessage
			}
			return NewAPIError(500, "Video processing failed", fmt.Sprintf("Container %s failed to process: %s", containerID, errorMsg), "")

		case ContainerStatusExpired:
			return NewAPIError(410, "Container expired", fmt.Sprintf("Container %s was not published within 24 hours and has expired", containerID), "")

		default:
			// Unknown status, continue waiting
			if c.config.Logger != nil {
				c.config.Logger.Warn("Unknown container status", "container_id", containerID, "status", statusResp.Status, "attempt", attempt)
			}
			if attempt < VideoProcessingMaxAttempts {
				time.Sleep(VideoProcessingPollInterval)
				continue
			}
			return NewAPIError(500, "Unknown container status", fmt.Sprintf("Container %s has unknown status: %s", containerID, statusResp.Status), "")
		}
	}

	// This should never be reached due to the logic above, but just in case
	return NewAPIError(408, "Video processing timeout", fmt.Sprintf("Container %s processing timed out after %d attempts", containerID, VideoProcessingMaxAttempts), "")
}

// GetContainerStatus retrieves the status of a media container
// This is useful for checking if a video or image container has finished processing
// before attempting to publish it. Returns container status information including:
// - ID: The container ID
// - Status: Current status (IN_PROGRESS, FINISHED, PUBLISHED, ERROR, EXPIRED)
// - ErrorMessage: Error details if status is ERROR
func (c *Client) GetContainerStatus(ctx context.Context, containerID ContainerID) (*ContainerStatus, error) {
	if !containerID.Valid() {
		return nil, NewValidationError(400, ErrEmptyContainerID, "Cannot check status without container ID", "container_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build request parameters with container status fields
	params := url.Values{
		"fields": {ContainerStatusFields},
	}

	// Make API call to get container status
	path := fmt.Sprintf("/%s", containerID.String())
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var status ContainerStatus
	if err := safeJSONUnmarshal(resp.Body, &status, "container status response", resp.RequestID); err != nil {
		return nil, err
	}

	// Validate response
	if status.ID == "" {
		return nil, NewAPIError(resp.StatusCode, "Container ID not returned", "API response missing container ID", resp.RequestID)
	}

	if status.Status == "" {
		return nil, NewAPIError(resp.StatusCode, "Container status not returned", "API response missing container status", resp.RequestID)
	}

	return &status, nil
}

// waitForContainerReady polls the container status until it's ready to be published
// Returns an error if the container fails or times out
func (c *Client) waitForContainerReady(ctx context.Context, containerID ContainerID, maxAttempts int, pollInterval time.Duration) error {
	for attempt := 0; attempt < maxAttempts; attempt++ {
		status, err := c.GetContainerStatus(ctx, containerID)
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}

		switch status.Status {
		case ContainerStatusFinished:
			// Container is ready to be published
			return nil
		case ContainerStatusError:
			if status.ErrorMessage != "" {
				return fmt.Errorf("container processing failed: %s", status.ErrorMessage)
			}
			return fmt.Errorf("container processing failed with error status")
		case ContainerStatusExpired:
			return fmt.Errorf("container expired before it could be published")
		case ContainerStatusInProgress, ContainerStatusPublished:
			// Still processing or already published, wait and retry
			time.Sleep(pollInterval)
			continue
		default:
			// Unknown status, wait and retry
			time.Sleep(pollInterval)
			continue
		}
	}

	return fmt.Errorf("timeout waiting for container to be ready after %d attempts", maxAttempts)
}
