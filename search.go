package threads

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// KeywordSearch searches for public Threads media by keyword
func (c *Client) KeywordSearch(ctx context.Context, query string, opts *SearchOptions) (*PostsResponse, error) {
	if strings.TrimSpace(query) == "" {
		return nil, NewValidationError(400, ErrEmptySearchQuery, "Cannot search without a query string", "query")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters according to API documentation
	params := url.Values{
		"q":      {query},
		"fields": {PostExtendedFields}, // Use PostExtendedFields for comprehensive search results
	}

	// Add search options if provided
	if opts != nil {
		if opts.SearchType != "" {
			params.Set("search_type", string(opts.SearchType))
		}
		if opts.SearchMode != "" {
			params.Set("search_mode", string(opts.SearchMode))
		}
		if opts.MediaType != "" {
			// Validate media type
			mediaType := strings.ToUpper(opts.MediaType)
			if mediaType != MediaTypeText && mediaType != MediaTypeImage && mediaType != MediaTypeVideo {
				return nil, NewValidationError(400, "Invalid media type", "Media type must be TEXT, IMAGE, or VIDEO", "media_type")
			}
			params.Set("media_type", mediaType)
		}
		if opts.Limit > 0 {
			if opts.Limit > 100 {
				return nil, NewValidationError(400, "Limit too large", "Maximum limit is 100 posts per request", "limit")
			}
			params.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Since > 0 {
			// Validate timestamp according to API documentation
			if opts.Since < 1688540400 {
				return nil, NewValidationError(400, "Invalid since timestamp", "Since timestamp must be greater than or equal to 1688540400", "since")
			}
			params.Set("since", fmt.Sprintf("%d", opts.Since))
		}
		if opts.Until > 0 {
			params.Set("until", fmt.Sprintf("%d", opts.Until))
		}
		if opts.Before != "" {
			params.Set("before", opts.Before)
		}
		if opts.After != "" {
			params.Set("after", opts.After)
		}
	}

	// Make API call to keyword search endpoint
	path := "/keyword_search"
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var postsResp PostsResponse
	if err := safeJSONUnmarshal(resp.Body, &postsResp, "keyword search response", resp.RequestID); err != nil {
		return nil, err
	}

	return &postsResp, nil
}

