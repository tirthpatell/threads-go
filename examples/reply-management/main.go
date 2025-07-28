// Package main demonstrates reply management functionality using the Threads API.
//
// This example shows how to:
// 1. Create replies to posts
// 2. Retrieve replies and conversations
// 3. Hide and unhide replies for moderation
// 4. Get user's reply history
// 5. Handle threaded conversations
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tirthpatell/threads-go"
)

func main() {
	fmt.Println(" Threads API Reply Management Examples")
	fmt.Println("========================================")
	fmt.Println()

	// Create client from environment variables
	client, err := threads.NewClientFromEnv()
	if err != nil {
		log.Fatalf(" Failed to create client: %v\nMake sure to set THREADS_CLIENT_ID, THREADS_CLIENT_SECRET, and THREADS_REDIRECT_URI", err)
	}

	// Check if we're authenticated
	if !client.IsAuthenticated() {
		fmt.Println(" Client is not authenticated")
		fmt.Println(" Run the authentication example first to get a token")
		return
	}

	fmt.Println(" Client authenticated and ready")
	fmt.Println()

	ctx := context.Background()
	// Get current user info
	me, err := client.GetMe(ctx)
	if err != nil {
		log.Fatalf(" Failed to get user info: %v", err)
	}

	fmt.Printf(" Authenticated as: %s (@%s)\n", me.Name, me.Username)
	fmt.Println()

	// Example 1: Create a post to reply to
	fmt.Println("Example 1: Create a Post for Reply Testing")
	fmt.Println("==========================================")
	originalPost := createTestPost(client)
	if originalPost == nil {
		return
	}
	fmt.Println()

	// Example 2: Create replies
	fmt.Println("Example 2: Create Replies")
	fmt.Println("========================")
	replies := createReplies(client, originalPost.ID)
	fmt.Println()

	// Example 3: Retrieve replies
	fmt.Println("Example 3: Retrieve Replies")
	fmt.Println("===========================")
	retrieveReplies(client, originalPost.ID)
	fmt.Println()

	// Example 4: Get conversation
	fmt.Println("Example 4: Get Full Conversation")
	fmt.Println("================================")
	getConversation(client, originalPost.ID)
	fmt.Println()

	// Example 5: Reply moderation
	fmt.Println("Example 5: Reply Moderation")
	fmt.Println("===========================")
	if len(replies) > 0 {
		moderateReplies(client, replies[0])
	} else {
		fmt.Println("Skipping moderation - no replies created")
	}
	fmt.Println()

	// Example 6: User reply history
	fmt.Println("Example 6: User Reply History")
	fmt.Println("=============================")
	getUserReplyHistory(client, me.ID)
	fmt.Println()

	// Example 7: Advanced reply options
	fmt.Println("Example 7: Advanced Reply Features")
	fmt.Println("==================================")
	demonstrateAdvancedReplyFeatures(client, originalPost.ID)
	fmt.Println()

	fmt.Println("Reply management examples completed!")
}

func createTestPost(client *threads.Client) *threads.Post {
	ctx := context.Background()
	content := &threads.TextPostContent{
		Text:         " This is a test post for reply management examples!\n\nFeel free to reply and test the conversation features. #ThreadsAPI #Testing",
		ReplyControl: threads.ReplyControlEveryone,
	}

	post, err := client.CreateTextPost(ctx, content)
	if err != nil {
		fmt.Printf(" Failed to create test post: %v\n", err)
		return nil
	}

	fmt.Println(" Test post created successfully!")
	fmt.Printf("   Post ID: %s\n", post.ID)
	fmt.Printf("   Text: %s\n", post.Text)
	fmt.Printf("   Permalink: %s\n", post.Permalink)

	return post
}

