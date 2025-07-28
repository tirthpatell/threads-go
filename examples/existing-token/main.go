// Package main demonstrates how to use the Threads API client with an existing access token.
//
// This example shows how to:
// 1. Create a client with an existing token (skip OAuth flow)
// 2. Validate the token automatically
// 3. Use the client immediately for API calls
// 4. Handle token-related errors
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tirthpatell/threads-go"
)

func main() {
	fmt.Println("Threads API Existing Token Example")
	fmt.Println("==================================")
	fmt.Println()

	// Method 1: Using token from environment (application manages env vars)
	fmt.Println("Method 1: Using token from environment")
	fmt.Println("--------------------------------------")

	// Your application reads the token from environment variables
	accessTokenFromEnv := os.Getenv("THREADS_ACCESS_TOKEN")

	if accessTokenFromEnv != "" {
		config := &threads.Config{
			ClientID:     os.Getenv("THREADS_CLIENT_ID"),
			ClientSecret: os.Getenv("THREADS_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("THREADS_REDIRECT_URI"),
		}

		// Validate required config
		if config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
			fmt.Println("Please set THREADS_CLIENT_ID, THREADS_CLIENT_SECRET, and THREADS_REDIRECT_URI")
		} else {
			client, err := threads.NewClientWithToken(accessTokenFromEnv, config)
			if err != nil {
				fmt.Printf("Failed to create client with environment token: %v\n", err)

				// Handle specific error types
				if threads.IsAuthenticationError(err) {
					fmt.Println("The access token might be invalid or expired")
				}
			} else {
				fmt.Println("Client created successfully with token from environment")
				demonstrateUsage(client)
			}
		}
	} else {
		fmt.Println("THREADS_ACCESS_TOKEN not set, skipping environment method")
	}

	fmt.Println()

	// Method 2: Direct token usage
	fmt.Println("Method 2: Direct token usage")
	fmt.Println("----------------------------")

	// Replace with your actual token for testing
	accessToken := "your-access-token-here"

	if accessToken == "your-access-token-here" {
		fmt.Println("Please replace 'your-access-token-here' with your actual token")
		fmt.Println("Or set the THREADS_ACCESS_TOKEN environment variable")
		return
	}

	config := &threads.Config{
		ClientID:     os.Getenv("THREADS_CLIENT_ID"),
		ClientSecret: os.Getenv("THREADS_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("THREADS_REDIRECT_URI"),
	}

	// Validate required config
	if config.ClientID == "" || config.ClientSecret == "" || config.RedirectURI == "" {
		fmt.Println("Please set THREADS_CLIENT_ID, THREADS_CLIENT_SECRET, and THREADS_REDIRECT_URI")
		return
	}

	client, err := threads.NewClientWithToken(accessToken, config)
	if err != nil {
		fmt.Printf("Failed to create client with token: %v\n", err)

		// Handle specific error types
		if threads.IsAuthenticationError(err) {
			fmt.Println("The access token might be invalid or expired")
		} else if threads.IsValidationError(err) {
			fmt.Println("There might be an issue with the token format or configuration")
		}
		return
	}

	fmt.Println("Client created successfully with direct token")
	demonstrateUsage(client)
}

func demonstrateUsage(client *threads.Client) {
	ctx := context.Background()
	fmt.Println()
	fmt.Println("Demonstrating client usage:")
	fmt.Println("---------------------------")

	// Test 1: Get current user information
	fmt.Println("1. Getting current user information...")
	user, err := client.GetMe(ctx)
	if err != nil {
		fmt.Printf("   Failed to get user info: %v\n", err)
		return
	}

	fmt.Printf("   Authenticated as: %s (@%s)\n", user.Name, user.Username)
	if user.Biography != "" {
		fmt.Printf("   Bio: %s\n", user.Biography)
	}

	// Test 2: Check token information
	fmt.Println("\n2. Checking token information...")
	tokenInfo := client.GetTokenInfo()
	if tokenInfo != nil {
		fmt.Printf("   Token Type: %s\n", tokenInfo.TokenType)
		fmt.Printf("   User ID: %s\n", tokenInfo.UserID)
		fmt.Printf("   Expires At: %s\n", tokenInfo.ExpiresAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Created At: %s\n", tokenInfo.CreatedAt.Format("2006-01-02 15:04:05"))

		// Check token status
		fmt.Printf("   Is Expired: %t\n", client.IsTokenExpired())
		fmt.Printf("   Is Authenticated: %t\n", client.IsAuthenticated())
	}

	// Test 3: Get recent posts
	fmt.Println("\n3. Getting recent posts...")
	userID := threads.ConvertToUserID(user.ID)
	posts, err := client.GetUserPosts(ctx, userID, &threads.PaginationOptions{
		Limit: 5,
	})
	if err != nil {
		fmt.Printf("   Failed to get posts: %v\n", err)
	} else {
		fmt.Printf("   Found %d recent posts\n", len(posts.Data))
		for i, post := range posts.Data {
			if i >= 3 { // Show only first 3
				break
			}
			fmt.Printf("   Post %d: %s\n", i+1, truncateText(post.Text, 60))
		}
	}

	// Test 4: Check publishing limits
	fmt.Println("\n4. Checking publishing limits...")
	limits, err := client.GetPublishingLimits(ctx)
	if err != nil {
		fmt.Printf("   Failed to get limits: %v\n", err)
	} else {
		fmt.Printf("   Posts quota: %d/%d used\n", limits.QuotaUsage, limits.Config.QuotaTotal)
		fmt.Printf("   Replies quota: %d/%d used\n", limits.ReplyQuotaUsage, limits.ReplyConfig.QuotaTotal)
	}

	fmt.Println("\nClient is working correctly with the existing token!")
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
