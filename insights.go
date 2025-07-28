package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// PostInsightMetric represents available post insight metrics
type PostInsightMetric string

const (
	// PostInsightViews represents the number of times a post was viewed
	PostInsightViews PostInsightMetric = "views"
	// PostInsightLikes represents the number of likes a post received
	PostInsightLikes PostInsightMetric = "likes"
	// PostInsightReplies represents the number of replies a post received
	PostInsightReplies PostInsightMetric = "replies"
	// PostInsightReposts represents the number of times a post was reposted
	PostInsightReposts PostInsightMetric = "reposts"
	// PostInsightQuotes represents the number of times a post was quoted
	PostInsightQuotes PostInsightMetric = "quotes"
	// PostInsightShares represents the number of times a post was shared
	PostInsightShares PostInsightMetric = "shares"
)

// AccountInsightMetric represents available account insight metrics
type AccountInsightMetric string

const (
	// AccountInsightViews represents the total views across all account posts
	AccountInsightViews AccountInsightMetric = "views"
	// AccountInsightLikes represents the total likes across all account posts
	AccountInsightLikes AccountInsightMetric = "likes"
	// AccountInsightReplies represents the total replies across all account posts
	AccountInsightReplies AccountInsightMetric = "replies"
	// AccountInsightReposts represents the total reposts across all account posts
	AccountInsightReposts AccountInsightMetric = "reposts"
	// AccountInsightQuotes represents the total quotes across all account posts
	AccountInsightQuotes AccountInsightMetric = "quotes"
	// AccountInsightClicks represents the total clicks across all account posts
	AccountInsightClicks AccountInsightMetric = "clicks"
	// AccountInsightFollowersCount represents the account's follower count
	AccountInsightFollowersCount AccountInsightMetric = "followers_count"
	// AccountInsightFollowerDemographics represents the demographic breakdown of followers
	AccountInsightFollowerDemographics AccountInsightMetric = "follower_demographics"
)

// InsightPeriod represents the time period for insights
type InsightPeriod string

const (
	// InsightPeriodDay represents daily insights data
	InsightPeriodDay InsightPeriod = "day"
	// InsightPeriodLifetime represents lifetime/total insights data
	InsightPeriodLifetime InsightPeriod = "lifetime"
)

// Constants for API limitations
const (
	// MinInsightTimestamp is the earliest Unix timestamp that can be used (1712991600)
	MinInsightTimestamp int64 = 1712991600
)

// FollowerDemographicsBreakdown represents breakdown options for follower demographics
type FollowerDemographicsBreakdown string

const (
	// BreakdownCountry represents follower demographics breakdown by country
	BreakdownCountry FollowerDemographicsBreakdown = "country"
	// BreakdownCity represents follower demographics breakdown by city
	BreakdownCity FollowerDemographicsBreakdown = "city"
	// BreakdownAge represents follower demographics breakdown by age group
	BreakdownAge FollowerDemographicsBreakdown = "age"
	// BreakdownGender represents follower demographics breakdown by gender
	BreakdownGender FollowerDemographicsBreakdown = "gender"
)

// PostInsightsOptions represents options for post insights requests
type PostInsightsOptions struct {
	Metrics []PostInsightMetric `json:"metrics,omitempty"`
	Period  InsightPeriod       `json:"period,omitempty"`
	Since   *time.Time          `json:"since,omitempty"`
	Until   *time.Time          `json:"until,omitempty"`
}

// AccountInsightsOptions represents options for account insights requests
type AccountInsightsOptions struct {
	Metrics   []AccountInsightMetric `json:"metrics,omitempty"`
	Period    InsightPeriod          `json:"period,omitempty"`
	Since     *time.Time             `json:"since,omitempty"`
	Until     *time.Time             `json:"until,omitempty"`
	Breakdown string                 `json:"breakdown,omitempty"` // For follower_demographics: country, city, age, or gender
}