func createReplies(client *threads.Client, postID string) []*threads.Post {
	var replies []*threads.Post

	// Reply 1: Simple text reply
	fmt.Println(" Creating simple text reply...")
	reply1Content := &threads.PostContent{
		Text:    "Great post!  Thanks for sharing this with the community.",
		ReplyTo: postID,
	}

	ctx := context.Background()
	reply1, err := client.CreateReply(ctx, reply1Content)
	if err != nil {
		fmt.Printf(" Failed to create reply 1: %v\n", err)
	} else {
		fmt.Println(" Reply 1 created successfully!")
		fmt.Printf("   Reply ID: %s\n", reply1.ID)
		fmt.Printf("   Text: %s\n", reply1.Text)
		replies = append(replies, reply1)
	}

	// Small delay between replies
	time.Sleep(2 * time.Second)

	// Reply 2: Using ReplyToPost method
	fmt.Println("\n Creating reply using ReplyToPost method...")
	reply2Content := &threads.PostContent{
		Text: " This is another reply using the ReplyToPost method. Very convenient!",
	}

	postIDTyped := threads.ConvertToPostID(postID)
	reply2, err := client.ReplyToPost(ctx, postIDTyped, reply2Content)
	if err != nil {
		fmt.Printf(" Failed to create reply 2: %v\n", err)
	} else {
		fmt.Println(" Reply 2 created successfully!")
		fmt.Printf("   Reply ID: %s\n", reply2.ID)
		fmt.Printf("   Text: %s\n", reply2.Text)
		replies = append(replies, reply2)
	}

	// Small delay between replies
	time.Sleep(2 * time.Second)

	// Reply 3: Longer reply with more content
	fmt.Println("\n Creating detailed reply...")
	reply3Content := &threads.PostContent{
		Text: " This is a more detailed reply that demonstrates longer content.\n\nI'm testing the reply functionality of the Threads API, and it's working great! The threading system makes conversations easy to follow.\n\n#ThreadsAPI #Development",
	}

	reply3, err := client.ReplyToPost(ctx, postIDTyped, reply3Content)
	if err != nil {
		fmt.Printf(" Failed to create reply 3: %v\n", err)
	} else {
		fmt.Println(" Reply 3 created successfully!")
		fmt.Printf("   Reply ID: %s\n", reply3.ID)
		fmt.Printf("   Text: %s\n", reply3.Text[:100])
		if len(reply3.Text) > 100 {
			fmt.Print("...")
		}
		fmt.Println()
		replies = append(replies, reply3)
	}

	fmt.Printf("\n Created %d replies total\n", len(replies))
	return replies
}

func retrieveReplies(client *threads.Client, postID string) {
	fmt.Println(" Retrieving replies with default options...")

	repliesResp, err := client.GetReplies(context.Background(), threads.ConvertToPostID(postID), &threads.RepliesOptions{
		Limit: 25,
	})

	if err != nil {
		fmt.Printf(" Failed to retrieve replies: %v\n", err)
		return
	}

	fmt.Printf(" Retrieved %d replies\n", len(repliesResp.Data))

	for i, reply := range repliesResp.Data {
		fmt.Printf("\n   Reply %d:\n", i+1)
		fmt.Printf("     ID: %s\n", reply.ID)
		fmt.Printf("     Username: %s\n", reply.Username)
		fmt.Printf("     Timestamp: %s\n", reply.Timestamp.Format(time.RFC3339))
		fmt.Printf("     Is Reply: %t\n", reply.IsReply)

		// Truncate long text for display
		text := reply.Text
		if len(text) > 150 {
			text = text[:150] + "..."
		}
		fmt.Printf("     Text: %s\n", text)

		if reply.HasReplies {
			fmt.Printf("     Has Replies: %t\n", reply.HasReplies)
		}
	}

	// Show pagination info
	if repliesResp.Paging.Cursors != nil {
		fmt.Printf("\n Pagination:\n")
		if repliesResp.Paging.Cursors.Before != "" {
			fmt.Printf("   Before: %s\n", repliesResp.Paging.Cursors.Before)
		}
		if repliesResp.Paging.Cursors.After != "" {
			fmt.Printf("   After: %s\n", repliesResp.Paging.Cursors.After)
		}
	}

	// Test reverse chronological order
	fmt.Println("\n Testing reverse chronological order...")
	reverse := false
	repliesResp, err = client.GetReplies(context.Background(), threads.ConvertToPostID(postID), &threads.RepliesOptions{
		Limit:   10,
		Reverse: &reverse,
	})

	if err != nil {
		fmt.Printf(" Failed to retrieve replies in chronological order: %v\n", err)
	} else {
		fmt.Printf(" Retrieved %d replies in chronological order\n", len(repliesResp.Data))
	}
}

