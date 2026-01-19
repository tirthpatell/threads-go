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
		time.Sleep(1 * time.Second) // Wait a bit before deletion
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
		time.Sleep(1 * time.Second)
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
		time.Sleep(1 * time.Second)
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

		// Note: No sleep needed - CreateCarouselPost waits for child containers to be ready

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
		time.Sleep(1 * time.Second)
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

// TestIntegration_SpoilersAndTextAttachments tests spoilers and text attachments features
func TestIntegration_SpoilersAndTextAttachments(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("TextPostWithSpoilers", func(t *testing.T) {
		// Test text spoilers using text_entities
		content := &threads.TextPostContent{
			Text: "Spoiler alert: Darth Vader is Luke's father!",
			TextEntities: []threads.TextEntity{
				{
					EntityType: "SPOILER",
					Offset:     15, // Start of "Darth Vader is Luke's father!" (after "Spoiler alert: ")
					Length:     11, // Just cover "Darth Vader" for testing
				},
			},
		}

		post, err := client.CreateTextPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateTextPost with spoilers failed: %v", err)
			return
		}

		if post.ID == "" {
			t.Error("Created post should have an ID")
		}

		t.Logf("Created post with text spoiler: ID=%s, Text=%s", post.ID, post.Text)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete spoiler post %s: %v", post.ID, err)
		} else {
			t.Logf("Successfully deleted spoiler post %s", post.ID)
		}
	})

	t.Run("TextPostWithMultipleSpoilers", func(t *testing.T) {
		// Test multiple text spoilers
		content := &threads.TextPostContent{
			Text: "Two spoilers: Han dies and Rey is a Palpatine!",
			TextEntities: []threads.TextEntity{
				{
					EntityType: "SPOILER",
					Offset:     14, // "Han dies"
					Length:     8,
				},
				{
					EntityType: "SPOILER",
					Offset:     27, // "Rey is a Palpatine"
					Length:     18,
				},
			},
		}

		post, err := client.CreateTextPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateTextPost with multiple spoilers failed: %v", err)
			return
		}

		t.Logf("Created post with multiple spoilers: ID=%s", post.ID)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete multi-spoiler post %s: %v", post.ID, err)
		}
	})

	t.Run("ImagePostWithMediaSpoiler", func(t *testing.T) {
		if testImageURL1 == "" {
			t.Skip("Skipping image spoiler test: THREADS_TEST_IMG1 not provided")
		}

		// Test media spoiler with image
		content := &threads.ImagePostContent{
			Text:           fmt.Sprintf("CI test image spoiler created at %s", time.Now().Format(time.RFC3339)),
			ImageURL:       testImageURL1,
			AltText:        "Spoiler image",
			IsSpoilerMedia: true, // Mark the image as a spoiler
		}

		post, err := client.CreateImagePost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateImagePost with media spoiler failed: %v", err)
			return
		}

		t.Logf("Created image post with media spoiler: ID=%s", post.ID)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete image spoiler post %s: %v", post.ID, err)
		}
	})

	t.Run("ImagePostWithTextAndMediaSpoilers", func(t *testing.T) {
		if testImageURL1 == "" {
			t.Skip("Skipping combined spoiler test: THREADS_TEST_IMG1 not provided")
		}

		// Test both text and media spoilers
		content := &threads.ImagePostContent{
			Text:     "Spoiler: This image reveals the ending!",
			ImageURL: testImageURL1,
			TextEntities: []threads.TextEntity{
				{
					EntityType: "SPOILER",
					Offset:     9,  // Start of "This image reveals the ending!" (after "Spoiler: ")
					Length:     30, // Length of "This image reveals the ending!"
				},
			},
			IsSpoilerMedia: true,
		}

		post, err := client.CreateImagePost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateImagePost with text and media spoilers failed: %v", err)
			return
		}

		t.Logf("Created post with text and media spoilers: ID=%s", post.ID)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete combined spoiler post %s: %v", post.ID, err)
		}
	})

	t.Run("TextPostWithTextAttachment", func(t *testing.T) {
		// Test text attachment with styling
		content := &threads.TextPostContent{
			Text: fmt.Sprintf("CI test post with text attachment at %s", time.Now().Format(time.RFC3339)),
			TextAttachment: &threads.TextAttachment{
				Plaintext: "This is a long-form text attachment with up to 10,000 characters. " +
					"It supports rich formatting and allows you to share detailed content beyond the 500 character limit. " +
					"This is perfect for sharing articles, stories, or detailed explanations.",
				TextWithStylingInfo: []threads.TextStylingInfo{
					{
						Offset:      0,
						Length:      7, // "This is"
						StylingInfo: []string{"bold"},
					},
					{
						Offset:      10,
						Length:      9, // "long-form"
						StylingInfo: []string{"italic"},
					},
				},
			},
		}

		post, err := client.CreateTextPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateTextPost with text attachment failed: %v", err)
			return
		}

		t.Logf("Created post with text attachment: ID=%s", post.ID)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete text attachment post %s: %v", post.ID, err)
		}
	})

	t.Run("TextAttachmentWithLink", func(t *testing.T) {
		// Test text attachment with link
		content := &threads.TextPostContent{
			Text: "Check out my detailed post with a link!",
			TextAttachment: &threads.TextAttachment{
				Plaintext:         "Here's a detailed explanation with additional information that couldn't fit in the main post. This text attachment includes a link for more details.",
				LinkAttachmentURL: "https://example.com/more-info",
			},
		}

		post, err := client.CreateTextPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateTextPost with text attachment link failed: %v", err)
			return
		}

		t.Logf("Created post with text attachment and link: ID=%s", post.ID)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete text attachment link post %s: %v", post.ID, err)
		}
	})

	t.Run("CarouselWithMediaSpoiler", func(t *testing.T) {
		if testImageURL1 == "" || testImageURL2 == "" {
			t.Skip("Skipping carousel spoiler test: THREADS_TEST_IMG1 or THREADS_TEST_IMG2 not provided")
		}

		// Create media containers
		container1, err := client.CreateMediaContainer(context.Background(), "IMAGE", testImageURL1, "First carousel image")
		if err != nil {
			t.Errorf("Failed to create first media container: %v", err)
			return
		}

		container2, err := client.CreateMediaContainer(context.Background(), "IMAGE", testImageURL2, "Second carousel image")
		if err != nil {
			t.Errorf("Failed to create second media container: %v", err)
			return
		}

		// Note: No sleep needed - CreateCarouselPost waits for child containers to be ready

		// Create carousel with all media marked as spoilers
		content := &threads.CarouselPostContent{
			Text:           fmt.Sprintf("CI test carousel with spoilers at %s", time.Now().Format(time.RFC3339)),
			Children:       []string{string(container1), string(container2)},
			IsSpoilerMedia: true, // Marks ALL carousel media as spoilers
		}

		post, err := client.CreateCarouselPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateCarouselPost with spoilers failed: %v", err)
			return
		}

		t.Logf("Created carousel with spoiler media: ID=%s", post.ID)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete carousel spoiler post %s: %v", post.ID, err)
		}
	})
}

