package threads

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// GetUser retrieves user profile information by user ID
func (c *Client) GetUser(ctx context.Context, userID UserID) (*User, error) {
	if !userID.Valid() {
		return nil, NewValidationError(400, ErrEmptyUserID, "Cannot retrieve user without ID", "user_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters for user fields based on Threads API documentation
	params := url.Values{
		"fields": {UserProfileFields},
	}

	// Make API call to get user
	path := fmt.Sprintf("/%s", userID.String())
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
		return nil, NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot access user %s - insufficient permissions", userID.String()))
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var apiUser struct {
		ID                       string `json:"id"`
		Username                 string `json:"username"`
		ThreadsProfilePictureURL string `json:"threads_profile_picture_url,omitempty"`
		ThreadsBiography         string `json:"threads_biography,omitempty"`
	}

	if err := safeJSONUnmarshal(resp.Body, &apiUser, "user profile", resp.RequestID); err != nil {
		return nil, err
	}

	// Convert to our User struct format
	user := &User{
		ID:            apiUser.ID,
		Username:      apiUser.Username,
		ProfilePicURL: apiUser.ThreadsProfilePictureURL,
		Biography:     apiUser.ThreadsBiography,
	}

	return user, nil
}

// GetMe retrieves the authenticated user's profile information
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Get user ID from token info
	userID := c.getUserID()
	if userID == "" {
		return nil, NewAuthenticationError(401, "User ID not available", "Cannot determine user ID from token")
	}

	// Use the standard GetUser method for consistency
	return c.GetUser(ctx, ConvertToUserID(userID))
}

// GetUserFields retrieves specific fields for a user
func (c *Client) GetUserFields(ctx context.Context, userID UserID, fields []string) (*User, error) {
	if !userID.Valid() {
		return nil, NewValidationError(400, ErrEmptyUserID, "Cannot retrieve user without ID", "user_id")
	}

	if len(fields) == 0 {
		// Default to basic fields
		fields = []string{"id", "username", "threads_profile_picture_url", "threads_biography"}
	}

	// Validate fields against allowed fields from API documentation
	allowedFields := map[string]bool{
		"id":                          true,
		"username":                    true,
		"name":                        true,
		"threads_profile_picture_url": true,
		"threads_biography":           true,
		"is_verified":                 true,
		"recently_searched_keywords":  true,
	}

	var validFields []string
	for _, field := range fields {
		if allowedFields[field] {
			validFields = append(validFields, field)
		}
	}

	if len(validFields) == 0 {
		return nil, NewValidationError(400, "No valid fields specified", "Must specify at least one valid field", "fields")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters
	params := url.Values{
		"fields": {strings.Join(validFields, ",")},
	}

	// Make API call to get user
	path := fmt.Sprintf("/%s", userID.String())
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "User not found", fmt.Sprintf("User with ID %s does not exist or is not accessible", userID.String()), "user_id")
	}

	if resp.StatusCode == 403 {
		return nil, NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot access user %s - insufficient permissions", userID.String()))
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response with all possible fields
	var apiUser struct {
		ID                       string   `json:"id"`
		Username                 string   `json:"username"`
		Name                     string   `json:"name,omitempty"`
		ThreadsProfilePictureURL string   `json:"threads_profile_picture_url,omitempty"`
		ThreadsBiography         string   `json:"threads_biography,omitempty"`
		IsVerified               bool     `json:"is_verified,omitempty"`
		RecentlySearchedKeywords []string `json:"recently_searched_keywords,omitempty"`
	}

	if err := safeJSONUnmarshal(resp.Body, &apiUser, "user response", resp.RequestID); err != nil {
		return nil, err
	}

	// Convert to our User struct format
	user := &User{
		ID:            apiUser.ID,
		Username:      apiUser.Username,
		Name:          apiUser.Name,
		ProfilePicURL: apiUser.ThreadsProfilePictureURL,
		Biography:     apiUser.ThreadsBiography,
		IsVerified:    apiUser.IsVerified,
	}

	return user, nil
}

// LookupPublicProfile looks up a public profile by username
func (c *Client) LookupPublicProfile(ctx context.Context, username string) (*PublicUser, error) {
	if strings.TrimSpace(username) == "" {
		return nil, NewValidationError(400, "Username is required", "Cannot lookup profile without username", "username")
	}

	// Remove @ symbol if present
	username = strings.TrimPrefix(username, "@")

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters
	params := url.Values{
		"username": {username},
	}

	// Make API call to lookup public profile
	path := "/profile_lookup"
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "Profile not found", fmt.Sprintf("Public profile with username %s not found", username), "username")
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var publicUser PublicUser
	if err := safeJSONUnmarshal(resp.Body, &publicUser, "public profile response", resp.RequestID); err != nil {
		return nil, err
	}

	return &publicUser, nil
}

// GetPublicProfilePosts retrieves posts from a public profile by username
func (c *Client) GetPublicProfilePosts(ctx context.Context, username string, opts *PostsOptions) (*PostsResponse, error) {
	if strings.TrimSpace(username) == "" {
		return nil, NewValidationError(400, "Username is required", "Cannot retrieve posts without username", "username")
	}

	// Remove @ symbol if present
	username = strings.TrimPrefix(username, "@")

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters with enhanced fields from API documentation
	params := url.Values{
		"username": {username},
		"fields":   {PostExtendedFields},
	}

	// Add pagination and filtering options if provided
	if opts != nil {
		if opts.Limit > 0 {
			if opts.Limit > 100 {
				return nil, NewValidationError(400, "Limit too large", "Maximum limit is 100 posts per request", "limit")
			}
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

	// Make API call to get public profile posts
	path := "/profile_posts"
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "Profile not found", fmt.Sprintf("Public profile with username %s not found", username), "username")
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var postsResp PostsResponse
	if err := safeJSONUnmarshal(resp.Body, &postsResp, "public profile posts", resp.RequestID); err != nil {
		return nil, err
	}

	return &postsResp, nil
}

// GetUserReplies retrieves all replies created by a user
func (c *Client) GetUserReplies(ctx context.Context, userID UserID, opts *PostsOptions) (*RepliesResponse, error) {
	if !userID.Valid() {
		return nil, NewValidationError(400, ErrEmptyUserID, "Cannot retrieve replies without user ID", "user_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters with reply-specific fields from API documentation
	params := url.Values{
		"fields": {ReplyFields},
	}

	// Add pagination and filtering options if provided
	if opts != nil {
		if opts.Limit > 0 {
			if opts.Limit > 100 {
				return nil, NewValidationError(400, "Limit too large", "Maximum limit is 100 replies per request", "limit")
			}
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

	// Make API call to get user replies
	path := fmt.Sprintf("/%s/replies", userID.String())
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "User not found", fmt.Sprintf("User with ID %s does not exist or is not accessible", userID.String()), "user_id")
	}

	if resp.StatusCode == 403 {
		return nil, NewAuthenticationError(403, "Access denied", fmt.Sprintf("Cannot access replies for user %s - insufficient permissions", userID.String()))
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var repliesResp RepliesResponse
	if err := safeJSONUnmarshal(resp.Body, &repliesResp, "user replies", resp.RequestID); err != nil {
		return nil, err
	}

	return &repliesResp, nil
}
