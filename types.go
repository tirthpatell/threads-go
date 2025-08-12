package threads

import (
	"encoding/json"
	"strings"
	"time"
)

// Time is a custom time type that handles Threads API timestamp format.
// For timestamp format details, see: https://developers.facebook.com/docs/threads/reference
type Time struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler for Time
func (t *Time) UnmarshalJSON(data []byte) error {
	// Remove quotes from JSON string
	str := strings.Trim(string(data), `"`)

	// Try different timestamp formats used by Threads API
	formats := []string{
		"2006-01-02T15:04:05+0000", // Threads API format
		"2006-01-02T15:04:05Z",     // ISO 8601 UTC
		time.RFC3339,               // Standard RFC3339
		"2006-01-02T15:04:05-0700", // With timezone offset
	}

	for _, format := range formats {
		if parsedTime, err := time.Parse(format, str); err == nil {
			t.Time = parsedTime
			return nil
		}
	}

	// If all formats fail, try the default time.Time unmarshalling
	return t.Time.UnmarshalJSON(data)
}

// MarshalJSON implements json.Marshaler for Time
func (t *Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(time.RFC3339))
}

// Post represents a Threads post with all its metadata and content.
// This is the primary data structure returned by most post-related API operations.
// Posts can contain text, images, videos, carousels, or be quote/reply posts.
type Post struct {
	ID                string        `json:"id"`
	Text              string        `json:"text,omitempty"`
	MediaType         string        `json:"media_type,omitempty"`
	MediaURL          string        `json:"media_url,omitempty"`
	Permalink         string        `json:"permalink"`
	Timestamp         Time          `json:"timestamp"`
	Username          string        `json:"username"`
	Owner             *PostOwner    `json:"owner,omitempty"`
	IsReply           bool          `json:"is_reply"`
	ReplyTo           string        `json:"reply_to,omitempty"`
	MediaProductType  string        `json:"media_product_type"`
	Shortcode         string        `json:"shortcode,omitempty"`
	ThumbnailURL      string        `json:"thumbnail_url,omitempty"`
	AltText           string        `json:"alt_text,omitempty"`
	Children          *ChildrenData `json:"children,omitempty"`
	IsQuotePost       bool          `json:"is_quote_post,omitempty"`
	LinkAttachmentURL string        `json:"link_attachment_url,omitempty"`
	HasReplies        bool          `json:"has_replies,omitempty"`
	ReplyAudience     string        `json:"reply_audience,omitempty"`
	QuotedPost        *Post         `json:"quoted_post,omitempty"`
	RepostedPost      *Post         `json:"reposted_post,omitempty"`
	GifURL            string        `json:"gif_url,omitempty"`
	PollAttachment    *PollResult   `json:"poll_attachment,omitempty"`
	RootPost          *Post         `json:"root_post,omitempty"`
	RepliedTo         *Post         `json:"replied_to,omitempty"`
	IsReplyOwnedByMe  bool          `json:"is_reply_owned_by_me,omitempty"`
	HideStatus        string        `json:"hide_status,omitempty"`
	TopicTag          string        `json:"topic_tag,omitempty"`
}

// User represents a Threads user profile with app-scoped data.
// The user ID and other fields are specific to your app and cannot be used
// with other apps. Contains basic profile information accessible via API.
type User struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Name           string `json:"name,omitempty"`            // Available with appropriate fields
	ProfilePicURL  string `json:"profile_pic_url,omitempty"` // Maps to threads_profile_picture_url
	Biography      string `json:"biography,omitempty"`       // Maps to threads_biography
	Website        string `json:"website,omitempty"`         // Not available in basic profile
	FollowersCount int    `json:"followers_count"`           // Not available in basic profile
	MediaCount     int    `json:"media_count"`               // Not available in basic profile
	IsVerified     bool   `json:"is_verified,omitempty"`     // Available with is_verified field
}