// TestIntegration_ContainerStatus tests the GetContainerStatus method
func TestIntegration_ContainerStatus(t *testing.T) {
	skipIfNoCredentials(t)

	if testImageURL1 == "" {
		t.Skip("Skipping container status test: THREADS_TEST_IMG1 not provided")
	}

	client := createTestClient(t)

	t.Run("GetContainerStatus", func(t *testing.T) {
		// Create a media container
		containerID, err := client.CreateMediaContainer(context.Background(), "IMAGE", testImageURL1, "Test image for container status")
		if err != nil {
			t.Errorf("Failed to create media container: %v", err)
			return
		}

		t.Logf("Created media container: %s", containerID)

		// Get the container status
		status, err := client.GetContainerStatus(context.Background(), containerID)
		if err != nil {
			t.Errorf("GetContainerStatus failed: %v", err)
			return
		}

		if status.ID == "" {
			t.Error("Container status should have an ID")
		}

		if status.Status == "" {
			t.Error("Container status should have a status field")
		}

		t.Logf("Container status: ID=%s, Status=%s, ErrorMessage=%s",
			status.ID, status.Status, status.ErrorMessage)

		// Verify status is one of the valid values
		validStatuses := map[string]bool{
			"IN_PROGRESS": true,
			"FINISHED":    true,
			"PUBLISHED":   true,
			"ERROR":       true,
			"EXPIRED":     true,
		}

		if !validStatuses[status.Status] {
			t.Errorf("Invalid container status: %s", status.Status)
		}
	})
}

