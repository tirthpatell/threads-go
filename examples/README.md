# Examples

Working examples for the Threads API Go client.

## Prerequisites

1. Create a Meta App at [Meta for Developers](https://developers.facebook.com/apps/)
2. Enable Threads API following the [setup guide](https://developers.facebook.com/docs/threads/getting-started)
3. Configure OAuth redirect URI in app settings

## Setup

```bash
# Copy environment template
cp .env.example .env

# Add your credentials to .env
THREADS_CLIENT_ID=your_app_id_here
THREADS_CLIENT_SECRET=your_app_secret_here  
THREADS_REDIRECT_URI=https://your-domain.com/callback

# Load environment
source .env
```

## Available Examples

### Authentication (`authentication/`)
Complete OAuth 2.0 flow with token management:

```bash
cd authentication && go run main.go
```

- Authorization URL generation
- Code exchange for tokens
- Long-lived token conversion
- Token storage and validation

### Existing Token (`existing-token/`)  
Use client with existing access token (skip OAuth):

```bash
cd existing-token && go run main.go
```

- Direct token usage
- Token validation
- Immediate client setup

### Post Creation (`post-creation/`)
Create different post types:

```bash
cd post-creation && go run main.go
```

- Text, image, video posts
- Carousel posts (multiple media)
- Quote posts and reposts
- Advanced options (reply controls, tags)

### Reply Management (`reply-management/`)
Handle conversations and replies:

```bash
cd reply-management && go run main.go
```

- Create and retrieve replies
- Conversation threading
- Reply moderation (hide/unhide)
- Pagination and sorting

### Insights & Analytics (`insights/`)
Access performance metrics:

```bash
cd insights && go run main.go
```

- Post and account insights
- Publishing quotas
- Follower demographics
- Time-based filtering

## Quick Start Workflow

```bash
# 1. Authenticate first
cd authentication && go run main.go

# 2. Create posts  
cd ../post-creation && go run main.go

# 3. Check analytics
cd ../insights && go run main.go
```

## Environment Variables

**Required:**
- `THREADS_CLIENT_ID` - Your Meta app client ID
- `THREADS_CLIENT_SECRET` - Your Meta app secret  
- `THREADS_REDIRECT_URI` - OAuth redirect URI

**Optional:**
- `THREADS_ACCESS_TOKEN` - Existing token (for testing)
- `THREADS_DEBUG` - Enable debug logging

## Troubleshooting

- **Invalid credentials**: Check app ID/secret in Meta Developer Console
- **Redirect URI mismatch**: Ensure URI matches app configuration  
- **Rate limits**: Client handles automatically with backoff
- **Timeouts**: Set `THREADS_HTTP_TIMEOUT` if needed

## Support

- [Meta Threads API Documentation](https://developers.facebook.com/docs/threads) - Official API docs
- [Threads API Reference](https://developers.facebook.com/docs/threads/reference) - Complete endpoint reference
- Use debug mode for detailed request/response logging
