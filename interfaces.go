package threads

import (
	"context"
)

// ClientInterface is the main interface that composes all Threads API functionality
// This replaces the large monolithic interface with smaller, focused interfaces
type ClientInterface interface {
	Authenticator
	PostManager
	UserManager
	ReplyManager
	InsightsProvider
	LocationManager
	SearchProvider
	RateLimitController
}

// Authenticator handles OAuth 2.0 authentication and token management
type Authenticator interface {
	// GetAuthURL generates an authorization URL for the OAuth 2.0 flow
	GetAuthURL(scopes []string) string

	// ExchangeCodeForToken exchanges an authorization code for an access token
	ExchangeCodeForToken(ctx context.Context, code string) error

	// GetLongLivedToken converts a short-lived token to a long-lived token
	GetLongLivedToken(ctx context.Context) error

	// RefreshToken refreshes the current access token
	RefreshToken(ctx context.Context) error

	// DebugToken validates and returns information about a token
	DebugToken(ctx context.Context, inputToken string) (*DebugTokenResponse, error)

	// SetTokenFromDebugInfo sets token info from debug token response
	SetTokenFromDebugInfo(accessToken string, debugResp *DebugTokenResponse) error

	// GetTokenDebugInfo returns detailed token information
	GetTokenDebugInfo() map[string]interface{}
}

// PostManager handles post creation, retrieval, and management
type PostManager interface {
	PostCreator
	PostReader
	PostDeleter
	PostValidator
}

// PostCreator handles creation of different post types
type PostCreator interface {
	// CreateTextPost creates a new text post
	CreateTextPost(ctx context.Context, content *TextPostContent) (*Post, error)

	// CreateImagePost creates a new image post
	CreateImagePost(ctx context.Context, content *ImagePostContent) (*Post, error)

	// CreateVideoPost creates a new video post
	CreateVideoPost(ctx context.Context, content *VideoPostContent) (*Post, error)

	// CreateCarouselPost creates a carousel post with multiple media items
	CreateCarouselPost(ctx context.Context, content *CarouselPostContent) (*Post, error)

	// CreateQuotePost creates a quote post using any supported content type
	CreateQuotePost(ctx context.Context, content interface{}, quotedPostID string) (*Post, error)

	// RepostPost reposts an existing post
	RepostPost(ctx context.Context, postID PostID) (*Post, error)

	// CreateMediaContainer creates a media container for carousel items
	CreateMediaContainer(ctx context.Context, mediaType, mediaURL, altText string) (ContainerID, error)
}

// PostReader handles post retrieval operations
type PostReader interface {
	// GetPost retrieves a specific post by ID
	GetPost(ctx context.Context, postID PostID) (*Post, error)

	// GetUserPosts retrieves posts from a specific user
	GetUserPosts(ctx context.Context, userID UserID, opts *PaginationOptions) (*PostsResponse, error)

	// GetUserPostsWithOptions retrieves posts with enhanced filtering
	GetUserPostsWithOptions(ctx context.Context, userID UserID, opts *PostsOptions) (*PostsResponse, error)

	// GetUserMentions retrieves posts where the user is mentioned
	GetUserMentions(ctx context.Context, userID UserID, opts *PaginationOptions) (*PostsResponse, error)

	// GetPublishingLimits retrieves current API quota usage
	GetPublishingLimits(ctx context.Context) (*PublishingLimits, error)
}

// PostDeleter handles post deletion operations
type PostDeleter interface {
	// DeletePost deletes a specific post
	DeletePost(ctx context.Context, postID PostID) error

	// DeletePostWithConfirmation deletes a post with confirmation
	DeletePostWithConfirmation(ctx context.Context, postID PostID, confirmationCallback func(post *Post) bool) error
}

// PostValidator provides validation for post content
type PostValidator interface {
	// ValidateTextPostContent validates text post content
	ValidateTextPostContent(content *TextPostContent) error

	// ValidateImagePostContent validates image post content
	ValidateImagePostContent(content *ImagePostContent) error

	// ValidateVideoPostContent validates video post content
	ValidateVideoPostContent(content *VideoPostContent) error

	// ValidateCarouselPostContent validates carousel post content
	ValidateCarouselPostContent(content *CarouselPostContent) error

	// ValidateCarouselChildren validates carousel children containers
	ValidateCarouselChildren(childrenIDs []string) error

	// ValidateTopicTag validates a topic tag format
	ValidateTopicTag(tag string) error

	// ValidateCountryCodes validates country codes
	ValidateCountryCodes(codes []string) error
}

