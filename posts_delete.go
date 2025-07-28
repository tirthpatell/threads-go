package threads

import (
	"context"
	"encoding/json"
	"fmt"
)

// DeletePost deletes a specific post by ID with proper validation and confirmation
func (c *Client) DeletePost(ctx context.Context, postID PostID) error {
	if !postID.Valid() {
		return NewValidationError(400, ErrEmptyPostID, "Cannot delete post without ID", "post_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return err
	}

	// First, validate that the post exists and is owned by the authenticated user
	if err := c.validatePostOwnership(ctx, postID); err != nil {
		return err
	}

	// Make API call to delete post
	path := fmt.Sprintf("/%s", postID.String())
	resp, err := c.httpClient.DELETE(path, c.getAccessTokenSafe())
	if err != nil {
		return err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return NewValidationError(404, "Post not found", fmt.Sprintf("Post with ID %s does not exist or is not accessible", postID.String()), "post_id")
	}

	if resp.StatusCode == 403 {
		return NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot delete post %s - insufficient permissions or not the post owner", postID.String()))
	}

	if resp.StatusCode != 200 {
		return c.handleAPIError(resp)
	}

	// Parse response to confirm deletion
	var deleteResp struct {
		Success bool `json:"success"`
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

	return nil
}

// DeletePostWithConfirmation deletes a post with an additional confirmation step
func (c *Client) DeletePostWithConfirmation(ctx context.Context, postID PostID, confirmationCallback func(post *Post) bool) error {
	if !postID.Valid() {
		return NewValidationError(400, ErrEmptyPostID, "Cannot delete post without ID", "post_id")
	}

	if confirmationCallback == nil {
		return NewValidationError(400, "Confirmation callback is required", "Must provide confirmation callback", "confirmation_callback")
	}

	// Get the post first to show details for confirmation
	post, err := c.GetPost(ctx, postID)
	if err != nil {
		return err
	}

	// Call confirmation callback
	if !confirmationCallback(post) {
		return NewValidationError(400, "Deletion cancelled", "User cancelled the deletion", "confirmation")
	}

	// Proceed with deletion
	return c.DeletePost(ctx, postID)
}

// validatePostOwnership validates that the post exists and is owned by the authenticated user
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

	// Check if the post belongs to the authenticated user
	if post.Username != me.Username {
		return NewAuthenticationError(403, "Cannot delete post", fmt.Sprintf("Post %s belongs to user %s, not %s", postID.String(), post.Username, me.Username))
	}

	return nil
}
