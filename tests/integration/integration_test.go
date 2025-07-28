//go:build integration

// This file contains integration tests for GitHub CI that use environment variables
// from GitHub secrets. For local testing with hardcoded credentials, see integration_local_test.go

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	threads "github.com/tirthpatell/threads-go"
)

// Integration test configuration - using GitHub secrets
var (
	testUserID      = os.Getenv("THREADS_USER_ID")
	testAccessToken = os.Getenv("THREADS_ACCESS_TOKEN")
	testClientID    = os.Getenv("THREADS_CLIENT_ID")
	testSecret      = os.Getenv("THREADS_CLIENT_SECRET")
	testRedirectURI = os.Getenv("THREADS_REDIRECT_URI")

	// Media URLs for testing
	testImageURL1 = os.Getenv("THREADS_TEST_IMG1")
	testImageURL2 = os.Getenv("THREADS_TEST_IMG2")
	testVideoURL  = os.Getenv("THREADS_TEST_VID1")
)

// TestLogger implements the threads.Logger interface for testing
type TestLogger struct {
	t       *testing.T
	verbose bool
}

// NewTestLogger creates a new test logger with appropriate verbosity for the environment
func NewTestLogger(t *testing.T) *TestLogger {
	// Reduce verbosity in CI environments
	verbose := os.Getenv("CI") == "" && os.Getenv("GITHUB_ACTIONS") == ""
	return &TestLogger{t: t, verbose: verbose}
}

func (l *TestLogger) Debug(msg string, fields ...any) {
	if l.verbose {
		l.t.Logf("[DEBUG] %s %v", msg, l.sanitizeFields(fields...))
	}
}

func (l *TestLogger) Info(msg string, fields ...any) {
	l.t.Logf("[INFO] %s %v", msg, l.sanitizeFields(fields...))
}

func (l *TestLogger) Warn(msg string, fields ...any) {
	l.t.Logf("[WARN] %s %v", msg, l.sanitizeFields(fields...))
}

func (l *TestLogger) Error(msg string, fields ...any) {
	l.t.Logf("[ERROR] %s %v", msg, l.sanitizeFields(fields...))
}

// sanitizeFields redacts sensitive information from log fields
func (l *TestLogger) sanitizeFields(fields ...any) []any {
	if len(fields) == 0 {
		return fields
	}

	sanitized := make([]any, len(fields))
	for i, field := range fields {
		switch v := field.(type) {
		case string:
			sanitized[i] = l.sanitizeString(v)
		default:
			sanitized[i] = field
		}
	}
	return sanitized
}

// sanitizeString redacts sensitive information from strings
func (l *TestLogger) sanitizeString(s string) string {
	// Don't redact in verbose mode (local development)
	if l.verbose {
		return s
	}

	// In CI, redact usernames and other sensitive patterns
	// This is a simple approach - in production you'd want more sophisticated redaction
	if len(s) > 0 && (s == "tirth.im" ||
		len(s) > 10 && (s[:4] == "@" || s[:5] == "http")) {
		return "[REDACTED]"
	}
	return s
}

// createTestClient creates a client using NewClientWithToken (preferred method when token exists)
func createTestClient(t *testing.T) *threads.Client {
	// Create configuration using public API
	config := &threads.Config{
		ClientID:     testClientID,
		ClientSecret: testSecret,
		RedirectURI:  testRedirectURI,
		HTTPTimeout:  30 * time.Second,
		Logger:       NewTestLogger(t),
	}

	// Create client with existing token using public API (simulates real usage)
	client, err := threads.NewClientWithToken(testAccessToken, config)
	if err != nil {
		t.Fatalf("Failed to create client with token: %v", err)
	}

	return client
}

// skipIfNoCredentials skips the test if credentials are not available
func skipIfNoCredentials(t *testing.T) {
	if testAccessToken == "" || testUserID == "" {
		t.Skip("Skipping integration test: no credentials available")
	}
}

