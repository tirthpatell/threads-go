package threads

// getUserID extracts user ID from token info
func (c *Client) getUserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.tokenInfo != nil && c.tokenInfo.UserID != "" {
		return c.tokenInfo.UserID
	}
	return ""
}

// handleAPIError processes API error responses
func (c *Client) handleAPIError(resp *Response) error {
	return c.httpClient.createErrorFromResponse(resp)
}