// UserManager handles user profile operations
type UserManager interface {
	// GetUser retrieves user profile information
	GetUser(ctx context.Context, userID UserID) (*User, error)

	// GetMe retrieves the authenticated user's profile
	GetMe(ctx context.Context) (*User, error)

	// GetUserFields retrieves specific user fields
	GetUserFields(ctx context.Context, userID UserID, fields []string) (*User, error)

	// LookupPublicProfile looks up a public profile by username
	LookupPublicProfile(ctx context.Context, username string) (*PublicUser, error)

	// GetPublicProfilePosts retrieves posts from a public profile
	GetPublicProfilePosts(ctx context.Context, username string, opts *PostsOptions) (*PostsResponse, error)
}

// ReplyManager handles reply and conversation operations
type ReplyManager interface {
	// CreateReply creates a reply to a post
	CreateReply(ctx context.Context, content *PostContent) (*Post, error)

	// ReplyToPost creates a reply to a specific post
	ReplyToPost(ctx context.Context, postID PostID, content *PostContent) (*Post, error)

	// GetReplies retrieves replies to a post
	GetReplies(ctx context.Context, postID PostID, opts *RepliesOptions) (*RepliesResponse, error)

	// GetConversation retrieves a conversation thread
	GetConversation(ctx context.Context, postID PostID, opts *RepliesOptions) (*RepliesResponse, error)

	// HideReply hides a specific reply
	HideReply(ctx context.Context, replyID PostID) error

	// UnhideReply unhides a previously hidden reply
	UnhideReply(ctx context.Context, replyID PostID) error

	// GetUserReplies retrieves all replies by a user
	GetUserReplies(ctx context.Context, userID UserID, opts *PostsOptions) (*RepliesResponse, error)
}

// InsightsProvider handles analytics and insights operations
type InsightsProvider interface {
	// GetPostInsights retrieves insights for a post
	GetPostInsights(ctx context.Context, postID PostID, metrics []string) (*InsightsResponse, error)

	// GetPostInsightsWithOptions retrieves post insights with options
	GetPostInsightsWithOptions(ctx context.Context, postID PostID, opts *PostInsightsOptions) (*InsightsResponse, error)

	// GetAccountInsights retrieves account-level insights
	GetAccountInsights(ctx context.Context, userID UserID, metrics []string, period string) (*InsightsResponse, error)

	// GetAccountInsightsWithOptions retrieves account insights with options
	GetAccountInsightsWithOptions(ctx context.Context, userID UserID, opts *AccountInsightsOptions) (*InsightsResponse, error)
}

// LocationManager handles location-related operations
type LocationManager interface {
	// SearchLocations searches for locations
	SearchLocations(ctx context.Context, query string, latitude, longitude *float64) (*LocationSearchResponse, error)

	// GetLocation retrieves location details
	GetLocation(ctx context.Context, locationID LocationID) (*Location, error)
}

// SearchProvider handles search operations
type SearchProvider interface {
	// KeywordSearch searches posts by keyword
	KeywordSearch(ctx context.Context, query string, opts *SearchOptions) (*PostsResponse, error)
}

// RateLimitController manages rate limiting behavior
type RateLimitController interface {
	// IsRateLimited returns true if currently rate limited
	IsRateLimited() bool

	// DisableRateLimiting disables rate limiting
	DisableRateLimiting()

	// EnableRateLimiting re-enables rate limiting
	EnableRateLimiting()

	// GetRateLimitStatus returns current rate limit status
	GetRateLimitStatus() RateLimitStatus

	// IsNearRateLimit checks if near rate limit threshold
	IsNearRateLimit(threshold float64) bool

	// WaitForRateLimit blocks until safe to make request
	WaitForRateLimit(ctx context.Context) error
}
