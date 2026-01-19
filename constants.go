package threads

import "time"

// Constants and limits for the Threads API.
// These values are based on the official API documentation and may change.
// Always refer to the latest Threads API documentation for current limits.

// API Limits
const (
	// Text limits
	MaxTextLength           = 500   // Maximum characters for post text
	MaxTextAttachmentLength = 10000 // Maximum characters for text attachment plaintext
	MaxTextEntities         = 10    // Maximum text spoiler entities per post
	MaxLinks                = 5     // Maximum number of links in a post

	// Pagination limits
	MaxPostsPerRequest = 100 // Maximum posts per API request
	DefaultPostsLimit  = 25  // Default number of posts if not specified

	// Carousel limits
	MinCarouselItems = 2  // Minimum items in a carousel
	MaxCarouselItems = 20 // Maximum items in a carousel

	// Reply processing
	ReplyPublishDelay = 10 * time.Second // Recommended delay before publishing reply

	// Search constraints
	MinSearchTimestamp = 1688540400 // Minimum timestamp for search queries (July 5, 2023)

	// Library version
	Version = "1.1.0"

	// HTTP client defaults
	DefaultHTTPTimeout = 30 * time.Second // Default HTTP request timeout
	DefaultUserAgent   = "threads-go/" + Version
)

// API Endpoints
const (
	BaseAPIURL = "https://graph.threads.net"
)

// Field Sets for API requests
const (
	// Post fields (is_verified and profile_picture_url added December 16, 2025 for replies/mentions)
	PostExtendedFields = "id,media_product_type,media_type,media_url,permalink,owner,username,text,timestamp,shortcode,thumbnail_url,children,is_quote_post,alt_text,link_attachment_url,has_replies,reply_audience,quoted_post,reposted_post,gif_url,is_verified,profile_picture_url"

	// Ghost Post fields
	GhostPostFields = "id,media_product_type,media_type,media_url,permalink,owner,username,text,timestamp,shortcode,thumbnail_url,ghost_post_status,ghost_post_expiration_timestamp"

	// User fields
	UserProfileFields = "id,username,name,threads_profile_picture_url,threads_biography,is_verified"

	// Reply fields (includes additional reply-specific fields)
	// is_verified and profile_picture_url added December 16, 2025 (only available on direct replies for conversations)
	ReplyFields = "id,media_product_type,media_type,media_url,permalink,username,text,timestamp,shortcode,thumbnail_url,children,is_quote_post,has_replies,root_post,replied_to,is_reply,is_reply_owned_by_me,reply_audience,quoted_post,reposted_post,gif_url,alt_text,hide_status,topic_tag,is_verified,profile_picture_url"

	// Container status fields
	ContainerStatusFields = "id,status,error_message"

	// Location fields
	LocationFields = "id,address,name,city,country,latitude,longitude,postal_code"

	// Publishing limit fields
	PublishingLimitFields = "quota_usage,config,reply_quota_usage,reply_config,delete_quota_usage,delete_config,location_search_quota_usage,location_search_config"
)

// Container Status values
const (
	ContainerStatusInProgress = "IN_PROGRESS"
	ContainerStatusFinished   = "FINISHED"
	ContainerStatusPublished  = "PUBLISHED"
	ContainerStatusError      = "ERROR"
	ContainerStatusExpired    = "EXPIRED"

	// Container polling configuration
	DefaultContainerPollMaxAttempts = 30              // Maximum number of polling attempts
	DefaultContainerPollInterval    = 1 * time.Second // Interval between polling attempts
)

// Media Types
const (
	MediaTypeText     = "TEXT"
	MediaTypeImage    = "IMAGE"
	MediaTypeVideo    = "VIDEO"
	MediaTypeCarousel = "CAROUSEL"
)

// Error messages
const (
	ErrEmptyPostID      = "Post ID is required"
	ErrEmptyUserID      = "User ID is required"
	ErrEmptyContainerID = "Container ID is required"
	ErrEmptySearchQuery = "Search query is required"
)

// API Error Codes returned by the Threads API
const (
	// ErrCodeLinkLimitExceeded is returned when a post contains more than 5 unique links.
	// This error occurs during media container creation (POST /{threads-user-id}/threads).
	// Added December 22, 2025. Reduce the number of unique links to 5 or fewer to resolve.
	ErrCodeLinkLimitExceeded = "THREADS_API__LINK_LIMIT_EXCEEDED"
)