func getConversation(client *threads.Client, postID string) {
	fmt.Println(" Retrieving full conversation thread...")

	conversationResp, err := client.GetConversation(context.Background(), threads.ConvertToPostID(postID), &threads.RepliesOptions{
		Limit: 50,
	})

	if err != nil {
		fmt.Printf(" Failed to retrieve conversation: %v\n", err)
		return
	}

	fmt.Printf(" Retrieved conversation with %d posts\n", len(conversationResp.Data))

	fmt.Println("\n Conversation thread:")
	for i, post := range conversationResp.Data {
		indent := ""
		if post.IsReply {
			indent = "  ↳ "
		}

		fmt.Printf("\n   %s%d. %s (@%s)\n", indent, i+1, post.ID, post.Username)
		fmt.Printf("   %s   Time: %s\n", indent, post.Timestamp.Format("15:04:05"))

		// Truncate long text for display
		text := post.Text
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		fmt.Printf("   %s   Text: %s\n", indent, text)

		if post.IsReply && post.RepliedTo != nil {
			fmt.Printf("   %s   Replied to: %s\n", indent, post.RepliedTo.ID)
		}
	}
}

func moderateReplies(client *threads.Client, reply *threads.Post) {
	fmt.Printf("  Testing reply moderation with reply: %s\n", reply.ID)

	// Hide the reply
	fmt.Println("\n🙈 Hiding reply...")
	err := client.HideReply(context.Background(), threads.ConvertToPostID(reply.ID))
	if err != nil {
		fmt.Printf(" Failed to hide reply: %v\n", err)

		if threads.IsAuthenticationError(err) {
			fmt.Println(" You can only hide replies to your own posts")
		}
	} else {
		fmt.Println(" Reply hidden successfully!")
		fmt.Printf("   Hidden reply ID: %s\n", reply.ID)
	}

	// Wait a moment
	time.Sleep(2 * time.Second)

	// Unhide the reply
	fmt.Println("\n  Unhiding reply...")
	err = client.UnhideReply(context.Background(), threads.ConvertToPostID(reply.ID))
	if err != nil {
		fmt.Printf(" Failed to unhide reply: %v\n", err)

		if threads.IsAuthenticationError(err) {
			fmt.Println(" You can only unhide replies to your own posts")
		}
	} else {
		fmt.Println(" Reply unhidden successfully!")
		fmt.Printf("   Unhidden reply ID: %s\n", reply.ID)
	}
}