// TestIntegration_GhostPosts tests ghost post functionality
func TestIntegration_GhostPosts(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("GetUserGhostPosts", func(t *testing.T) {
		// Just test the endpoint call, we might not have any ghost posts
		posts, err := client.GetUserGhostPosts(context.Background(), threads.ConvertToUserID(testUserID), &threads.PaginationOptions{
			Limit: 5,
		})

		if err != nil {
			t.Errorf("GetUserGhostPosts failed: %v", err)
			return
		}

		t.Logf("Retrieved %d ghost posts", len(posts.Data))

		for i, post := range posts.Data {
			if post.ID == "" {
				t.Errorf("Ghost post %d has empty ID", i)
			}
			t.Logf("Ghost Post %d: ID=%s, Status=%s, Expires=%v", i, post.ID, post.GhostPostStatus, post.GhostPostExpirationTimestamp)
		}
	})
}

// TestIntegration_ValidationErrors tests validation for new features
func TestIntegration_ValidationErrors(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("TooManyTextEntities", func(t *testing.T) {
		// Try to create post with more than 10 text entities (should fail validation)
		entities := make([]threads.TextEntity, 11)
		for i := 0; i < 11; i++ {
			entities[i] = threads.TextEntity{
				EntityType: "SPOILER",
				Offset:     i * 5,
				Length:     3,
			}
		}

		content := &threads.TextPostContent{
			Text:         "This post has too many spoiler entities and should fail validation",
			TextEntities: entities,
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for too many text entities")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("TextAttachmentWithPoll", func(t *testing.T) {
		// Try to create post with both text attachment and poll (should fail validation)
		content := &threads.TextPostContent{
			Text: "This should fail validation",
			PollAttachment: &threads.PollAttachment{
				OptionA: "Option A",
				OptionB: "Option B",
			},
			TextAttachment: &threads.TextAttachment{
				Plaintext: "This should not be allowed with a poll",
			},
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for text attachment with poll")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("InvalidTextEntityOffset", func(t *testing.T) {
		// Try to create post with negative offset (should fail validation)
		content := &threads.TextPostContent{
			Text: "This should fail validation",
			TextEntities: []threads.TextEntity{
				{
					EntityType: "SPOILER",
					Offset:     -1, // Invalid negative offset
					Length:     5,
				},
			},
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for negative offset")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("EmptyTextAttachmentPlaintext", func(t *testing.T) {
		// Try to create post with empty text attachment plaintext (should fail validation)
		content := &threads.TextPostContent{
			Text: "This should fail validation",
			TextAttachment: &threads.TextAttachment{
				Plaintext: "", // Empty plaintext is not allowed
			},
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for empty text attachment plaintext")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("GhostPostAsReply", func(t *testing.T) {
		// Try to create ghost post as a reply (should fail validation)
		content := &threads.TextPostContent{
			Text:        "This should fail validation",
			IsGhostPost: true,
			ReplyTo:     "some_post_id",
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for ghost post as reply")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("TooManyLinks", func(t *testing.T) {
		// Try to create post with more than 5 unique links (should fail validation)
		// Added December 22, 2025: THREADS_API__LINK_LIMIT_EXCEEDED
		content := &threads.TextPostContent{
			Text: "Check out these links: https://example1.com https://example2.com https://example3.com https://example4.com https://example5.com https://example6.com",
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for too many links (max 5)")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("TooManyLinksWithLinkAttachment", func(t *testing.T) {
		// Try to create post with 5 links in text + 1 different link_attachment (should fail validation)
		content := &threads.TextPostContent{
			Text:           "Links: https://example1.com https://example2.com https://example3.com https://example4.com https://example5.com",
			LinkAttachment: "https://example6.com", // 6th unique link
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for too many links with link_attachment")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("LinksWithDuplicateLinkAttachment", func(t *testing.T) {
		// 5 unique links in text + link_attachment that duplicates one = should pass (5 unique total)
		// This tests that duplicate link_attachment is not double-counted
		content := &threads.TextPostContent{
			Text:           "Links: https://example1.com https://example2.com https://example3.com https://example4.com https://example5.com",
			LinkAttachment: "https://example1.com", // Duplicate, should not count as 6th
		}

		// This should NOT fail validation (5 unique links is the limit)
		err := client.ValidateTextPostContent(content)
		if err != nil {
			t.Errorf("Unexpected validation error for 5 unique links with duplicate link_attachment: %v", err)
		} else {
			t.Log("Validation passed (expected): 5 unique links with duplicate link_attachment")
		}
	})
}

// TestIntegration_GIFPosts tests GIF attachment functionality
func TestIntegration_GIFPosts(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	// Tenor GIF ID for testing
	testTenorGIFID := "11366929630539488910"

	t.Run("CreateAndDeleteGIFPost", func(t *testing.T) {
		content := &threads.TextPostContent{
			Text: fmt.Sprintf("CI Integration test GIF post created at %s", time.Now().Format(time.RFC3339)),
			GIFAttachment: &threads.GIFAttachment{
				GIFID:    testTenorGIFID,
				Provider: threads.GIFProviderTenor,
			},
		}

		post, err := client.CreateTextPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateTextPost with GIF attachment failed: %v", err)
			return
		}

		if post.ID == "" {
			t.Error("Created GIF post should have an ID")
		}

		t.Logf("Created GIF post: ID=%s, Text=%s", post.ID, post.Text)

		// Verify the post has a GIF URL (if returned by API)
		if post.GifURL != "" {
			t.Logf("GIF URL: %s", post.GifURL)
		}

		// Clean up - delete the test post
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete GIF post %s: %v", post.ID, err)
		} else {
			t.Logf("Successfully deleted GIF post %s", post.ID)
		}
	})

	t.Run("GIFPostWithReplyControl", func(t *testing.T) {
		content := &threads.TextPostContent{
			Text:         fmt.Sprintf("CI test GIF post with reply control at %s", time.Now().Format(time.RFC3339)),
			ReplyControl: threads.ReplyControlFollowersOnly,
			GIFAttachment: &threads.GIFAttachment{
				GIFID:    testTenorGIFID,
				Provider: threads.GIFProviderTenor,
			},
		}

		post, err := client.CreateTextPost(context.Background(), content)
		if err != nil {
			t.Errorf("CreateTextPost with GIF and reply control failed: %v", err)
			return
		}

		t.Logf("Created GIF post with reply control: ID=%s", post.ID)

		// Clean up
		time.Sleep(1 * time.Second)
		err = client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID))
		if err != nil {
			t.Logf("Warning: Failed to delete GIF post %s: %v", post.ID, err)
		}
	})
}

// TestIntegration_GIFValidationErrors tests validation errors for GIF attachments
func TestIntegration_GIFValidationErrors(t *testing.T) {
	skipIfNoCredentials(t)

	client := createTestClient(t)

	t.Run("EmptyGIFID", func(t *testing.T) {
		content := &threads.TextPostContent{
			Text: "This should fail validation - empty GIF ID",
			GIFAttachment: &threads.GIFAttachment{
				GIFID:    "",
				Provider: threads.GIFProviderTenor,
			},
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for empty GIF ID")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("EmptyGIFProvider", func(t *testing.T) {
		content := &threads.TextPostContent{
			Text: "This should fail validation - empty provider",
			GIFAttachment: &threads.GIFAttachment{
				GIFID:    "11366929630539488910",
				Provider: "",
			},
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for empty GIF provider")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
	})

	t.Run("InvalidGIFProvider", func(t *testing.T) {
		content := &threads.TextPostContent{
			Text: "This should fail validation - invalid provider",
			GIFAttachment: &threads.GIFAttachment{
				GIFID:    "11366929630539488910",
				Provider: threads.GIFProvider("GIPHY"), // Invalid provider
			},
		}

		_, err := client.CreateTextPost(context.Background(), content)
		if err == nil {
			t.Error("Expected validation error for invalid GIF provider")
		} else {
			t.Logf("Validation error (expected): %v", err)
		}
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
