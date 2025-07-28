// Package main demonstrates various post creation methods using the Threads API.
//
// This example shows how to:
// 1. Create text posts with different options
// 2. Create image and video posts
// 3. Create carousel posts with multiple media
// 4. Create quote posts and reposts
// 5. Handle post creation errors
// 6. Work with post metadata and options
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/tirthpatell/threads-go"
)

func main() {
	fmt.Println("Threads API Post Creation Examples")
	fmt.Println("==================================")
	fmt.Println()

	// Create client from environment variables
	client, err := threads.NewClientFromEnv()
	if err != nil {
		log.Fatalf("Failed to create client: %v\nMake sure to set THREADS_CLIENT_ID, THREADS_CLIENT_SECRET, and THREADS_REDIRECT_URI", err)
	}

	// Check if we're authenticated
	if !client.IsAuthenticated() {
		fmt.Println("Client is not authenticated")
		fmt.Println("Run the authentication example first to get a token")
		fmt.Println("   Or set THREADS_ACCESS_TOKEN environment variable")
		return
	}

	fmt.Println("Client authenticated and ready")
	fmt.Println()

	// Example 1: Simple text post
	fmt.Println("Example 1: Simple Text Post")
	fmt.Println("===========================")
	createSimpleTextPost(client)
	fmt.Println()

	// Example 2: Text post with advanced options
	fmt.Println("Example 2: Advanced Text Post")
	fmt.Println("=============================")
	createAdvancedTextPost(client)
	fmt.Println()

	// Example 3: Image post
	fmt.Println("Example 3: Image Post")
	fmt.Println("====================")
	createImagePost(client)
	fmt.Println()

	// Example 4: Video post
	fmt.Println("Example 4: Video Post")
	fmt.Println("====================")
	createVideoPost(client)
	fmt.Println()

	// Example 5: Carousel post
	fmt.Println("Example 5: Carousel Post")
	fmt.Println("=======================")
	createCarouselPost(client)
	fmt.Println()

	// Example 6: Quote post
	fmt.Println("Example 6: Quote Post")
	fmt.Println("====================")
	createQuotePost(client)
	fmt.Println()

	// Example 7: Repost
	fmt.Println("Example 7: Repost")
	fmt.Println("================")
	createRepost(client)
	fmt.Println()

	// Example 8: Error handling
	fmt.Println("Example 8: Error Handling")
	fmt.Println("========================")
	demonstrateErrorHandling(client)
	fmt.Println()

	fmt.Println("Post creation examples completed!")
}

func createSimpleTextPost(client *threads.Client) {
	ctx := context.Background()
	content := &threads.TextPostContent{
		Text: "Hello from the Threads Go client!\n\nThis is a simple text post created using the API.",
	}

	post, err := client.CreateTextPost(ctx, content)
	if err != nil {
		fmt.Printf("Failed to create text post: %v\n", err)
		handlePostError(err)
		return
	}

	fmt.Println("Text post created successfully!")
	printPostInfo(post)
}

func createAdvancedTextPost(client *threads.Client) {
	ctx := context.Background()
	content := &threads.TextPostContent{
		Text:           "Check out the Threads API documentation for developers!\n\nPerfect for building integrations and automating your Threads presence.",
		LinkAttachment: "https://developers.facebook.com/docs/threads",
		ReplyControl:   threads.ReplyControlAccountsYouFollow,
		TopicTag:       "ThreadsAPI",
		// AutoPublishText: true, // Uncomment to use direct publishing
	}

	post, err := client.CreateTextPost(ctx, content)
	if err != nil {
		fmt.Printf("Failed to create advanced text post: %v\n", err)
		handlePostError(err)
		return
	}

	fmt.Println("Advanced text post created successfully!")
	fmt.Printf("   Link attachment: %s\n", content.LinkAttachment)
	fmt.Printf("   Reply control: %s\n", content.ReplyControl)
	fmt.Printf("   Topic tag: %s\n", content.TopicTag)
	printPostInfo(post)
}