// PublicUser represents a public Threads user profile retrieved via the
// threads_profile_discovery scope. This contains public-facing information
// about a user that can be accessed without authentication context.
type PublicUser struct {
	Username          string `json:"username"`
	Name              string `json:"name"`
	ProfilePictureURL string `json:"profile_picture_url"`
	Biography         string `json:"biography"`
	IsVerified        bool   `json:"is_verified"`
	FollowerCount     int    `json:"follower_count"`
	LikesCount        int    `json:"likes_count"`
	QuotesCount       int    `json:"quotes_count"`
	RepliesCount      int    `json:"replies_count"`
	RepostsCount      int    `json:"reposts_count"`
	ViewsCount        int    `json:"views_count"`
}

// PostContent represents generic post content interface.
// This is a base structure for creating various types of posts.
// For specific post types, use TextPostContent, ImagePostContent, etc.
type PostContent struct {
	Text      string `json:"text,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	ReplyTo   string `json:"reply_to_id,omitempty"`
}

// TextPostContent represents content for text posts.
// Set QuotedPostID to create a quote post, or leave empty for regular text posts.
type TextPostContent struct {
	Text                    string          `json:"text"`
	LinkAttachment          string          `json:"link_attachment,omitempty"`
	PollAttachment          *PollAttachment `json:"poll_attachment,omitempty"`
	ReplyControl            ReplyControl    `json:"reply_control,omitempty"`
	ReplyTo                 string          `json:"reply_to_id,omitempty"`
	TopicTag                string          `json:"topic_tag,omitempty"`
	AllowlistedCountryCodes []string        `json:"allowlisted_country_codes,omitempty"`
	LocationID              string          `json:"location_id,omitempty"`
	AutoPublishText         bool            `json:"auto_publish_text,omitempty"`
	// QuotedPostID makes this a quote post when provided
	// Leave empty for regular text posts
	QuotedPostID string `json:"quoted_post_id,omitempty"`
}

// ImagePostContent represents content for image posts.
// Set QuotedPostID to create a quote post, or leave empty for regular image posts.
type ImagePostContent struct {
	Text                    string       `json:"text,omitempty"`
	ImageURL                string       `json:"image_url"`
	AltText                 string       `json:"alt_text,omitempty"`
	ReplyControl            ReplyControl `json:"reply_control,omitempty"`
	ReplyTo                 string       `json:"reply_to_id,omitempty"`
	TopicTag                string       `json:"topic_tag,omitempty"`
	AllowlistedCountryCodes []string     `json:"allowlisted_country_codes,omitempty"`
	LocationID              string       `json:"location_id,omitempty"`
	// QuotedPostID makes this a quote post when provided
	// Leave empty for regular image posts
	QuotedPostID string `json:"quoted_post_id,omitempty"`
}

// VideoPostContent represents content for video posts.
// Set QuotedPostID to create a quote post, or leave empty for regular video posts.
type VideoPostContent struct {
	Text                    string       `json:"text,omitempty"`
	VideoURL                string       `json:"video_url"`
	AltText                 string       `json:"alt_text,omitempty"`
	ReplyControl            ReplyControl `json:"reply_control,omitempty"`
	ReplyTo                 string       `json:"reply_to_id,omitempty"`
	TopicTag                string       `json:"topic_tag,omitempty"`
	AllowlistedCountryCodes []string     `json:"allowlisted_country_codes,omitempty"`
	LocationID              string       `json:"location_id,omitempty"`
	// QuotedPostID makes this a quote post when provided
	// Leave empty for regular image posts
	QuotedPostID string `json:"quoted_post_id,omitempty"`
}

// CarouselPostContent represents content for carousel posts.
// Set QuotedPostID to create a quote post, or leave empty for regular carousel posts.
type CarouselPostContent struct {
	Text                    string       `json:"text,omitempty"`
	Children                []string     `json:"children"` // Container IDs
	ReplyControl            ReplyControl `json:"reply_control,omitempty"`
	ReplyTo                 string       `json:"reply_to_id,omitempty"`
	TopicTag                string       `json:"topic_tag,omitempty"`
	AllowlistedCountryCodes []string     `json:"allowlisted_country_codes,omitempty"`
	LocationID              string       `json:"location_id,omitempty"`
	// QuotedPostID makes this a quote post when provided
	// Leave empty for regular image posts
	QuotedPostID string `json:"quoted_post_id,omitempty"`
}

// ReplyControl defines who can reply to a post
type ReplyControl string

const (
	// ReplyControlEveryone allows anyone to reply to the post
	ReplyControlEveryone ReplyControl = "everyone"
	// ReplyControlAccountsYouFollow allows only accounts you follow to reply
	ReplyControlAccountsYouFollow ReplyControl = "accounts_you_follow"
	// ReplyControlMentioned allows only mentioned users to reply
	ReplyControlMentioned ReplyControl = "mentioned_only"
	// ReplyControlParentPostAuthorOnly allows only the parent post author to reply
	ReplyControlParentPostAuthorOnly ReplyControl = "parent_post_author_only"
	// ReplyControlFollowersOnly allows only followers to reply to the post
	ReplyControlFollowersOnly ReplyControl = "followers_only"
)

// PostsResponse represents a paginated response containing multiple posts.
// Use the Paging field to navigate through large result sets.
// This is returned by endpoints like GetUserPosts, SearchPosts, etc.
type PostsResponse struct {
	Data   []Post `json:"data"`
	Paging Paging `json:"paging"`
}

// RepliesResponse represents a paginated response containing reply posts.
// Use the Paging field to navigate through conversation threads.
// This is returned by endpoints like GetReplies, GetConversation, etc.
type RepliesResponse struct {
	Data   []Post `json:"data"`
	Paging Paging `json:"paging"`
}

// InsightsResponse represents analytics and insights data for posts or user profiles.
// Contains an array of Insight objects with various metrics like views, likes, replies.
// Requires threads_manage_insights scope.
type InsightsResponse struct {
	Data []Insight `json:"data"`
}

// Insight represents an individual analytics metric with its values over time.
// Common metrics include views, likes, replies, reposts, quotes, follows, etc.
// The Period field indicates the time aggregation (e.g., "day", "week", "lifetime").
type Insight struct {
	Name        string      `json:"name"`
	Period      string      `json:"period"`
	Values      []Value     `json:"values"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	ID          string      `json:"id"`
	TotalValue  *TotalValue `json:"total_value,omitempty"`
}

