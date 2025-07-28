package threads

import (
	"context"
	"fmt"
	"net/url"
)

// SearchLocations searches for locations by query, latitude/longitude
func (c *Client) SearchLocations(ctx context.Context, query string, latitude, longitude *float64) (*LocationSearchResponse, error) {
	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters
	params := url.Values{
		"fields": {LocationFields}, // Include all location fields for search results
	}

	// At least one parameter must be provided
	hasParams := false

	if query != "" {
		params.Set("q", query)
		hasParams = true
	}

	if latitude != nil {
		params.Set("latitude", fmt.Sprintf("%f", *latitude))
		hasParams = true
	}

	if longitude != nil {
		params.Set("longitude", fmt.Sprintf("%f", *longitude))
		hasParams = true
	}

	if !hasParams {
		return nil, NewValidationError(400, "At least one search parameter required", "Must provide query, latitude, or longitude", "search_params")
	}

	// Make API call
	resp, err := c.httpClient.GET("/location_search", params, c.getAccessTokenSafe())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var locationResp LocationSearchResponse
	if err := safeJSONUnmarshal(resp.Body, &locationResp, "location search response", resp.RequestID); err != nil {
		return nil, err
	}

	return &locationResp, nil
}

// GetLocation retrieves location details
func (c *Client) GetLocation(ctx context.Context, locationID LocationID) (*Location, error) {
	if !locationID.Valid() {
		return nil, NewValidationError(400, "Location ID is required", "locationID cannot be empty", "location_id")
	}

	// Ensure we have a valid token
	if err := c.EnsureValidToken(ctx); err != nil {
		return nil, err
	}

	// Build query parameters with location fields
	params := url.Values{
		"fields": {LocationFields},
	}

	// Make API call
	path := fmt.Sprintf("/%s", locationID.String())
	resp, err := c.httpClient.GET(path, params, c.getAccessTokenSafe())
	if err != nil {
		return nil, fmt.Errorf("failed to get location: %w", err)
	}

	// Handle specific error cases
	if resp.StatusCode == 404 {
		return nil, NewValidationError(404, "Location not found", fmt.Sprintf("Location with ID %s does not exist", locationID.String()), "location_id")
	}

	if resp.StatusCode != 200 {
		return nil, c.handleAPIError(resp)
	}

	// Parse response
	var location Location
	if err := safeJSONUnmarshal(resp.Body, &location, "location details", resp.RequestID); err != nil {
		return nil, err
	}

	return &location, nil
}