// TestIntegration_Authentication tests basic authentication functionality
func TestIntegration_Authentication(t *testing.T) {
	skipIfNoCredentials(t)

	t.Run("ClientCreation", func(t *testing.T) {
		client := createTestClient(t)

		if !client.IsAuthenticated() {
			t.Error("Client should be authenticated")
		}

		if client.IsTokenExpired() {
			t.Error("Token should not be expired")
		}
	})

	t.Run("TokenValidation", func(t *testing.T) {
		client := createTestClient(t)

		err := client.ValidateToken()
		if err != nil {
			t.Errorf("Token validation failed: %v", err)
		}
	})

	t.Run("GetMe", func(t *testing.T) {
		client := createTestClient(t)

		user, err := client.GetMe(context.Background())
		if err != nil {
			t.Errorf("GetMe failed: %v", err)
			return
		}

		if user.ID != testUserID {
			t.Errorf("Expected user ID %s, got %s", testUserID, user.ID)
		}

		if user.Username == "" {
			t.Error("Username should not be empty")
		}

		// Log user info with sanitization
		username := user.Username
		if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
			username = "[REDACTED]"
		}
		t.Logf("User: ID=%s, Username=%s", user.ID, username)
	})
}

// TestIntegration_PostOperations tests basic post operations
func TestIntegration_PostOperations(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("GetUserPosts", func(t *testing.T) {
		posts, err := client.GetUserPosts(context.Background(), threads.ConvertToUserID(testUserID), &threads.PaginationOptions{
			Limit: 5,
		})

		if err != nil {
			t.Errorf("GetUserPosts failed: %v", err)
			return
		}

		t.Logf("Retrieved %d posts", len(posts.Data))

		for i, post := range posts.Data {
			if post.ID == "" {
				t.Errorf("Post %d has empty ID", i)
			}

			// Sanitize post text in CI environments
			postText := truncateString(post.Text, 50)
			if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
				postText = "[REDACTED]"
			}
			t.Logf("Post %d: ID=%s, Text=%s", i, post.ID, postText)
		}
	})

	t.Run("CreateAndDeleteTextPost", func(t *testing.T) {
		// Create a test post using public API
		content := &threads.TextPostContent{
			Text:         fmt.Sprintf("CI Integration test post created at %s", time.Now().Format(time.RFC3339)),
			ReplyControl: threads.ReplyControlEveryone,
		}

		post, err := client.CreateTextPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateTextPost failed: %v", err)
			return
		}

		if post.ID == "" {
			t.Error("Created post should have an ID")
		}

		if post.Text != content.Text {
			t.Errorf("Expected text %s, got %s", content.Text, post.Text)
		}

		t.Logf("Created post: ID=%s, Text=%s", post.ID, post.Text)

		// Test getting the specific post
		retrievedPost, err := client.GetPost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Errorf("GetPost failed: %v", err)
		} else if retrievedPost.ID != post.ID {
			t.Errorf("Retrieved post ID mismatch: expected %s, got %s", post.ID, retrievedPost.ID)
		}

		// Clean up - delete the test post using public API
		time.Sleep(2 * time.Second) // Wait a bit before deletion
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete test post %s: %v", post.ID, err)
		} else {
			t.Logf("Successfully deleted test post %s", post.ID)
		}
	})

	t.Run("CreateAndDeleteImagePost", func(t *testing.T) {
		if testImageURL1 == "" {
			t.Skip("Skipping image post test: THREADS_TEST_IMG1 not provided")
		}

		content := &threads.ImagePostContent{
			Text:     fmt.Sprintf("CI Integration test image post created at %s", time.Now().Format(time.RFC3339)),
			ImageURL: testImageURL1,
			AltText:  "Test image for CI integration testing",
		}

		post, err := client.CreateImagePost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateImagePost failed: %v", err)
			return
		}

		if post.ID == "" {
			t.Error("Created image post should have an ID")
		}

		if post.Text != content.Text {
			t.Errorf("Expected text %s, got %s", content.Text, post.Text)
		}

		t.Logf("Created image post: ID=%s, Text=%s, MediaType=%s", post.ID, post.Text, post.MediaType)

		// Clean up - delete the test post
		time.Sleep(2 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete image post %s: %v", post.ID, err)
		} else {
			t.Logf("Successfully deleted image post %s", post.ID)
		}
	})

	t.Run("CreateAndDeleteVideoPost", func(t *testing.T) {
		if testVideoURL == "" {
			t.Skip("Skipping video post test: THREADS_TEST_VID1 not provided")
		}

		content := &threads.VideoPostContent{
			Text:     fmt.Sprintf("CI Integration test video post created at %s", time.Now().Format(time.RFC3339)),
			VideoURL: testVideoURL,
			AltText:  "Test video for CI integration testing",
		}

		post, err := client.CreateVideoPost(context.Background(), content)
		if err != nil {
			// Video processing can be unpredictable - log the error but don't fail the test
			t.Logf("Video post creation failed (this is often due to video processing issues): %v", err)
			t.Skip("Skipping video post test due to processing failure - this is common with video URLs")
			return
		}

		if post.ID == "" {
			t.Error("Created video post should have an ID")
		}

		if post.Text != content.Text {
			t.Errorf("Expected text %s, got %s", content.Text, post.Text)
		}

		t.Logf("Created video post: ID=%s, Text=%s, MediaType=%s", post.ID, post.Text, post.MediaType)

		// Clean up - delete the test post
		time.Sleep(2 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete video post %s: %v", post.ID, err)
		} else {
			t.Logf("Successfully deleted video post %s", post.ID)
		}
	})

	t.Run("CreateAndDeleteCarouselPost", func(t *testing.T) {
		if testImageURL1 == "" || testImageURL2 == "" {
			t.Skip("Skipping carousel post test: THREADS_TEST_IMG1 or THREADS_TEST_IMG2 not provided")
		}

		// First create media containers for the carousel
		container1, err := client.CreateMediaContainer(context.Background(), "IMAGE", testImageURL1, "First carousel image")
		if err != nil {
			t.Errorf("Failed to create first media container: %v", err)
			return
		}
		t.Logf("Created first media container: %s", container1)

		container2, err := client.CreateMediaContainer(context.Background(), "IMAGE", testImageURL2, "Second carousel image")
		if err != nil {
			t.Errorf("Failed to create second media container: %v", err)
			return
		}
		t.Logf("Created second media container: %s", container2)

		// Wait a moment for containers to be ready
		time.Sleep(3 * time.Second)

		// Create carousel post
		content := &threads.CarouselPostContent{
			Text:     fmt.Sprintf("CI Integration test carousel post created at %s", time.Now().Format(time.RFC3339)),
			Children: []string{string(container1), string(container2)},
		}

		post, err := client.CreateCarouselPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateCarouselPost failed: %v", err)
			return
		}

		if post.ID == "" {
			t.Error("Created carousel post should have an ID")
		}

		if post.Text != content.Text {
			t.Errorf("Expected text %s, got %s", content.Text, post.Text)
		}

		t.Logf("Created carousel post: ID=%s, Text=%s, MediaType=%s", post.ID, post.Text, post.MediaType)

		// Clean up - delete the test post
		time.Sleep(2 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete carousel post %s: %v", post.ID, err)
		} else {
			t.Logf("Successfully deleted carousel post %s", post.ID)
		}
	})
}

