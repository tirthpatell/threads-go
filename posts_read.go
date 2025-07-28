package threads

import (
	"context"
	"fmt"
	"net/url"
)

// GetPost retrieves a specific post by ID with all available fields
func (c *Client) GetPost(ctx context.Context, postID PostID) (*Post, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "Cannot retrieve post without ID", "post_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters with extended fields for comprehensive data
	params := url.Values{
		"fields": {PostExtendedFields},
	}

	// Make API call to get post
	path := fmt.Sprintf("/%s", postID.String())
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases for non-existent posts
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "Post not found", fmt.Sprintf("Post with ID %s does not exist or is not accessible", postID.String()), "post_id")
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var post Post
	if err := safeJSONUnmarshal(resp.Body, &post, "post response", resp.RequestID); err != nil {
		return nil, err
	}

	return &post, nil
}

// GetUserPosts retrieves posts from a specific user with pagination support
func (c *Client) GetUserPosts(ctx context.Context, userID UserID, opts *PaginationOptions) (*PostsResponse, error) {
	// Convert PaginationOptions to PostsOptions for backward compatibility
	var postsOpts *PostsOptions
	if opts != nil {
		postsOpts = &PostsOptions{
			Limit:  opts.Limit,
			Before: opts.Before,
			After:  opts.After,
		}
	}
	return c.GetUserPostsWithOptions(ctx, userID, postsOpts)
}

// GetUserPostsWithOptions retrieves posts from a specific user with enhanced options
func (c *Client) GetUserPostsWithOptions(ctx context.Context, userID UserID, opts *PostsOptions) (*PostsResponse, error) {
	if !userID.Valid() {
		return nil, NewValidationError(400, ErrEmptyUserID, "Cannot retrieve posts without user ID", "user_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Validate pagination options
	validator := NewValidator()
	if opts != nil {
		paginationOpts := &PaginationOptions{
			Limit:  opts.Limit,
			Before: opts.Before,
			After:  opts.After,
		}
		if err := validator.ValidatePaginationOptions(paginationOpts); err != nil {
			return nil, err
		}
	}

	// Build query parameters with enhanced fields from API documentation
	params := url.Values{
		"fields": {PostExtendedFields},
	}

	// Add pagination and filtering options if provided
	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Before != "" {
			params.Set("before", opts.Before)
		}
		if opts.After != "" {
			params.Set("after", opts.After)
		}
		if opts.Since > 0 {
			params.Set("since", fmt.Sprintf("%d", opts.Since))
		}
		if opts.Until > 0 {
			params.Set("until", fmt.Sprintf("%d", opts.Until))
		}
	}

	// Make API call to get user posts
	path := fmt.Sprintf("/%s/threads", userID.String())
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases for non-existent users
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "User not found", fmt.Sprintf("User with ID %s does not exist or is not accessible", userID.String()), "user_id")
	}

	// Handle permission errors
	if resp.StatusCode == 403 {
		return nil, NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot access posts for user %s - insufficient permissions", userID.String()))
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var postsResp PostsResponse
	if err := safeJSONUnmarshal(resp.Body, &postsResp, "posts response", resp.RequestID); err != nil {
		return nil, err
	}

	return &postsResp, nil
}

// GetUserMentions retrieves posts where the user is mentioned
func (c *Client) GetUserMentions(ctx context.Context, userID UserID, opts *PaginationOptions) (*PostsResponse, error) {
	if !userID.Valid() {
		return nil, NewValidationError(400, ErrEmptyUserID, "Cannot retrieve mentions without user ID", "user_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Validate pagination options
	validator := NewValidator()
	if err := validator.ValidatePaginationOptions(opts); err != nil {
		return nil, err
	}

	// Build query parameters
	params := url.Values{
		"fields": {PostExtendedFields},
	}

	// Add pagination options if provided
	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Before != "" {
			params.Set("before", opts.Before)
		}
		if opts.After != "" {
			params.Set("after", opts.After)
		}
	}

	// Make API call to get user mentions
	path := fmt.Sprintf("/%s/mentions", userID.String())
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "User not found", fmt.Sprintf("User with ID %s does not exist or is not accessible", userID.String()), "user_id")
	}

	if resp.StatusCode == 403 {
		return nil, NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot access mentions for user %s - insufficient permissions", userID.String()))
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var postsResp PostsResponse
	if err := safeJSONUnmarshal(resp.Body, &postsResp, "mentions response", resp.RequestID); err != nil {
		return nil, err
	}

	return &postsResp, nil
}

// GetPublishingLimits retrieves the current API quota usage for the user
func (c *Client) GetPublishingLimits(ctx context.Context) (*PublishingLimits, error) {
	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Get user ID from token info
	userID := c.getUserID()
	if userID == "" {
		return nil, NewAuthenticationError(401, "User ID not available", "Cannot determine user ID from token")
	}

	// Build query parameters
	params := url.Values{
		"fields": {PublishingLimitFields},
	}

	// Make API call
	path := fmt.Sprintf("/%s/threads_publishing_limit", userID)
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var limitsResp struct {
		Data []PublishingLimits `json:"data"`
	}

	if err := safeJSONUnmarshal(resp.Body, &limitsResp, "publishing limits response", resp.RequestID); err != nil {
		return nil, err
	}

	if len(limitsResp.Data) == 0 {
		return nil, NewAPIError(resp.StatusCode, "No publishing limits data returned", "API response missing data", resp.RequestID)
	}

	return &limitsResp.Data[0], nil
}
