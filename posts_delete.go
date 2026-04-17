package threads

import (
	"context"
	"encoding/json"
	"fmt"
)

// DeletePost deletes a specific post by ID with proper validation and confirmation.
// Returns the deleted post ID as reported by the API. If the API response cannot
// be parsed, the returned ID will be an empty string; a non-nil error is only
// returned when the HTTP request itself fails or the server returns a non-200 status.
// Note: The API enforces a limit of 100 deletes per 24-hour window. Check
// PublishingLimits.DeleteQuotaUsage via GetPublishingLimits to monitor usage.
// The threads_delete permission scope is required for this endpoint.
func (c *Client) DeletePost(ctx context.Context, postID PostID) (string, error) {
	if !postID.Valid() {
		return "", NewValidationError(400, ErrEmptyPostID, "Cannot delete post without ID", "post_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return "", err
	}

	// First, validate that the post exists and is owned by the authenticated user
	if err := c.validatePostOwnership(ctx, postID); err != nil {
		return "", err
	}

	// Make API call to delete post
	path := fmt.Sprintf("/%s", postID.String())
	resp, err := c.httpClient.DELETE(path, c.getAccessTokenSafe())
	if err != nil {
		return "", err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return "", NewValidationError(404, "Post not found", fmt.Sprintf("Post with ID %s does not exist or is not accessible", postID.String()), "post_id")
	}

	if resp.StatusCode == 403 {
		return "", NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot delete post %s - insufficient permissions or not the post owner", postID.String()))
	}

	if resp.StatusCode != 200 {
		return "", c.handleAPIError(resp)
	}

	// Parse response to confirm deletion
	var deleteResp struct {
		Success   bool   `json:"success"`
		DeletedID string `json:"deleted_id"`
	}

	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &deleteResp); err != nil {
			// If we can't parse the response but got 200, assume success
			if c.config.Logger != nil {
				c.config.Logger.Warn("Could not parse delete response, but got 200 status", "post_id", postID.String())
			}
		}
	}

	// Log successful deletion if logger is available
	if c.config.Logger != nil {
		c.config.Logger.Info("Successfully deleted post", "post_id", postID.String())
	}

	return deleteResp.DeletedID, nil
}

// DeletePostWithConfirmation deletes a post with an additional confirmation step
func (c *Client) DeletePostWithConfirmation(ctx context.Context, postID PostID, confirmationCallback func(post *Post) bool) (string, error) {
	if !postID.Valid() {
		return "", NewValidationError(400, ErrEmptyPostID, "Cannot delete post without ID", "post_id")
	}

	if confirmationCallback == nil {
		return "", NewValidationError(400, "Confirmation callback is required", "Must provide confirmation callback", "confirmation_callback")
	}

	// Get the post first to show details for confirmation
	post, err := c.GetPost(ctx, postID)
	if err != nil {
		return "", err
	}

	// Call confirmation callback
	if !confirmationCallback(post) {
		return "", NewValidationError(400, "Deletion cancelled", "User cancelled the deletion", "confirmation")
	}

	// Proceed with deletion
	return c.DeletePost(ctx, postID)
}

// validatePostOwnership validates that the post exists and is owned by the
// authenticated user. The check prefers the stable numeric owner ID
// (post.Owner.ID vs me.ID) and only falls back to username comparison when
// the API response doesn't include an owner object. Empty identifiers on
// either side are treated as "cannot verify" and cause the check to fail
// closed — otherwise two empty strings would compare equal and silently
// authorise deletion of a post whose ownership we could not actually
// determine.
func (c *Client) validatePostOwnership(ctx context.Context, postID PostID) error {
	// Get the post to check ownership
	post, err := c.GetPost(ctx, postID)
	if err != nil {
		return err
	}

	// Get authenticated user info
	me, err := c.GetMe(ctx)
	if err != nil {
		return NewAuthenticationError(401, "Cannot verify post ownership", "Failed to get authenticated user information")
	}

	// Prefer owner ID when available — it's the canonical identifier and
	// doesn't collide with username handle changes.
	if post.Owner != nil && post.Owner.ID != "" && me.ID != "" {
		if post.Owner.ID != me.ID {
			return NewAuthenticationError(403, "Cannot delete post", fmt.Sprintf("Post %s is owned by user ID %s, not %s", postID.String(), post.Owner.ID, me.ID))
		}
		return nil
	}

	// Fall back to username when owner ID is unavailable.
	if post.Username == "" || me.Username == "" {
		return NewAuthenticationError(403, "Cannot verify post ownership", fmt.Sprintf("Post %s has no comparable owner identifier (post.Owner=%v, post.Username=%q, me.ID=%q, me.Username=%q)", postID.String(), post.Owner, post.Username, me.ID, me.Username))
	}
	if post.Username != me.Username {
		return NewAuthenticationError(403, "Cannot delete post", fmt.Sprintf("Post %s belongs to user %s, not %s", postID.String(), post.Username, me.Username))
	}

	return nil
}