// TestIntegration_SearchOperations tests search functionality
func TestIntegration_SearchOperations(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("KeywordSearch", func(t *testing.T) {
		results, err := client.KeywordSearch(context.Background(), "technology", &threads.SearchOptions{
			Limit:      3,
			SearchType: threads.SearchTypeTop,
		})
		if err != nil {
			t.Errorf("KeywordSearch failed: %v", err)
			return
		}

		t.Logf("Keyword search returned %d results", len(results.Data))

		// Check the results
		for i, post := range results.Data {
			// Sanitize search result text in CI environments
			resultText := truncateString(post.Text, 100)
			if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
				resultText = "[REDACTED]"
			}
			t.Logf("Result %d: ID=%s, Text=%s", i+1, post.ID, resultText)
		}
	})
}

// TestIntegration_RateLimiting tests rate limiting functionality
func TestIntegration_RateLimiting(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("RateLimitStatus", func(t *testing.T) {
		status := client.GetRateLimitStatus()

		t.Logf("Rate limit status: Limit=%d, Remaining=%d, Reset=%v",
			status.Limit, status.Remaining, status.ResetTime)

		if status.Limit <= 0 {
			t.Error("Rate limit should be positive")
		}
	})
}

// TestIntegration_ErrorHandling tests error handling
func TestIntegration_ErrorHandling(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("InvalidPostID", func(t *testing.T) {
		_, err := client.GetPost(context.Background(), threads.ConvertToPostID("invalid_post_id"))
		if err == nil {
			t.Error("Expected error for invalid post ID")
		}

		// Test that we can identify the error type using public API
		if !threads.IsAPIError(err) && !threads.IsValidationError(err) {
			t.Errorf("Expected API or validation error, got %T", err)
		}

		t.Logf("Invalid post ID error (expected): %v", err)
	})

	t.Run("EmptyTextPost", func(t *testing.T) {
		content := &threads.TextPostContent{
			Text: "", // Empty text should cause validation error
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected error for empty text post")
		}

		t.Logf("Empty text post error (expected): %v", err)
	})
}

// Helper functions

// truncateString truncates strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