// GetPostInsights retrieves insights for a specific post.
// For insights API documentation, see: https://developers.facebook.com/docs/threads/insights
func (c *Client) GetPostInsights(ctx context.Context, postID PostID, metrics []string) (*InsightsResponse, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "postID cannot be empty", "postID")
	}

	// Validate metrics
	validMetrics := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		if err := c.validatePostInsightMetric(metric); err != nil {
			return nil, err
		}
		validMetrics = append(validMetrics, metric)
	}

	// If no metrics specified, use default metrics
	if len(validMetrics) == 0 {
		validMetrics = []string{
			string(PostInsightViews),
			string(PostInsightLikes),
			string(PostInsightReplies),
			string(PostInsightReposts),
		}
	}

	params := url.Values{}
	params.Set("metric", strings.Join(validMetrics, ","))

	path := fmt.Sprintf("/%s/insights", postID.String())
	response, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, fmt.Errorf("failed to get post insights: %w", err)
	}

	var insightsResponse InsightsResponse
	if err := json.Unmarshal(response.Body, &insightsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode insights response: %w", err)
	}

	return &insightsResponse, nil
}

// GetPostInsightsWithOptions retrieves insights for a specific post with advanced options
func (c *Client) GetPostInsightsWithOptions(ctx context.Context, postID PostID, opts *PostInsightsOptions) (*InsightsResponse, error) {
	if !postID.Valid() {
		return nil, NewValidationError(400, ErrEmptyPostID, "postID cannot be empty", "postID")
	}

	if opts == nil {
		opts = &PostInsightsOptions{}
	}

	// Validate and prepare metrics
	var validMetrics []string
	if len(opts.Metrics) > 0 {
		for _, metric := range opts.Metrics {
			if err := c.validatePostInsightMetric(string(metric)); err != nil {
				return nil, err
			}
			validMetrics = append(validMetrics, string(metric))
		}
	} else {
		// Use default metrics if none specified
		validMetrics = []string{
			string(PostInsightViews),
			string(PostInsightLikes),
			string(PostInsightReplies),
			string(PostInsightReposts),
		}
	}

	params := url.Values{}
	params.Set("metric", strings.Join(validMetrics, ","))

	// Add period if specified
	if opts.Period != "" {
		if err := c.validateInsightPeriod(string(opts.Period)); err != nil {
			return nil, err
		}
		params.Set("period", string(opts.Period))
	}

	// Add date range if specified
	if opts.Since != nil {
		params.Set("since", fmt.Sprintf("%d", opts.Since.Unix()))
	}
	if opts.Until != nil {
		params.Set("until", fmt.Sprintf("%d", opts.Until.Unix()))
	}

	// Validate date range
	if opts.Since != nil && opts.Until != nil {
		if opts.Since.After(*opts.Until) {
			return nil, NewValidationError(400, "Invalid date range", "since date cannot be after until date", "since")
		}
	}

	path := fmt.Sprintf("/%s/insights", postID.String())
	response, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, fmt.Errorf("failed to get post insights: %w", err)
	}

	var insightsResponse InsightsResponse
	if err := json.Unmarshal(response.Body, &insightsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode insights response: %w", err)
	}

	return &insightsResponse, nil
}

// GetAccountInsights retrieves insights for a user account
func (c *Client) GetAccountInsights(ctx context.Context, userID UserID, metrics []string, period string) (*InsightsResponse, error) {
	if !userID.Valid() {
		return nil, NewValidationError(400, ErrEmptyUserID, "userID cannot be empty", "userID")
	}

	// Validate metrics
	validMetrics := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		if err := c.validateAccountInsightMetric(metric); err != nil {
			return nil, err
		}
		validMetrics = append(validMetrics, metric)
	}

	// If no metrics specified, use default metrics
	if len(validMetrics) == 0 {
		validMetrics = []string{
			string(AccountInsightViews),
			string(AccountInsightLikes),
			string(AccountInsightReplies),
			string(AccountInsightReposts),
		}
	}

	params := url.Values{}
	params.Set("metric", strings.Join(validMetrics, ","))

	// Validate and set period
	if period != "" {
		if err := c.validateInsightPeriod(period); err != nil {
			return nil, err
		}
		params.Set("period", period)
	} else {
		// Default to lifetime if no period specified
		params.Set("period", string(InsightPeriodLifetime))
	}

	path := fmt.Sprintf("/%s/threads_insights", userID.String())
	response, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, fmt.Errorf("failed to get account insights: %w", err)
	}

	var insightsResponse InsightsResponse
	if err := json.Unmarshal(response.Body, &insightsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode insights response: %w", err)
	}

	return &insightsResponse, nil
}

