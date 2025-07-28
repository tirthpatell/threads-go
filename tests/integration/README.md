# Integration Tests

Tests the **public API only**, simulating how real users interact with the package.

For official API behavior, see [Meta Threads API Documentation](https://developers.facebook.com/docs/threads).

## Running Tests

```bash
# All tests
go test -v -tags=integration

# Specific test
go test -v -tags=integration -run TestIntegration_PublicAPI_Authentication

# With timeout
go test -v -tags=integration -timeout 300s
```

## Test Coverage

- Authentication and token management
- User operations (`GetMe`, `GetUser`)  
- Post operations (`GetUserPosts`, `CreateTextPost`, `DeletePost`)
- Rate limiting and error handling
- Insights and publishing limits

## Test Approach

These tests behave like package users:
- Import: `import threads "github.com/tirthpatell/threads-go"`
- Use only public API (exported functions)
- Test against actual Threads API
- Validate API contracts and behavior
