package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// buildRepliesParams builds query parameters for replies and conversation requests
func buildRepliesParams(opts *RepliesOptions, maxLimit int, limitDescription string) (url.Values, error) {
	params := url.Values{
		"fields": {ReplyFields},
	}

	if opts != nil {
		if opts.Limit > 0 {
			if opts.Limit > maxLimit {
				return nil, NewValidationError(400, "Limit too large", fmt.Sprintf("Maximum limit is %d %s", maxLimit, limitDescription), "limit")
			}
			params.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Before != "" {
			params.Set("before", opts.Before)
		}
		if opts.After != "" {
			params.Set("after", opts.After)
		}
		if opts.Reverse != nil {
			params.Set("reverse", fmt.Sprintf("%t", *opts.Reverse))
		}
	}

	return params, nil
}

// fetchRepliesData makes the API call and handles common error cases
func (c *Client) fetchRepliesData(path string, params url.Values, postID PostID, dataType string) (*RepliesResponse, error) {
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "Post not found", fmt.Sprintf("Post with ID %s does not exist or is not accessible", postID.String()), "post_id")
	}

	if resp.StatusCode == 403 {
		return nil, NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot access %s for post %s - insufficient permissions", dataType, postID.String()))
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var repliesResp RepliesResponse
	if err := safeJSONUnmarshal(resp.Body, &repliesResp, dataType, resp.RequestID); err != nil {
		return nil, err
	}

	return &repliesResp, nil
}

// GetReplies retrieves replies to a specific post with pagination support
func (c *Client) GetReplies(ctx context.Context, postID PostID, opts *RepliesOptions) (*RepliesResponse, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "Cannot retrieve replies without post ID", "post_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters
	params, err := buildRepliesParams(opts, 100, "replies per request")
	if err != nil {
		return nil, err
	}

	// Make API call to get post replies
	path := fmt.Sprintf("/%s/replies", postID.String())
	return c.fetchRepliesData(path, params, postID, "post replies")
}

// GetConversation retrieves a flattened conversation thread for a specific post
func (c *Client) GetConversation(ctx context.Context, postID PostID, opts *RepliesOptions) (*RepliesResponse, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "Cannot retrieve conversation without post ID", "post_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters
	params, err := buildRepliesParams(opts, 100, "posts per request")
	if err != nil {
		return nil, err
	}

	// Make API call to get conversation
	path := fmt.Sprintf("/%s/conversation", postID.String())
	return c.fetchRepliesData(path, params, postID, "conversation")
}

