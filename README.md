# Threads API Go Client

[![Go Reference](https://pkg.go.dev/badge/github.com/tirthpatell/threads-go.svg)](https://pkg.go.dev/github.com/tirthpatell/threads-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tirthpatell/threads-go)](https://goreportcard.com/report/github.com/tirthpatell/threads-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Production-ready Go client for the Threads API with complete endpoint coverage, OAuth 2.0 authentication, rate limiting, and comprehensive error handling.

## Features

- Complete API coverage (posts, users, replies, insights, locations)
- OAuth 2.0 flow and existing token support
- Intelligent rate limiting with exponential backoff
- Type-safe error handling
- Thread-safe concurrent usage
- Comprehensive test coverage

## Installation

```bash
go get github.com/tirthpatell/threads-go
```

## Quick Start

### With Existing Token

```go
client, err := threads.NewClientWithToken("your-access-token", &threads.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    RedirectURI:  "your-redirect-uri",
})

// Create a post
post, err := client.CreateTextPost(context.Background(), &threads.TextPostContent{
    Text: "Hello Threads!",
})
```

### OAuth Flow

```go
config := &threads.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret", 
    RedirectURI:  "your-redirect-uri",
    Scopes:       []string{"threads_basic", "threads_content_publish"},
}

client, err := threads.NewClient(config)

// Get authorization URL
authURL := client.GetAuthURL(config.Scopes)
// Redirect user to authURL

// Exchange authorization code for token
err = client.ExchangeCodeForToken("auth-code-from-callback")
err = client.GetLongLivedToken() // Convert to long-lived token
```

### Environment Variables

```bash
export THREADS_CLIENT_ID="your-client-id"
export THREADS_CLIENT_SECRET="your-client-secret"
export THREADS_REDIRECT_URI="your-redirect-uri"
export THREADS_ACCESS_TOKEN="your-access-token"  # optional
```

```go
client, err := threads.NewClientFromEnv()
```

## Available Scopes

- `threads_basic` - Basic profile access
- `threads_content_publish` - Create and publish posts  
- `threads_manage_insights` - Access analytics data
- `threads_manage_replies` - Manage replies and conversations
- `threads_read_replies` - Read replies to posts
- `threads_keyword_search` - Search functionality
- `threads_delete` - Delete posts
- `threads_location_tagging` - Location services

## API Usage

### Posts

```go
// Create different post types
textPost, err := client.CreateTextPost(ctx, &threads.TextPostContent{
    Text: "Hello Threads!",
})

imagePost, err := client.CreateImagePost(ctx, &threads.ImagePostContent{
    Text: "Check this out!",
    ImageURL: "https://example.com/image.jpg",
})

// Get posts
post, err := client.GetPost(ctx, threads.PostID("123"))
posts, err := client.GetUserPosts(ctx, threads.UserID("456"), &threads.PostsOptions{Limit: 25})

// Delete post
err = client.DeletePost(ctx, threads.PostID("123"))
```

### Users & Profiles

```go
// Get user info
me, err := client.GetMe(ctx)
user, err := client.GetUser(ctx, threads.UserID("123"))

// Public profiles
publicUser, err := client.LookupPublicProfile(ctx, "@username")
posts, err := client.GetPublicProfilePosts(ctx, "username", nil)
```

### Replies & Conversations

```go
// Reply to posts
reply, err := client.ReplyToPost(ctx, threads.PostID("123"), &threads.PostContent{
    Text: "Great post!",
})

// Get replies
replies, err := client.GetReplies(ctx, threads.PostID("123"), &threads.RepliesOptions{Limit: 50})

// Manage visibility
err = client.HideReply(ctx, threads.PostID("456"))
```

### Insights & Analytics

```go
// Post insights
insights, err := client.GetPostInsights(ctx, threads.PostID("123"), []string{"views", "likes"})

// Account insights  
insights, err := client.GetAccountInsights(ctx, threads.UserID("456"), []string{"views"}, "lifetime")
```

### Search & Locations

```go
// Search posts
results, err := client.KeywordSearch(ctx, "golang", &threads.SearchOptions{Limit: 25})

// Location search
locations, err := client.SearchLocations(ctx, "New York", nil, nil)
```

### Pagination & Iterators

For large datasets, use iterators to automatically handle pagination:

```go
// Posts iterator
userID := threads.ConvertToUserID("user_id")
iterator := threads.NewPostIterator(client, userID, &threads.PostsOptions{
    Limit: 25,
})

for iterator.HasNext() {
    response, err := iterator.Next(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, post := range response.Data {
        fmt.Printf("Post: %s\n", post.Text)
    }
}

// Replies iterator
replyIterator := threads.NewReplyIterator(client, threads.PostID("123"), &threads.RepliesOptions{
    Limit: 50,
})

// Search iterator
searchIterator := threads.NewSearchIterator(client, "golang", "keyword", &threads.SearchOptions{
    Limit: 25,
})

// Collect all results at once
allPosts, err := iterator.Collect(ctx)
```

## Configuration

```go
config := &threads.Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    RedirectURI:  "https://yourapp.com/callback",
    Scopes:       []string{"threads_basic", "threads_content_publish"},
    HTTPTimeout:  30 * time.Second,
    Debug:        false,
}
```

For advanced configuration including retry logic, custom logging, and token storage, see the [GoDoc documentation](https://pkg.go.dev/github.com/tirthpatell/threads-go).

## Error Handling

The client provides typed errors for different scenarios:

```go
switch {
case threads.IsAuthenticationError(err):
    // Handle authentication issues
case threads.IsRateLimitError(err):
    rateLimitErr := err.(*threads.RateLimitError)
    time.Sleep(rateLimitErr.RetryAfter)
case threads.IsValidationError(err):
    validationErr := err.(*threads.ValidationError)
    log.Printf("Invalid %s: %s", validationErr.Field, err.Error())
}
```

Error types: `AuthenticationError`, `RateLimitError`, `ValidationError`, `NetworkError`, `APIError`

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires valid credentials)
export THREADS_ACCESS_TOKEN="your-token"
go test ./tests/integration/...
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Official Documentation

- [Meta Threads API Documentation](https://developers.facebook.com/docs/threads)
- [Threads API Reference](https://developers.facebook.com/docs/threads/reference)
- [Authentication Guide](https://developers.facebook.com/docs/threads/getting-started)

## License

MIT License - see [LICENSE](LICENSE) file for details.
