package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tirthpatell/threads-go"
)

func main() {
	// Example 1: Create client from environment variables
	fmt.Println("=== Example 1: Creating client from environment ===")

	// Set required environment variables (in production, set these in your deployment)
	// export THREADS_CLIENT_ID="your-client-id"
	// export THREADS_CLIENT_SECRET="your-client-secret"
	// export THREADS_REDIRECT_URI="https://yourapp.com/callback"

	client, err := threads.NewClientFromEnv()
	if err != nil {
		log.Printf("Failed to create client from env (this is expected in demo): %v", err)

		// Fallback: Create client manually for demo
		config := &threads.Config{
			ClientID:     getEnvOrDefault("THREADS_CLIENT_ID", "your-client-id"),
			ClientSecret: getEnvOrDefault("THREADS_CLIENT_SECRET", "your-client-secret"),
			RedirectURI:  getEnvOrDefault("THREADS_REDIRECT_URI", "https://yourapp.com/callback"),
			Scopes:       []string{"threads_basic", "threads_content_publish"},
			HTTPTimeout:  30 * time.Second,
		}

		client, err = threads.NewClient(config)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Example 2: OAuth Flow (when you don't have a token)
	fmt.Println("\n=== Example 2: OAuth Authorization Flow ===")

	scopes := []string{"threads_basic", "threads_content_publish", "threads_manage_insights"}
	authURL := client.GetAuthURL(scopes)
	fmt.Printf("1. Direct user to: %s\n", authURL)
	fmt.Printf("2. User authorizes and you get a code in your redirect URI\n")
	fmt.Printf("3. Exchange code for token:\n")
	fmt.Printf("   err := client.ExchangeCodeForToken(ctx, authorizationCode)\n")
	fmt.Printf("4. Convert to long-lived token:\n")
	fmt.Printf("   err := client.GetLongLivedToken(ctx)\n")

	// Example 3: Using existing access token
	fmt.Println("\n=== Example 3: Using existing access token ===")

	accessToken := os.Getenv("THREADS_ACCESS_TOKEN")
	if accessToken != "" {
		clientWithToken, err := threads.NewClientWithToken(accessToken, &threads.Config{
			ClientID:     getEnvOrDefault("THREADS_CLIENT_ID", "your-client-id"),
			ClientSecret: getEnvOrDefault("THREADS_CLIENT_SECRET", "your-client-secret"),
			RedirectURI:  getEnvOrDefault("THREADS_REDIRECT_URI", "https://yourapp.com/callback"),
		})
		if err != nil {
			log.Printf("Failed to create client with token: %v", err)
		} else {
			fmt.Println("✅ Successfully created client with existing token")

			// Example API calls (only if we have a real token)
			ctx := context.Background()

			// Get user info
			user, err := clientWithToken.GetMe(ctx)
			if err != nil {
				log.Printf("GetMe failed: %v", err)
			} else {
				fmt.Printf("✅ Authenticated as: %s (ID: %s)\n", user.Username, user.ID)
			}

			// Check publishing limits
			limits, err := clientWithToken.GetPublishingLimits(ctx)
			if err != nil {
				log.Printf("GetPublishingLimits failed: %v", err)
			} else {
				fmt.Printf("✅ Publishing limits: Posts=%d/%d, Replies=%d/%d\n",
					limits.QuotaUsage, limits.Config.QuotaTotal,
					limits.ReplyQuotaUsage, limits.ReplyConfig.QuotaTotal)
			}
		}
	} else {
		fmt.Println("ℹ️  Set THREADS_ACCESS_TOKEN environment variable to test API calls")
	}

	// Example 4: Creating different types of posts
	fmt.Println("\n=== Example 4: Creating posts (code examples) ===")

	fmt.Println("Text post:")
	fmt.Printf(`
post, err := client.CreateTextPost(ctx, &threads.TextPostContent{
    Text:         "Hello from Go SDK!",
    ReplyControl: threads.ReplyControlEveryone,
})
`)

	fmt.Println("Image post:")
	fmt.Printf(`
post, err := client.CreateImagePost(ctx, &threads.ImagePostContent{
    Text:     "Check out this image!",
    ImageURL: "https://example.com/image.jpg",
    AltText:  "A beautiful sunset",
})
`)

	fmt.Println("Quote post:")
	fmt.Printf(`
post, err := client.CreateQuotePost(ctx, &threads.QuotePostContent{
    Text:         "Adding my thoughts to this",
    QuotedPostID: "existing_post_id",
})
`)

	// Example 5: Error handling
	fmt.Println("\n=== Example 5: Error handling ===")

	fmt.Printf(`
if err != nil {
    switch {
    case threads.IsAuthenticationError(err):
        log.Println("Authentication failed - check your credentials")
        
    case threads.IsRateLimitError(err):
        rateLimitErr := err.(*threads.RateLimitError)
        log.Printf("Rate limited. Retry after: %%v", rateLimitErr.RetryAfter)
        
    case threads.IsValidationError(err):
        validationErr := err.(*threads.ValidationError)
        log.Printf("Validation error in field '%%s': %%s", validationErr.Field, err.Error())
        
    default:
        log.Printf("Unexpected error: %%v", err)
    }
}
`)

	// Example 6: Pagination
	fmt.Println("\n=== Example 6: Pagination with iterators ===")

	fmt.Printf(`
// Create iterator for user posts
userID := threads.ConvertToUserID("user_id")
iterator := threads.NewPostIterator(client, userID, &threads.PostsOptions{
    Limit: 25,
})

// Iterate through all pages
for iterator.HasNext() {
    response, err := iterator.Next(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, post := range response.Data {
        fmt.Printf("Post: %%s\n", post.Text)
    }
}
`)

	fmt.Println("\n🎉 Demo complete! Check the examples/ directory for more detailed examples.")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