func createImagePost(client *threads.Client) {
	ctx := context.Background()
	// Example image URL - replace with your own
	imageURL := "https://picsum.photos/800/600?random=1"

	content := &threads.ImagePostContent{
		Text:         "Beautiful image shared via the Threads API!\n\nThis demonstrates image posting capabilities.",
		ImageURL:     imageURL,
		AltText:      "A randomly generated beautiful image from Picsum",
		ReplyControl: threads.ReplyControlEveryone,
	}

	post, err := client.CreateImagePost(ctx, content)
	if err != nil {
		fmt.Printf(" Failed to create image post: %v\n", err)
		handlePostError(err)
		return
	}

	fmt.Println(" Image post created successfully!")
	fmt.Printf("   Image URL: %s\n", content.ImageURL)
	fmt.Printf("   Alt text: %s\n", content.AltText)
	printPostInfo(post)
}

func createVideoPost(client *threads.Client) {
	ctx := context.Background()
	// Example video URL - replace with your own
	videoURL := "https://sample-videos.com/zip/10/mp4/SampleVideo_1280x720_1mb.mp4"

	content := &threads.VideoPostContent{
		Text:         "Amazing video content shared via the Threads API!\n\nVideo posts are great for engagement.",
		VideoURL:     videoURL,
		AltText:      "A sample video demonstrating video post capabilities",
		ReplyControl: threads.ReplyControlEveryone,
	}

	fmt.Println("⏳ Creating video post (this may take longer due to processing)...")

	post, err := client.CreateVideoPost(ctx, content)
	if err != nil {
		fmt.Printf(" Failed to create video post: %v\n", err)
		handlePostError(err)
		return
	}

	fmt.Println(" Video post created successfully!")
	fmt.Printf("   Video URL: %s\n", content.VideoURL)
	fmt.Printf("   Alt text: %s\n", content.AltText)
	printPostInfo(post)
}

func createCarouselPost(_ *threads.Client) {
	fmt.Println(" Note: Carousel posts require pre-created container IDs")
	fmt.Println("   In a real application, you would:")
	fmt.Println("   1. Create individual media containers first")
	fmt.Println("   2. Use their IDs in the carousel post")
	fmt.Println("   3. This example shows the structure")

	// In a real application, you would create containers first:
	// container1, err := client.createImageContainer(...)
	// container2, err := client.createVideoContainer(...)

	// For demonstration, we'll show what the call would look like
	content := &threads.CarouselPostContent{
		Text: "Amazing carousel post with multiple media items!\n\nSwipe to see more content.",
		Children: []string{
			"container_id_1", // These would be real container IDs
			"container_id_2",
			"container_id_3",
		},
		ReplyControl: threads.ReplyControlEveryone,
	}

	fmt.Printf(" Carousel post structure:\n")
	fmt.Printf("   Text: %s\n", content.Text)
	fmt.Printf("   Children: %v\n", content.Children)
	fmt.Printf("   Reply control: %s\n", content.ReplyControl)

	// Uncomment to actually create (requires real container IDs):
	// post, err := client.CreateCarouselPost(content)
	// if err != nil {
	//     fmt.Printf(" Failed to create carousel post: %v\n", err)
	//     handlePostError(err)
	//     return
	// }
	// fmt.Println(" Carousel post created successfully!")
	// printPostInfo(post)

	fmt.Println(" This example shows the structure - implement container creation for full functionality")
}

func createQuotePost(_ *threads.Client) {
	fmt.Println(" Note: Quote posts require an existing post ID to quote")
	fmt.Println("   In a real application, you would have a specific post ID")

	// Example post ID - replace with a real post ID
	quotedPostID := "example_post_id_to_quote"

	content := &threads.TextPostContent{
		Text:         "Adding my thoughts to this great post!\n\nQuote posts are perfect for commentary and discussion.",
		QuotedPostID: quotedPostID,
		ReplyControl: threads.ReplyControlEveryone,
	}

	fmt.Printf(" Quote post structure:\n")
	fmt.Printf("   Text: %s\n", content.Text)
	fmt.Printf("   Quoted post ID: %s\n", content.QuotedPostID)
	fmt.Printf("   Reply control: %s\n", content.ReplyControl)

	// Uncomment to actually create (requires real post ID):
	// post, err := client.CreateQuotePost(content)
	// if err != nil {
	//     fmt.Printf(" Failed to create quote post: %v\n", err)
	//     handlePostError(err)
	//     return
	// }
	// fmt.Println(" Quote post created successfully!")
	// printPostInfo(post)

	fmt.Println(" Replace 'example_post_id_to_quote' with a real post ID to create quote posts")
}