func getUserReplyHistory(client *threads.Client, userID string) {
	fmt.Printf(" Retrieving reply history for user: %s\n", userID)

	repliesResp, err := client.GetUserReplies(context.Background(), threads.ConvertToUserID(userID), &threads.PostsOptions{
		Limit: 20,
	})

	if err != nil {
		fmt.Printf(" Failed to retrieve user replies: %v\n", err)
		return
	}

	fmt.Printf(" Retrieved %d replies from user's history\n", len(repliesResp.Data))

	if len(repliesResp.Data) == 0 {
		fmt.Println("   No replies found in user's history")
		return
	}

	fmt.Println("\n Recent replies:")
	for i, reply := range repliesResp.Data {
		if i >= 5 { // Show only first 5 for brevity
			fmt.Printf("   ... and %d more replies\n", len(repliesResp.Data)-5)
			break
		}

		fmt.Printf("\n   Reply %d:\n", i+1)
		fmt.Printf("     ID: %s\n", reply.ID)
		fmt.Printf("     Timestamp: %s\n", reply.Timestamp.Format(time.RFC3339))

		// Truncate long text for display
		text := reply.Text
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		fmt.Printf("     Text: %s\n", text)

		if reply.RootPost != nil {
			fmt.Printf("     Root Post: %s\n", reply.RootPost.ID)
		}
		if reply.RepliedTo != nil {
			fmt.Printf("     Replied To: %s\n", reply.RepliedTo.ID)
		}
	}
}

func demonstrateAdvancedReplyFeatures(client *threads.Client, postID string) {
	fmt.Println(" Demonstrating advanced reply features...")

	// Test pagination with replies
	fmt.Println("\n Testing reply pagination...")

	// Get first page
	firstPage, err := client.GetReplies(context.Background(), threads.ConvertToPostID(postID), &threads.RepliesOptions{
		Limit: 2, // Small limit to test pagination
	})

	if err != nil {
		fmt.Printf(" Failed to get first page: %v\n", err)
		return
	}

	fmt.Printf(" First page: %d replies\n", len(firstPage.Data))

	// Get next page if available
	if firstPage.Paging.Cursors != nil && firstPage.Paging.Cursors.After != "" {
		fmt.Println(" Getting next page...")

		nextPage, err := client.GetReplies(context.Background(), threads.ConvertToPostID(postID), &threads.RepliesOptions{
			Limit: 2,
			After: firstPage.Paging.Cursors.After,
		})

		if err != nil {
			fmt.Printf(" Failed to get next page: %v\n", err)
		} else {
			fmt.Printf(" Next page: %d replies\n", len(nextPage.Data))
		}
	} else {
		fmt.Println(" No next page available")
	}

	// Test different sorting orders
	fmt.Println("\n Testing different reply sorting...")

	// Chronological order (oldest first)
	reverse := false
	chronological, err := client.GetReplies(context.Background(), threads.ConvertToPostID(postID), &threads.RepliesOptions{
		Limit:   5,
		Reverse: &reverse,
	})

	if err != nil {
		fmt.Printf(" Failed to get chronological replies: %v\n", err)
	} else {
		fmt.Printf(" Chronological order: %d replies\n", len(chronological.Data))
		if len(chronological.Data) > 0 {
			fmt.Printf("   First reply timestamp: %s\n", chronological.Data[0].Timestamp.Format(time.RFC3339))
		}
	}

	// Reverse chronological order (newest first)
	reverse = true
	reverseChronological, err := client.GetReplies(context.Background(), threads.ConvertToPostID(postID), &threads.RepliesOptions{
		Limit:   5,
		Reverse: &reverse,
	})

	if err != nil {
		fmt.Printf(" Failed to get reverse chronological replies: %v\n", err)
	} else {
		fmt.Printf(" Reverse chronological order: %d replies\n", len(reverseChronological.Data))
		if len(reverseChronological.Data) > 0 {
			fmt.Printf("   First reply timestamp: %s\n", reverseChronological.Data[0].Timestamp.Format(time.RFC3339))
		}
	}

	// Demonstrate error handling
	fmt.Println("\n Testing error handling...")

	// Try to get replies for non-existent post
	_, err = client.GetReplies(context.Background(), threads.ConvertToPostID("non_existent_post_id"), &threads.RepliesOptions{
		Limit: 10,
	})

	if err != nil {
		fmt.Printf(" Correctly caught error for non-existent post: %v\n", err)

		if threads.IsValidationError(err) {
			fmt.Println("    Validation error - post not found")
		}
	} else {
		fmt.Println(" Unexpected success for non-existent post")
	}
}
