package threads

import (
	"context"
	"fmt"
)

// PostIterator provides an iterator for paginating through posts
type PostIterator struct {
	client     PostReader
	userID     UserID
	options    *PostsOptions
	nextCursor string
	done       bool
}

// NewPostIterator creates a new post iterator
func NewPostIterator(client PostReader, userID UserID, opts *PostsOptions) *PostIterator {
	if opts == nil {
		opts = &PostsOptions{
			Limit: DefaultPostsLimit,
		}
	}

	return &PostIterator{
		client:  client,
		userID:  userID,
		options: opts,
		done:    false,
	}
}

// Next retrieves the next page of posts
func (p *PostIterator) Next(ctx context.Context) (*PostsResponse, error) {
	if p.done {
		return nil, nil
	}

	// Update cursor for pagination
	opts := *p.options
	if p.nextCursor != "" {
		opts.After = p.nextCursor
	}

	// Fetch posts
	response, err := p.client.GetUserPostsWithOptions(ctx, p.userID, &opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch posts: %w", err)
	}

	// Update cursor for next iteration
	if response.Paging.Cursors != nil && response.Paging.Cursors.After != "" {
		p.nextCursor = response.Paging.Cursors.After
	} else if response.Paging.After != "" {
		p.nextCursor = response.Paging.After
	} else {
		// No more pages
		p.done = true
	}

	// Mark as done if no posts returned
	if len(response.Data) == 0 {
		p.done = true
	}

	return response, nil
}

// HasNext returns true if there are more pages to fetch
func (p *PostIterator) HasNext() bool {
	return !p.done
}

// Reset resets the iterator to start from the beginning
func (p *PostIterator) Reset() {
	p.nextCursor = ""
	p.done = false
}

// Collect fetches all remaining pages and returns them as a single slice
func (p *PostIterator) Collect(ctx context.Context) ([]Post, error) {
	var allPosts []Post

	for p.HasNext() {
		response, err := p.Next(ctx)
		if err != nil {
			return nil, err
		}

		if response != nil {
			allPosts = append(allPosts, response.Data...)
		}
	}

	return allPosts, nil
}

// ReplyIterator provides an iterator for paginating through replies
type ReplyIterator struct {
	client     ReplyManager
	postID     PostID
	options    *RepliesOptions
	nextCursor string
	done       bool
}

// NewReplyIterator creates a new reply iterator
func NewReplyIterator(client ReplyManager, postID PostID, opts *RepliesOptions) *ReplyIterator {
	if opts == nil {
		opts = &RepliesOptions{
			Limit: DefaultPostsLimit,
		}
	}

	return &ReplyIterator{
		client:  client,
		postID:  postID,
		options: opts,
		done:    false,
	}
}

// Next retrieves the next page of replies
func (r *ReplyIterator) Next(ctx context.Context) (*RepliesResponse, error) {
	if r.done {
		return nil, nil
	}

	// Update cursor for pagination
	opts := *r.options
	if r.nextCursor != "" {
		opts.After = r.nextCursor
	}

	// Fetch replies
	response, err := r.client.GetReplies(ctx, r.postID, &opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch replies: %w", err)
	}

	// Update cursor for next iteration
	if response.Paging.Cursors != nil && response.Paging.Cursors.After != "" {
		r.nextCursor = response.Paging.Cursors.After
	} else if response.Paging.After != "" {
		r.nextCursor = response.Paging.After
	} else {
		// No more pages
		r.done = true
	}

	// Mark as done if no replies returned
	if len(response.Data) == 0 {
		r.done = true
	}

	return response, nil
}

// HasNext returns true if there are more pages to fetch
func (r *ReplyIterator) HasNext() bool {
	return !r.done
}

// Reset resets the iterator to start from the beginning
func (r *ReplyIterator) Reset() {
	r.nextCursor = ""
	r.done = false
}

// Collect fetches all remaining pages and returns them as a single slice
func (r *ReplyIterator) Collect(ctx context.Context) ([]Post, error) {
	var allReplies []Post

	for r.HasNext() {
		response, err := r.Next(ctx)
		if err != nil {
			return nil, err
		}

		if response != nil {
			allReplies = append(allReplies, response.Data...)
		}
	}

	return allReplies, nil
}

// SearchIterator provides an iterator for paginating through search results
type SearchIterator struct {
	client     SearchProvider
	query      string
	options    *SearchOptions
	searchType string // "keyword" or "tag"
	nextCursor string
	done       bool
}

// NewSearchIterator creates a new search iterator
func NewSearchIterator(client SearchProvider, query string, searchType string, opts *SearchOptions) *SearchIterator {
	if opts == nil {
		opts = &SearchOptions{
			Limit: DefaultPostsLimit,
		}
	}

	return &SearchIterator{
		client:     client,
		query:      query,
		searchType: searchType,
		options:    opts,
		done:       false,
	}
}

// Next retrieves the next page of search results
func (s *SearchIterator) Next(ctx context.Context) (*PostsResponse, error) {
	if s.done {
		return nil, nil
	}

	// Update cursor for pagination
	opts := *s.options
	if s.nextCursor != "" {
		opts.After = s.nextCursor
	}

	// Perform search based on type
	var response *PostsResponse
	var err error

	switch s.searchType {
	case "keyword":
		response, err = s.client.KeywordSearch(ctx, s.query, &opts)
	case "tag":
		opts.SearchMode = SearchModeTag
		response, err = s.client.KeywordSearch(ctx, s.query, &opts)
	default:
		return nil, fmt.Errorf("invalid search type: %s", s.searchType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to perform search: %w", err)
	}

	// Update cursor for next iteration
	if response.Paging.Cursors != nil && response.Paging.Cursors.After != "" {
		s.nextCursor = response.Paging.Cursors.After
	} else if response.Paging.After != "" {
		s.nextCursor = response.Paging.After
	} else {
		// No more pages
		s.done = true
	}

	// Mark as done if no results returned
	if len(response.Data) == 0 {
		s.done = true
	}

	return response, nil
}

// HasNext returns true if there are more pages to fetch
func (s *SearchIterator) HasNext() bool {
	return !s.done
}

// Reset resets the iterator to start from the beginning
func (s *SearchIterator) Reset() {
	s.nextCursor = ""
	s.done = false
}

// Collect fetches all remaining pages and returns them as a single slice
func (s *SearchIterator) Collect(ctx context.Context) ([]Post, error) {
	var allPosts []Post

	for s.HasNext() {
		response, err := s.Next(ctx)
		if err != nil {
			return nil, err
		}

		if response != nil {
			allPosts = append(allPosts, response.Data...)
		}
	}

	return allPosts, nil
}