func createRepost(_ *threads.Client) {
	fmt.Println(" Note: Reposts require an existing post ID to repost")
	fmt.Println("   This is similar to retweeting on Twitter")

	// Example post ID - replace with a real post ID
	postIDToRepost := "example_post_id_to_repost"

	fmt.Printf(" Repost structure:\n")
	fmt.Printf("   Post ID to repost: %s\n", postIDToRepost)

	// Uncomment to actually create (requires real post ID):
	// post, err := client.RepostPost(postIDToRepost)
	// if err != nil {
	//     fmt.Printf(" Failed to create repost: %v\n", err)
	//     handlePostError(err)
	//     return
	// }
	// fmt.Println(" Repost created successfully!")
	// printPostInfo(post)

	fmt.Println(" Replace 'example_post_id_to_repost' with a real post ID to create reposts")
}

func demonstrateErrorHandling(client *threads.Client) {
	ctx := context.Background()
	fmt.Println("Testing various error scenarios...")

	// Test 1: Empty text post
	fmt.Println("\nTest 1: Empty text post")
	emptyContent := &threads.TextPostContent{
		Text: "", // Empty text should cause validation error
	}

	_, err := client.CreateTextPost(ctx, emptyContent)
	if err != nil {
		fmt.Printf(" Correctly caught error: %v\n", err)
		handlePostError(err)
	} else {
		fmt.Println(" Unexpected success with empty text")
	}

	// Test 2: Invalid image URL
	fmt.Println("\n Test 2: Invalid image URL")
	invalidImageContent := &threads.ImagePostContent{
		Text:     "Test post with invalid image",
		ImageURL: "not-a-valid-url",
	}

	_, err = client.CreateImagePost(ctx, invalidImageContent)
	if err != nil {
		fmt.Printf(" Correctly caught error: %v\n", err)
		handlePostError(err)
	} else {
		fmt.Println(" Unexpected success with invalid image URL")
	}

	// Test 3: Text too long (if there are limits)
	fmt.Println("\n Test 3: Very long text")
	longText := ""
	for i := 0; i < 1000; i++ {
		longText += "This is a very long text post that might exceed API limits. "
	}

	longTextContent := &threads.TextPostContent{
		Text: longText,
	}

	_, err = client.CreateTextPost(ctx, longTextContent)
	if err != nil {
		fmt.Printf("Correctly caught error: %v\n", err)
		handlePostError(err)
	} else {
		fmt.Println("Long text post succeeded (no length limit or limit not reached)")
	}
}

func handlePostError(err error) {
	switch {
	case threads.IsValidationError(err):
		var validationErr *threads.ValidationError
		errors.As(err, &validationErr)
		fmt.Printf("    Validation error in field '%s': %s\n", validationErr.Field, validationErr.Message)

	case threads.IsAuthenticationError(err):
		fmt.Println("    Authentication error - check your token")

	case threads.IsRateLimitError(err):
		var rateLimitErr *threads.RateLimitError
		errors.As(err, &rateLimitErr)
		fmt.Printf("    Rate limit error - retry after %v\n", rateLimitErr.RetryAfter)

	case threads.IsNetworkError(err):
		var networkErr *threads.NetworkError
		errors.As(err, &networkErr)
		if networkErr.Temporary {
			fmt.Println("    Temporary network error - retry might succeed")
		} else {
			fmt.Println("    Permanent network error - check connectivity")
		}

	case threads.IsAPIError(err):
		var apiErr *threads.APIError
		errors.As(err, &apiErr)
		fmt.Printf("    API error (Request ID: %s): %s\n", apiErr.RequestID, apiErr.Message)

	default:
		fmt.Printf("    Unknown error type: %T\n", err)
	}
}

func printPostInfo(post *threads.Post) {
	fmt.Printf("   Post ID: %s\n", post.ID)
	if post.Text != "" {
		// Truncate long text for display
		text := post.Text
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		fmt.Printf("   Text: %s\n", text)
	}
	if post.MediaType != "" {
		fmt.Printf("   Media Type: %s\n", post.MediaType)
	}
	if post.MediaURL != "" {
		fmt.Printf("   Media URL: %s\n", post.MediaURL)
	}
	fmt.Printf("   Permalink: %s\n", post.Permalink)
	fmt.Printf("   Username: %s\n", post.Username)
	fmt.Printf("   Timestamp: %s\n", post.Timestamp.Format(time.RFC3339))
	fmt.Printf("   Is Reply: %t\n", post.IsReply)
	if post.IsQuotePost {
		fmt.Printf("   Is Quote Post: %t\n", post.IsQuotePost)
	}
}
