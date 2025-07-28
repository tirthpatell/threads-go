# Contributing to Threads API Go Client

Thank you for contributing! This guide helps you get started with development.

## Prerequisites

- Go 1.21 or later
- Git
- Threads API credentials (for testing)

## Development Setup

```bash
# Fork and clone
git clone https://github.com/tirthpatell/threads-go.git
cd threads-go

# Install dependencies
go mod download

# Set up environment
export THREADS_CLIENT_ID="your-client-id"
export THREADS_CLIENT_SECRET="your-client-secret"
export THREADS_REDIRECT_URI="your-redirect-uri"
export THREADS_ACCESS_TOKEN="your-token"

# Run tests
go test ./...
go test ./tests/integration/...
```

## Code Guidelines

### Style

Follow standard Go conventions:

- Use `gofmt` and `goimports`
- Document all public APIs with GoDoc comments
- Handle errors explicitly with context
- Keep functions focused and under 50 lines
- Use descriptive names

```go
// Good
func validatePostContent(content *PostContent) error {
    if content.Text == "" {
        return NewValidationError(400, "Text required", "Post text cannot be empty", "text")
    }
    return nil
}
```

### Testing

Write tests for new code:

```go
func TestClient_CreateTextPost(t *testing.T) {
    tests := []struct {
        name    string
        content *TextPostContent
        wantErr bool
    }{
        {"valid post", &TextPostContent{Text: "Hello"}, false},
        {"empty text", &TextPostContent{Text: ""}, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Pull Request Process

### Before Submitting

1. Run tests: `go test ./...`
2. Check formatting: `go fmt ./...`
3. Update documentation for new features
4. Write descriptive commit messages

### Commit Format

```
type(scope): brief description

feat(auth): add token refresh support
fix(posts): handle empty response correctly
docs(readme): update authentication examples
```

### Branch Naming

- `feature/add-location-search`
- `fix/token-refresh-race`
- `docs/update-examples`

## Getting Help

- **Issues**: Bug reports and feature requests
- **Discussions**: Questions and general discussion

When reporting bugs, include:
- Go version and OS
- Code that reproduces the issue
- Expected vs actual behavior

## Recognition

Contributors are recognized in CHANGELOG.md and release notes for significant contributions.
