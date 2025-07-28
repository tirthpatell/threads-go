// Package main demonstrates the complete OAuth 2.0 authentication flow
// for the Threads API using the threads-go client library.
//
// This example shows how to:
// 1. Set up the client with configuration
// 2. Generate authorization URLs
// 3. Exchange authorization codes for tokens
// 4. Convert to long-lived tokens
// 5. Handle token storage and refresh
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tirthpatell/threads-go"
)

// SimpleLogger implements the threads.Logger interface for demonstration
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(msg string, fields ...any) {
	fmt.Printf("[DEBUG] %s", msg)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fmt.Printf(" %v=%v", fields[i], fields[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Info(msg string, fields ...any) {
	fmt.Printf("[INFO] %s", msg)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fmt.Printf(" %v=%v", fields[i], fields[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Warn(msg string, fields ...any) {
	fmt.Printf("[WARN] %s", msg)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fmt.Printf(" %v=%v", fields[i], fields[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Error(msg string, fields ...any) {
	fmt.Printf("[ERROR] %s", msg)
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			fmt.Printf(" %v=%v", fields[i], fields[i+1])
		}
	}
	fmt.Println()
}

// FileTokenStorage implements persistent token storage using a JSON file
type FileTokenStorage struct {
	filepath string
}

func (f *FileTokenStorage) Store(token *threads.TokenInfo) error {
	// In a real application, you would use proper JSON marshaling
	// and secure file permissions (0600)
	content := fmt.Sprintf(`{
	"access_token": "%s",
	"token_type": "%s",
	"expires_at": "%s",
	"user_id": "%s",
	"created_at": "%s"
}`, token.AccessToken, token.TokenType,
		token.ExpiresAt.Format(time.RFC3339),
		token.UserID,
		token.CreatedAt.Format(time.RFC3339))

	return os.WriteFile(f.filepath, []byte(content), 0600)
}

func (f *FileTokenStorage) Load() (*threads.TokenInfo, error) {
	// In a real application, you would use proper JSON unmarshalling
	// This is simplified for demonstration purposes
	if _, err := os.Stat(f.filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("token file does not exist")
	}

	// For this example, we'll return an error to demonstrate the flow
	return nil, fmt.Errorf("token loading not implemented in this example")
}

func (f *FileTokenStorage) Delete() error {
	return os.Remove(f.filepath)
}

func main() {
	fmt.Println("Threads API Authentication Example")
	fmt.Println("==================================")
	fmt.Println()

	// Step 1: Create client configuration
	fmt.Println("Step 1: Setting up client configuration")

	// Try to create client from environment variables first
	client, err := threads.NewClientFromEnv()
	if err != nil {
		fmt.Printf("Could not create client from environment: %v\n", err)
		fmt.Println("Make sure to set THREADS_CLIENT_ID, THREADS_CLIENT_SECRET, and THREADS_REDIRECT_URI")
		fmt.Println("   You can also create a .env file based on .env.example")
		fmt.Println()

		// Fallback to manual configuration for demonstration
		fmt.Println("Using manual configuration for demonstration...")
		config := &threads.Config{
			ClientID:     "your-client-id",
			ClientSecret: "your-client-secret",
			RedirectURI:  "https://yourapp.com/callback",
			Scopes: []string{
				"threads_basic",
				"threads_content_publish",
				"threads_manage_insights",
				"threads_manage_replies",
			},
			HTTPTimeout:  30 * time.Second,
			Logger:       &SimpleLogger{},
			TokenStorage: &FileTokenStorage{filepath: "token.json"},
			Debug:        true,
		}

		client, err = threads.NewClient(config)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
	}

	fmt.Println("Client created successfully")
	fmt.Printf("   Scopes: %v\n", client.GetConfig().Scopes)
	fmt.Printf("   Redirect URI: %s\n", client.GetConfig().RedirectURI)
	fmt.Println()

	// Step 2: Generate authorization URL
	fmt.Println("Step 2: Generate authorization URL")

	scopes := []string{
		"threads_basic",
		"threads_content_publish",
		"threads_manage_insights",
		"threads_manage_replies",
	}

	authURL := client.GetAuthURL(scopes)
	fmt.Println("Authorization URL generated:")
	fmt.Printf("   %s\n", authURL)
	fmt.Println()
	fmt.Println("Instructions:")
	fmt.Println("   1. Open the URL above in your browser")
	fmt.Println("   2. Log in to Threads and authorize your application")
	fmt.Println("   3. Copy the authorization code from the redirect URL")
	fmt.Println("   4. The code will be in the 'code' parameter of the callback URL")
	fmt.Println()

	// Step 3: Get authorization code from user
	fmt.Print("Enter the authorization code (or 'skip' to skip): ")
	reader := bufio.NewReader(os.Stdin)
	authCode, _ := reader.ReadString('\n')
	authCode = strings.TrimSpace(authCode)

	if authCode == "skip" || authCode == "" {
		fmt.Println("Skipping token exchange - this example will show the flow without actual tokens")
		demonstrateTokenManagement()
		return
	}

	// Step 4: Exchange authorization code for access token
	fmt.Println()
	fmt.Println("Step 3: Exchange authorization code for access token")

	err = client.ExchangeCodeForToken(context.Background(), authCode)
	if err != nil {
		fmt.Printf(" Token exchange failed: %v\n", err)

		// Handle different error types
		if threads.IsAuthenticationError(err) {
			fmt.Println(" Authentication error - check your authorization code and app configuration")
		} else if threads.IsValidationError(err) {
			fmt.Println(" Validation error - the authorization code might be invalid or expired")
		} else if threads.IsNetworkError(err) {
			fmt.Println(" Network error - check your internet connection")
		}
		return
	}

	fmt.Println(" Access token obtained successfully!")

	// Show token information
	tokenInfo := client.GetTokenInfo()
	if tokenInfo != nil {
		fmt.Printf("   User ID: %s\n", tokenInfo.UserID)
		fmt.Printf("   Token Type: %s\n", tokenInfo.TokenType)
		fmt.Printf("   Expires At: %s\n", tokenInfo.ExpiresAt.Format(time.RFC3339))
		fmt.Printf("   Time Until Expiry: %s\n", time.Until(tokenInfo.ExpiresAt).String())
	}
	fmt.Println()

	// Step 5: Convert to long-lived token
	fmt.Println(" Step 4: Convert to long-lived token")
	fmt.Println("   Short-lived tokens expire in ~1 hour")
	fmt.Println("   Long-lived tokens expire in ~60 days")

	err = client.GetLongLivedToken(context.Background())
	if err != nil {
		fmt.Printf(" Long-lived token conversion failed: %v\n", err)

		if threads.IsAuthenticationError(err) {
			fmt.Println(" The token might already be long-lived or invalid")
		}
	} else {
		fmt.Println(" Token converted to long-lived successfully!")

		// Show updated token information
		tokenInfo = client.GetTokenInfo()
		if tokenInfo != nil {
			fmt.Printf("   New Expires At: %s\n", tokenInfo.ExpiresAt.Format(time.RFC3339))
			fmt.Printf("   New Time Until Expiry: %s\n", time.Until(tokenInfo.ExpiresAt).String())
			fmt.Printf("   Token Lifetime: %.1f days\n", time.Until(tokenInfo.ExpiresAt).Hours()/24)
		}
	}
	fmt.Println()

	// Step 6: Test the token
	fmt.Println(" Step 5: Test the authenticated token")

	ctx := context.Background()
	user, err := client.GetMe(ctx)
	if err != nil {
		fmt.Printf(" Failed to get user profile: %v\n", err)
	} else {
		fmt.Println(" Token is working! User profile retrieved:")
		fmt.Printf("   ID: %s\n", user.ID)
		fmt.Printf("   Username: %s\n", user.Username)
		if user.Name != "" {
			fmt.Printf("   Name: %s\n", user.Name)
		}
	}
	fmt.Println()

	// Step 7: Demonstrate token management
	fmt.Println(" Step 6: Token management features")
	demonstrateTokenManagementWithClient(client)

	fmt.Println(" Authentication example completed successfully!")
	fmt.Println()
	fmt.Println(" Next steps:")
	fmt.Println("   - Your token is now stored and ready to use")
	fmt.Println("   - Check out other examples for post creation, user management, etc.")
	fmt.Println("   - The client will automatically refresh tokens when needed")
}

func demonstrateTokenManagement() {
	fmt.Println(" Token Management Features (Demo Mode)")
	fmt.Println("=======================================")
	fmt.Println()

	fmt.Println(" Token Information:")
	fmt.Println("   - Access tokens are used to authenticate API requests")
	fmt.Println("   - Short-lived tokens expire in ~1 hour")
	fmt.Println("   - Long-lived tokens expire in ~60 days")
	fmt.Println("   - Tokens can be refreshed before expiration")
	fmt.Println()

	fmt.Println(" Token Refresh:")
	fmt.Println("   - The client automatically refreshes tokens when needed")
	fmt.Println("   - You can manually refresh with client.RefreshToken()")
	fmt.Println("   - Refresh extends the token lifetime")
	fmt.Println()

	fmt.Println(" Token Storage:")
	fmt.Println("   - Implement TokenStorage interface for persistence")
	fmt.Println("   - Default MemoryTokenStorage loses tokens on restart")
	fmt.Println("   - FileTokenStorage example shown above")
	fmt.Println("   - Consider database storage for production apps")
	fmt.Println()

	fmt.Println(" Token Validation:")
	fmt.Println("   - client.IsAuthenticated() - check if token exists")
	fmt.Println("   - client.IsTokenExpired() - check if token is expired")
	fmt.Println("   - client.ValidateToken() - test token with API call")
	fmt.Println("   - client.GetTokenDebugInfo() - detailed token information")
}

func demonstrateTokenManagementWithClient(client *threads.Client) {
	fmt.Println(" Current token status:")
	fmt.Printf("   Authenticated: %t\n", client.IsAuthenticated())
	fmt.Printf("   Expired: %t\n", client.IsTokenExpired())
	fmt.Printf("   Expires soon (1 hour): %t\n", client.IsTokenExpiringSoon(time.Hour))
	fmt.Printf("   Expires soon (1 day): %t\n", client.IsTokenExpiringSoon(24*time.Hour))
	fmt.Println()

	fmt.Println(" Token debug information:")
	debugInfo := client.GetTokenDebugInfo()
	for key, value := range debugInfo {
		fmt.Printf("   %s: %v\n", key, value)
	}
	fmt.Println()

	fmt.Println(" Token validation:")
	err := client.ValidateToken()
	if err != nil {
		fmt.Printf("    Token validation failed: %v\n", err)
	} else {
		fmt.Println("    Token is valid and working")
	}
	fmt.Println()
}