// Value represents a metric value with optional timestamp
type Value struct {
	Value   int    `json:"value"`
	EndTime string `json:"end_time,omitempty"`
}

// TotalValue represents an aggregated metric value
type TotalValue struct {
	Value int `json:"value"`
}

// Paging represents pagination information for navigating through result sets.
// Use Before/After cursors to fetch previous/next pages of results.
// The direct Before/After fields are deprecated; use Cursors instead.
type Paging struct {
	Cursors *PagingCursors `json:"cursors,omitempty"`
	Before  string         `json:"before,omitempty"` // Deprecated: use Cursors.Before
	After   string         `json:"after,omitempty"`  // Deprecated: use Cursors.After
}

// PagingCursors represents cursor-based pagination for efficient data retrieval.
// Before cursor fetches older items, After cursor fetches newer items.
// These are opaque strings that should not be modified.
type PagingCursors struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// PaginationOptions represents standard pagination parameters for API requests.
// Limit controls the number of results per page (max varies by endpoint).
// Use Before/After cursors from previous responses to navigate pages.
type PaginationOptions struct {
	Limit  int    `json:"limit,omitempty"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// PostsOptions represents enhanced options for posts requests with time filtering
type PostsOptions struct {
	Limit  int    `json:"limit,omitempty"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Since  int64  `json:"since,omitempty"` // Unix timestamp
	Until  int64  `json:"until,omitempty"` // Unix timestamp
}

// RepliesOptions represents options for replies and conversation requests
type RepliesOptions struct {
	Limit   int    `json:"limit,omitempty"`
	Before  string `json:"before,omitempty"`
	After   string `json:"after,omitempty"`
	Reverse *bool  `json:"reverse,omitempty"` // true for reverse chronological, false for chronological (default: true)
}