// GetPendingReplies retrieves pending replies for a post with reply approvals enabled
func (c *Client) GetPendingReplies(ctx context.Context, postID PostID, opts *PendingRepliesOptions) (*RepliesResponse, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "Cannot retrieve pending replies without post ID", "post_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters
	params := url.Values{
		"fields": {ReplyFields},
	}

	if opts != nil {
		if opts.Limit > 0 {
			if opts.Limit > 100 {
				return nil, NewValidationError(400, "Limit too large", "Maximum limit is 100 pending replies per request", "limit")
			}
			params.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Before != "" {
			params.Set("before", opts.Before)
		}
		if opts.After != "" {
			params.Set("after", opts.After)
		}
		if opts.Reverse != nil {
			params.Set("reverse", fmt.Sprintf("%t", *opts.Reverse))
		}
		if opts.ApprovalStatus != "" {
			if opts.ApprovalStatus != ApprovalStatusPending && opts.ApprovalStatus != ApprovalStatusIgnored {
				return nil, NewValidationError(400, "Invalid approval status", "Approval status must be 'pending' or 'ignored'", "approval_status")
			}
			params.Set("approval_status", string(opts.ApprovalStatus))
		}
	}

	path := fmt.Sprintf("/%s/pending_replies", postID.String())
	return c.fetchRepliesData(path, params, postID, "pending replies")
}

// ApprovePendingReply approves a pending reply, making it publicly visible
func (c *Client) ApprovePendingReply(ctx context.Context, replyID PostID) error {
	return c.managePendingReply(ctx, replyID, true)
}

// IgnorePendingReply ignores a pending reply (it can still be approved later)
func (c *Client) IgnorePendingReply(ctx context.Context, replyID PostID) error {
	return c.managePendingReply(ctx, replyID, false)
}

// managePendingReply handles approving or ignoring a pending reply
func (c *Client) managePendingReply(ctx context.Context, replyID PostID, approve bool) error {
	action := "approve"
	if !approve {
		action = "ignore"
	}

	if !replyID.Valid() {
		return NewValidationError(400, "Reply ID is required", fmt.Sprintf("Cannot %s reply without ID", action), "reply_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return err
	}

	params := url.Values{
		"approve": {fmt.Sprintf("%t", approve)},
	}

	path := fmt.Sprintf("/%s/manage_pending_reply", replyID.String())
	resp, err := c.httpClient.POST(path, params, c.getAccessTokenSafe())
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		return NewValidationError(404, "Reply not found", fmt.Sprintf("Reply with ID %s does not exist or is not a pending reply", replyID.String()), "reply_id")
	}

	if resp.StatusCode == 403 {
		return NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot %s reply %s - insufficient permissions or not the post owner", action, replyID.String()))
	}

	if resp.StatusCode != 200 {
		return c.handleAPIError(resp)
	}

	var manageResp struct {
		Success bool `json:"success"`
	}

	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &manageResp); err != nil {
			if c.config.Logger != nil {
				c.config.Logger.Warn(fmt.Sprintf("Could not parse %s reply response, but got 200 status", action), "reply_id", replyID.String())
			}
		}
	}

	if c.config.Logger != nil {
		c.config.Logger.Info(fmt.Sprintf("Successfully %sd pending reply", action), "reply_id", replyID.String())
	}

	return nil
}

// manageReplyVisibility handles hiding and unhiding replies
func (c *Client) manageReplyVisibility(ctx context.Context, replyID PostID, hide bool) error {
	action := "hide"
	if !hide {
		action = "unhide"
	}

	if !replyID.Valid() {
		return NewValidationError(400, "Reply ID is required", fmt.Sprintf("Cannot %s reply without ID", action), "reply_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return err
	}

	// Build request parameters
	params := url.Values{
		"hide": {fmt.Sprintf("%t", hide)},
	}

	// Make API call to manage reply visibility
	path := fmt.Sprintf("/%s/manage_reply", replyID.String())
	resp, err := c.httpClient.POST(path, params, c.getAccessTokenSafe())
	if err != nil {
		return err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return NewValidationError(404, "Reply not found", fmt.Sprintf("Reply with ID %s does not exist or is not accessible", replyID.String()), "reply_id")
	}

	if resp.StatusCode == 403 {
		return NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot %s reply %s - insufficient permissions or not the post owner", action, replyID.String()))
	}

	if resp.StatusCode != 200 {
		return c.handleAPIError(resp)
	}

	// Parse response to confirm action
	var manageResp struct {
		Success bool `json:"success"`
	}

	if len(resp.Body) > 0 {
		if err := json.Unmarshal(resp.Body, &manageResp); err != nil {
			// If we can't parse the response but got 200, assume success
			if c.config.Logger != nil {
				c.config.Logger.Warn(fmt.Sprintf("Could not parse %s reply response, but got 200 status", action), "reply_id", replyID.String())
			}
		}
	}

	// Log successful action if logger is available
	if c.config.Logger != nil {
		c.config.Logger.Info(fmt.Sprintf("Successfully %sd reply", action), "reply_id", replyID.String())
	}

	return nil
}

// HideReply hides a specific reply for moderation purposes
func (c *Client) HideReply(ctx context.Context, replyID PostID) error {
	return c.manageReplyVisibility(ctx, replyID, true)
}

// UnhideReply unhides a specific reply that was previously hidden
func (c *Client) UnhideReply(ctx context.Context, replyID PostID) error {
	return c.manageReplyVisibility(ctx, replyID, false)
}