// GetAccountInsightsWithOptions retrieves insights for a user account with advanced options
func (c *Client) GetAccountInsightsWithOptions(ctx context.Context, userID UserID, opts *AccountInsightsOptions) (*InsightsResponse, error) {
	if !userID.Valid() {
		return nil, NewValidationError(400, ErrEmptyUserID, "userID cannot be empty", "userID")
	}

	if opts == nil {
		opts = &AccountInsightsOptions{}
	}

	// Validate and prepare metrics
	var validMetrics []string
	if len(opts.Metrics) > 0 {
		for _, metric := range opts.Metrics {
			if err := c.validateAccountInsightMetric(string(metric)); err != nil {
				return nil, err
			}
			validMetrics = append(validMetrics, string(metric))
		}
	} else {
		// Use default metrics if none specified
		validMetrics = []string{
			string(AccountInsightViews),
			string(AccountInsightLikes),
			string(AccountInsightReplies),
			string(AccountInsightReposts),
		}
	}

	params := url.Values{}
	params.Set("metric", strings.Join(validMetrics, ","))

	// Add period if specified, otherwise default to lifetime
	if opts.Period != "" {
		if err := c.validateInsightPeriod(string(opts.Period)); err != nil {
			return nil, err
		}
		params.Set("period", string(opts.Period))
	} else {
		params.Set("period", string(InsightPeriodLifetime))
	}

	// Check for metrics that don't support since/until parameters
	hasFollowerDemographics := false
	hasFollowersCount := false
	for _, metric := range validMetrics {
		if metric == string(AccountInsightFollowerDemographics) {
			hasFollowerDemographics = true
		}
		if metric == string(AccountInsightFollowersCount) {
			hasFollowersCount = true
		}
	}

	if hasFollowerDemographics {
		// follower_demographics doesn't support since/until parameters
		if opts.Since != nil || opts.Until != nil {
			return nil, NewValidationError(400, "Invalid parameters",
				"follower_demographics metric does not support since and until parameters", "metric")
		}

		// Validate breakdown parameter
		if opts.Breakdown != "" {
			if err := c.validateFollowerDemographicsBreakdown(opts.Breakdown); err != nil {
				return nil, err
			}
			params.Set("breakdown", opts.Breakdown)
		}
	}

	if hasFollowersCount {
		// followers_count doesn't support since/until parameters
		if opts.Since != nil || opts.Until != nil {
			return nil, NewValidationError(400, "Invalid parameters",
				"followers_count metric does not support since and until parameters", "metric")
		}
	}

	if !hasFollowerDemographics && !hasFollowersCount {
		// Validate minimum timestamp for other metrics
		if opts.Since != nil && opts.Since.Unix() < MinInsightTimestamp {
			return nil, NewValidationError(400, "Invalid since timestamp",
				fmt.Sprintf("since timestamp must be >= %d", MinInsightTimestamp), "since")
		}
		if opts.Until != nil && opts.Until.Unix() < MinInsightTimestamp {
			return nil, NewValidationError(400, "Invalid until timestamp",
				fmt.Sprintf("until timestamp must be >= %d", MinInsightTimestamp), "until")
		}

		// Add date range if specified
		if opts.Since != nil {
			params.Set("since", fmt.Sprintf("%d", opts.Since.Unix()))
		}
		if opts.Until != nil {
			params.Set("until", fmt.Sprintf("%d", opts.Until.Unix()))
		}
	}

	// Validate date range
	if opts.Since != nil && opts.Until != nil {
		if opts.Since.After(*opts.Until) {
			return nil, NewValidationError(400, "Invalid date range", "since date cannot be after until date", "since")
		}
	}

	path := fmt.Sprintf("/%s/threads_insights", userID.String())
	response, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, fmt.Errorf("failed to get account insights: %w", err)
	}

	var insightsResponse InsightsResponse
	if err := json.Unmarshal(response.Body, &insightsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode insights response: %w", err)
	}

	return &insightsResponse, nil
}