// SearchOptions represents options for keyword and topic tag search
type SearchOptions struct {
	SearchType SearchType `json:"search_type,omitempty"`
	SearchMode SearchMode `json:"search_mode,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Since      int64      `json:"since,omitempty"` // Unix timestamp (must be >= 1688540400)
	Until      int64      `json:"until,omitempty"` // Unix timestamp
	Before     string     `json:"before,omitempty"`
	After      string     `json:"after,omitempty"`
}

// SearchType defines the search behavior
type SearchType string

const (
	// SearchTypeTop represents most popular search results (default)
	SearchTypeTop SearchType = "TOP"
	// SearchTypeRecent represents most recent search results
	SearchTypeRecent SearchType = "RECENT"
)

// SearchMode defines the search mode
type SearchMode string

const (
	// SearchModeKeyword treats query as keyword (default)
	SearchModeKeyword SearchMode = "KEYWORD"
	// SearchModeTag treats query as topic tag
	SearchModeTag SearchMode = "TAG"
)

// PublishingLimits represents current API quota usage and limits for various operations.
// This helps track how many posts, replies, and other actions you can still perform
// within the rate limit window. Check these before performing bulk operations.
type PublishingLimits struct {
	QuotaUsage               int         `json:"quota_usage"`
	Config                   QuotaConfig `json:"config"`
	ReplyQuotaUsage          int         `json:"reply_quota_usage"`
	ReplyConfig              QuotaConfig `json:"reply_config"`
	DeleteQuotaUsage         int         `json:"delete_quota_usage"`
	DeleteConfig             QuotaConfig `json:"delete_config"`
	LocationSearchQuotaUsage int         `json:"location_search_quota_usage"`
	LocationSearchConfig     QuotaConfig `json:"location_search_config"`
}

// QuotaConfig represents quota configuration for a specific operation type.
// QuotaTotal is the maximum allowed operations, QuotaDuration is the time window
// in seconds during which the quota applies.
type QuotaConfig struct {
	QuotaTotal    int `json:"quota_total"`
	QuotaDuration int `json:"quota_duration"`
}

// PollAttachment represents poll options when creating a post with a poll.
// Polls must have at least options A and B. Options C and D are optional.
// Polls automatically expire after 24 hours.
type PollAttachment struct {
	OptionA string `json:"option_a"`
	OptionB string `json:"option_b"`
	OptionC string `json:"option_c,omitempty"`
	OptionD string `json:"option_d,omitempty"`
}

// PollResult represents poll results and voting statistics when retrieving posts with polls.
// Contains the poll options and their vote percentages. The ExpirationTimestamp
// indicates when the poll closes (typically 24 hours after creation).
// TotalVotes shows the total number of votes cast in the poll.
type PollResult struct {
	OptionA                string  `json:"option_a"`
	OptionB                string  `json:"option_b"`
	OptionC                string  `json:"option_c,omitempty"`
	OptionD                string  `json:"option_d,omitempty"`
	OptionAVotesPercentage float64 `json:"option_a_votes_percentage"`
	OptionBVotesPercentage float64 `json:"option_b_votes_percentage"`
	OptionCVotesPercentage float64 `json:"option_c_votes_percentage,omitempty"`
	OptionDVotesPercentage float64 `json:"option_d_votes_percentage,omitempty"`
	TotalVotes             int     `json:"total_votes"`
	ExpirationTimestamp    Time    `json:"expiration_timestamp"`
}

// Location represents a geographic location that can be tagged in posts.
// Use SearchLocations to find location IDs, then include the ID when creating posts.
// Requires threads_location_tagging scope.
type Location struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Address    string  `json:"address,omitempty"`
	City       string  `json:"city,omitempty"`
	Country    string  `json:"country,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
	PostalCode string  `json:"postal_code,omitempty"`
}

// LocationSearchResponse represents the response from location search endpoint.
// Contains an array of Location objects matching the search query.
// Use the location IDs from this response when creating location-tagged posts.
type LocationSearchResponse struct {
	Data []Location `json:"data"`
}

// RepostContent represents the content required to create a repost.
// A repost shares another user's post to your profile without modifications.
// Only the original post ID is required.
type RepostContent struct {
	PostID string `json:"post_id"`
}

// PostOwner represents the owner of a post (only available on top-level posts you own)
type PostOwner struct {
	ID string `json:"id"`
}

// ChildrenData represents the children structure for carousel posts
type ChildrenData struct {
	Data []ChildPost `json:"data"`
}

// ChildPost represents a child post in a carousel
type ChildPost struct {
	ID string `json:"id"`
}