// validatePostInsightMetric validates if the provided metric is supported for post insights
func (c *Client) validatePostInsightMetric(metric string) error {
	validMetrics := map[string]bool{
		string(PostInsightViews):   true,
		string(PostInsightLikes):   true,
		string(PostInsightReplies): true,
		string(PostInsightReposts): true,
		string(PostInsightQuotes):  true,
		string(PostInsightShares):  true,
	}

	if !validMetrics[metric] {
		return NewValidationError(400, "Invalid post insight metric",
			fmt.Sprintf("metric '%s' is not supported for post insights", metric), "metric")
	}

	return nil
}

// validateAccountInsightMetric validates if the provided metric is supported for account insights
func (c *Client) validateAccountInsightMetric(metric string) error {
	validMetrics := map[string]bool{
		string(AccountInsightViews):                true,
		string(AccountInsightLikes):                true,
		string(AccountInsightReplies):              true,
		string(AccountInsightReposts):              true,
		string(AccountInsightQuotes):               true,
		string(AccountInsightClicks):               true,
		string(AccountInsightFollowersCount):       true,
		string(AccountInsightFollowerDemographics): true,
	}

	if !validMetrics[metric] {
		return NewValidationError(400, "Invalid account insight metric",
			fmt.Sprintf("metric '%s' is not supported for account insights", metric), "metric")
	}

	return nil
}

// validateInsightPeriod validates if the provided period is supported
func (c *Client) validateInsightPeriod(period string) error {
	validPeriods := map[string]bool{
		string(InsightPeriodDay):      true,
		string(InsightPeriodLifetime): true,
	}

	if !validPeriods[period] {
		return NewValidationError(400, "Invalid insight period",
			fmt.Sprintf("period '%s' is not supported", period), "period")
	}

	return nil
}

// validateFollowerDemographicsBreakdown validates the breakdown parameter for follower demographics
func (c *Client) validateFollowerDemographicsBreakdown(breakdown string) error {
	validBreakdowns := map[string]bool{
		string(BreakdownCountry): true,
		string(BreakdownCity):    true,
		string(BreakdownAge):     true,
		string(BreakdownGender):  true,
	}

	if !validBreakdowns[breakdown] {
		return NewValidationError(400, "Invalid breakdown parameter",
			fmt.Sprintf("breakdown '%s' is not supported. Valid values: country, city, age, gender", breakdown), "breakdown")
	}

	return nil
}

// GetAvailablePostInsightMetrics returns all available post insight metrics
func (c *Client) GetAvailablePostInsightMetrics() []PostInsightMetric {
	return []PostInsightMetric{
		PostInsightViews,
		PostInsightLikes,
		PostInsightReplies,
		PostInsightReposts,
		PostInsightQuotes,
		PostInsightShares,
	}
}

// GetAvailableAccountInsightMetrics returns all available account insight metrics
func (c *Client) GetAvailableAccountInsightMetrics() []AccountInsightMetric {
	return []AccountInsightMetric{
		AccountInsightViews,
		AccountInsightLikes,
		AccountInsightReplies,
		AccountInsightReposts,
		AccountInsightQuotes,
		AccountInsightClicks,
		AccountInsightFollowersCount,
		AccountInsightFollowerDemographics,
	}
}

// GetAvailableInsightPeriods returns all available insight periods
func (c *Client) GetAvailableInsightPeriods() []InsightPeriod {
	return []InsightPeriod{
		InsightPeriodDay,
		InsightPeriodLifetime,
	}
}

// GetAvailableFollowerDemographicsBreakdowns returns all available breakdown options for follower demographics
func (c *Client) GetAvailableFollowerDemographicsBreakdowns() []FollowerDemographicsBreakdown {
	return []FollowerDemographicsBreakdown{
		BreakdownCountry,
		BreakdownCity,
		BreakdownAge,
		BreakdownGender,
	}
}
