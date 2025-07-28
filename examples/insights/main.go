// Package main demonstrates insights and analytics functionality using the Threads API.
//
// This example shows how to:
// 1. Get post insights and performance metrics
// 2. Get account insights and analytics
// 3. Work with different time periods and metrics
// 4. Handle follower demographics and breakdowns
// 5. Monitor publishing limits and quotas
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tirthpatell/threads-go"
)

func main() {
	fmt.Println(" Threads API Insights & Analytics Examples")
	fmt.Println("============================================")
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

	// Example 1: Publishing limits and quotas
	fmt.Println("Example 1: Publishing Limits & Quotas")
	fmt.Println("=====================================")
	checkPublishingLimits(client)
	fmt.Println()

	// Example 2: Basic post insights
	fmt.Println("Example 2: Basic Post Insights")
	fmt.Println("==============================")
	demonstratePostInsights(client, me.ID)
	fmt.Println()

	// Example 3: Advanced post insights
	fmt.Println("Example 3: Advanced Post Insights")
	fmt.Println("=================================")
	demonstrateAdvancedPostInsights(client, me.ID)
	fmt.Println()

	// Example 4: Account insights
	fmt.Println("Example 4: Account Insights")
	fmt.Println("===========================")
	demonstrateAccountInsights(client, me.ID)
	fmt.Println()

	// Example 5: Advanced account insights
	fmt.Println("Example 5: Advanced Account Insights")
	fmt.Println("====================================")
	demonstrateAdvancedAccountInsights(client, me.ID)
	fmt.Println()

	// Example 6: Follower demographics
	fmt.Println("Example 6: Follower Demographics")
	fmt.Println("================================")
	demonstrateFollowerDemographics(client, me.ID)
	fmt.Println()

	// Example 7: Available metrics and periods
	fmt.Println("Example 7: Available Metrics & Periods")
	fmt.Println("======================================")
	showAvailableOptions(client)
	fmt.Println()

	// Example 8: Error handling
	fmt.Println("Example 8: Error Handling")
	fmt.Println("========================")
	demonstrateInsightsErrorHandling(client)
	fmt.Println()

	fmt.Println("Insights and analytics examples completed!")
}

func checkPublishingLimits(client *threads.Client) {
	ctx := context.Background()
	fmt.Println(" Checking current API quota usage...")

	limits, err := client.GetPublishingLimits(ctx)
	if err != nil {
		fmt.Printf(" Failed to get publishing limits: %v\n", err)
		return
	}

	fmt.Println(" Publishing limits retrieved successfully!")
	fmt.Println()

	// Posts quota
	fmt.Printf(" Posts Quota:\n")
	fmt.Printf("   Used: %d / %d\n", limits.QuotaUsage, limits.Config.QuotaTotal)
	fmt.Printf("   Remaining: %d\n", limits.Config.QuotaTotal-limits.QuotaUsage)
	fmt.Printf("   Usage: %.1f%%\n", float64(limits.QuotaUsage)/float64(limits.Config.QuotaTotal)*100)
	fmt.Printf("   Duration: %d seconds\n", limits.Config.QuotaDuration)
	fmt.Println()

	// Replies quota
	fmt.Printf(" Replies Quota:\n")
	fmt.Printf("   Used: %d / %d\n", limits.ReplyQuotaUsage, limits.ReplyConfig.QuotaTotal)
	fmt.Printf("   Remaining: %d\n", limits.ReplyConfig.QuotaTotal-limits.ReplyQuotaUsage)
	fmt.Printf("   Usage: %.1f%%\n", float64(limits.ReplyQuotaUsage)/float64(limits.ReplyConfig.QuotaTotal)*100)
	fmt.Printf("   Duration: %d seconds\n", limits.ReplyConfig.QuotaDuration)
	fmt.Println()

	// Delete quota
	fmt.Printf("Delete Quota:\n")
	fmt.Printf("   Used: %d / %d\n", limits.DeleteQuotaUsage, limits.DeleteConfig.QuotaTotal)
	fmt.Printf("   Remaining: %d\n", limits.DeleteConfig.QuotaTotal-limits.DeleteQuotaUsage)
	fmt.Printf("   Usage: %.1f%%\n", float64(limits.DeleteQuotaUsage)/float64(limits.DeleteConfig.QuotaTotal)*100)
	fmt.Printf("   Duration: %d seconds\n", limits.DeleteConfig.QuotaDuration)
	fmt.Println()

	// Location search quota
	fmt.Printf(" Location Search Quota:\n")
	fmt.Printf("   Used: %d / %d\n", limits.LocationSearchQuotaUsage, limits.LocationSearchConfig.QuotaTotal)
	fmt.Printf("   Remaining: %d\n", limits.LocationSearchConfig.QuotaTotal-limits.LocationSearchQuotaUsage)
	fmt.Printf("   Usage: %.1f%%\n", float64(limits.LocationSearchQuotaUsage)/float64(limits.LocationSearchConfig.QuotaTotal)*100)
	fmt.Printf("   Duration: %d seconds\n", limits.LocationSearchConfig.QuotaDuration)
}

func demonstratePostInsights(client *threads.Client, userID string) {
	fmt.Println(" Getting basic post insights...")

	ctx := context.Background()
	// First, get some posts to analyze
	userIDTyped := threads.ConvertToUserID(userID)
	posts, err := client.GetUserPosts(ctx, userIDTyped, &threads.PaginationOptions{
		Limit: 5,
	})

	if err != nil {
		fmt.Printf(" Failed to get user posts: %v\n", err)
		return
	}

	if len(posts.Data) == 0 {
		fmt.Println("  No posts found for insights analysis")
		fmt.Println(" Create some posts first using the post-creation example")
		return
	}

	fmt.Printf(" Found %d posts to analyze\n", len(posts.Data))
	fmt.Println()

	// Analyze first few posts
	for i, post := range posts.Data {
		if i >= 3 { // Limit to first 3 posts
			break
		}

		fmt.Printf(" Post %d Insights (ID: %s)\n", i+1, post.ID)
		fmt.Printf("   Text: %s\n", truncateText(post.Text, 80))
		fmt.Printf("   Created: %s\n", post.Timestamp.Format(time.RFC3339))
		fmt.Println()

		// Get basic insights
		insights, err := client.GetPostInsights(context.Background(), threads.ConvertToPostID(post.ID), []string{
			"views", "likes", "replies", "reposts",
		})

		if err != nil {
			fmt.Printf("    Failed to get insights: %v\n", err)
			continue
		}

		fmt.Printf("    Metrics:\n")
		for _, insight := range insights.Data {
			if insight.TotalValue != nil {
				fmt.Printf("     %s: %d\n", insight.Name, insight.TotalValue.Value)
			} else if len(insight.Values) > 0 {
				fmt.Printf("     %s: %d\n", insight.Name, insight.Values[0].Value)
			}
		}
		fmt.Println()
	}
}

func demonstrateAdvancedPostInsights(client *threads.Client, userID string) {
	fmt.Println(" Getting advanced post insights with options...")

	ctx := context.Background()
	// Get recent posts
	userIDTyped := threads.ConvertToUserID(userID)
	posts, err := client.GetUserPosts(ctx, userIDTyped, &threads.PaginationOptions{
		Limit: 3,
	})

	if err != nil {
		fmt.Printf(" Failed to get user posts: %v\n", err)
		return
	}

	if len(posts.Data) == 0 {
		fmt.Println("  No posts found for advanced insights analysis")
		return
	}

	// Analyze first post with advanced options
	post := posts.Data[0]
	fmt.Printf(" Advanced insights for post: %s\n", post.ID)
	fmt.Printf("   Text: %s\n", truncateText(post.Text, 100))
	fmt.Println()

	// Get insights with advanced options
	since := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	until := time.Now()

	insights, err := client.GetPostInsightsWithOptions(context.Background(), threads.ConvertToPostID(post.ID), &threads.PostInsightsOptions{
		Metrics: []threads.PostInsightMetric{
			threads.PostInsightViews,
			threads.PostInsightLikes,
			threads.PostInsightReplies,
			threads.PostInsightReposts,
			threads.PostInsightQuotes,
			threads.PostInsightShares,
		},
		Period: threads.InsightPeriodLifetime,
		Since:  &since,
		Until:  &until,
	})

	if err != nil {
		fmt.Printf(" Failed to get advanced insights: %v\n", err)
		return
	}

	fmt.Printf(" Advanced insights retrieved:\n")
	for _, insight := range insights.Data {
		fmt.Printf("    %s (%s):\n", insight.Title, insight.Name)
		fmt.Printf("      Description: %s\n", insight.Description)
		fmt.Printf("      Period: %s\n", insight.Period)

		if insight.TotalValue != nil {
			fmt.Printf("      Total Value: %d\n", insight.TotalValue.Value)
		}

		if len(insight.Values) > 0 {
			fmt.Printf("      Values:\n")
			for j, value := range insight.Values {
				if j >= 3 { // Show only first 3 values
					fmt.Printf("        ... and %d more values\n", len(insight.Values)-3)
					break
				}
				fmt.Printf("        Value: %d", value.Value)
				if value.EndTime != "" {
					fmt.Printf(" (End Time: %s)", value.EndTime)
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}
}

func demonstrateAccountInsights(client *threads.Client, userID string) {
	fmt.Println(" Getting basic account insights...")

	insights, err := client.GetAccountInsights(context.Background(), threads.ConvertToUserID(userID), []string{
		"views", "likes", "replies", "reposts",
	}, "lifetime")

	if err != nil {
		fmt.Printf(" Failed to get account insights: %v\n", err)
		return
	}

	fmt.Printf(" Account insights retrieved:\n")
	for _, insight := range insights.Data {
		fmt.Printf("    %s:\n", insight.Name)

		if insight.TotalValue != nil {
			fmt.Printf("      Total: %d\n", insight.TotalValue.Value)
		} else if len(insight.Values) > 0 {
			fmt.Printf("      Value: %d\n", insight.Values[0].Value)
		}

		fmt.Printf("      Period: %s\n", insight.Period)
		if insight.Description != "" {
			fmt.Printf("      Description: %s\n", insight.Description)
		}
		fmt.Println()
	}
}

func demonstrateAdvancedAccountInsights(client *threads.Client, userID string) {
	fmt.Println(" Getting advanced account insights...")

	// Test with time range (30 days ago to now)
	since := time.Now().Add(-30 * 24 * time.Hour)
	until := time.Now()

	insights, err := client.GetAccountInsightsWithOptions(context.Background(), threads.ConvertToUserID(userID), &threads.AccountInsightsOptions{
		Metrics: []threads.AccountInsightMetric{
			threads.AccountInsightViews,
			threads.AccountInsightLikes,
			threads.AccountInsightReplies,
			threads.AccountInsightReposts,
			threads.AccountInsightQuotes,
			threads.AccountInsightClicks,
		},
		Period: threads.InsightPeriodLifetime,
		Since:  &since,
		Until:  &until,
	})

	if err != nil {
		fmt.Printf(" Failed to get advanced account insights: %v\n", err)
		return
	}

	fmt.Printf(" Advanced account insights (last 30 days):\n")
	for _, insight := range insights.Data {
		fmt.Printf("    %s:\n", insight.Name)

		if insight.TotalValue != nil {
			fmt.Printf("      Total: %d\n", insight.TotalValue.Value)
		}

		if len(insight.Values) > 0 {
			fmt.Printf("      Daily breakdown:\n")
			for j, value := range insight.Values {
				if j >= 5 { // Show only first 5 days
					fmt.Printf("        ... and %d more days\n", len(insight.Values)-5)
					break
				}
				fmt.Printf("        %s: %d\n", value.EndTime, value.Value)
			}
		}
		fmt.Println()
	}
}

func demonstrateFollowerDemographics(client *threads.Client, userID string) {
	fmt.Println(" Getting follower demographics...")

	// Test different demographic breakdowns
	breakdowns := []string{"country", "city", "age", "gender"}

	for _, breakdown := range breakdowns {
		fmt.Printf("\n Follower demographics by %s:\n", breakdown)

		insights, err := client.GetAccountInsightsWithOptions(context.Background(), threads.ConvertToUserID(userID), &threads.AccountInsightsOptions{
			Metrics: []threads.AccountInsightMetric{
				threads.AccountInsightFollowerDemographics,
			},
			Period:    threads.InsightPeriodLifetime,
			Breakdown: breakdown,
		})

		if err != nil {
			fmt.Printf("    Failed to get %s demographics: %v\n", breakdown, err)
			continue
		}

		if len(insights.Data) == 0 {
			fmt.Printf("     No demographic data available for %s\n", breakdown)
			continue
		}

		for _, insight := range insights.Data {
			fmt.Printf("    %s breakdown:\n", insight.Name)

			if len(insight.Values) > 0 {
				for j, value := range insight.Values {
					if j >= 10 { // Show only top 10
						fmt.Printf("      ... and %d more entries\n", len(insight.Values)-10)
						break
					}
					fmt.Printf("      %s: %d\n", value.EndTime, value.Value)
				}
			}
		}
	}

	// Get followers count separately (doesn't support time filtering)
	fmt.Printf("\n👥 Current followers count:\n")
	followersInsights, err := client.GetAccountInsightsWithOptions(context.Background(), threads.ConvertToUserID(userID), &threads.AccountInsightsOptions{
		Metrics: []threads.AccountInsightMetric{
			threads.AccountInsightFollowersCount,
		},
		Period: threads.InsightPeriodLifetime,
	})

	if err != nil {
		fmt.Printf("    Failed to get followers count: %v\n", err)
	} else {
		for _, insight := range followersInsights.Data {
			if insight.TotalValue != nil {
				fmt.Printf("   Total Followers: %d\n", insight.TotalValue.Value)
			} else if len(insight.Values) > 0 {
				fmt.Printf("   Current Followers: %d\n", insight.Values[0].Value)
			}
		}
	}
}

func showAvailableOptions(client *threads.Client) {
	fmt.Println(" Available insights options:")
	fmt.Println()

	// Show available post metrics
	fmt.Printf(" Post Insight Metrics:\n")
	postMetrics := client.GetAvailablePostInsightMetrics()
	for _, metric := range postMetrics {
		fmt.Printf("   - %s\n", metric)
	}
	fmt.Println()

	// Show available account metrics
	fmt.Printf(" Account Insight Metrics:\n")
	accountMetrics := client.GetAvailableAccountInsightMetrics()
	for _, metric := range accountMetrics {
		fmt.Printf("   - %s\n", metric)
	}
	fmt.Println()

	// Show available periods
	fmt.Printf(" Available Periods:\n")
	periods := client.GetAvailableInsightPeriods()
	for _, period := range periods {
		fmt.Printf("   - %s\n", period)
	}
	fmt.Println()

	// Show available demographic breakdowns
	fmt.Printf(" Available Demographic Breakdowns:\n")
	breakdowns := client.GetAvailableFollowerDemographicsBreakdowns()
	for _, breakdown := range breakdowns {
		fmt.Printf("   - %s\n", breakdown)
	}
}

func demonstrateInsightsErrorHandling(client *threads.Client) {
	fmt.Println(" Testing insights error handling...")

	// Test 1: Invalid post ID
	fmt.Println("\n Test 1: Invalid post ID")
	_, err := client.GetPostInsights(context.Background(), threads.ConvertToPostID("invalid_post_id"), []string{"views"})
	if err != nil {
		fmt.Printf(" Correctly caught error: %v\n", err)
		if threads.IsValidationError(err) {
			fmt.Println("    Validation error - post not found")
		}
	}

	// Test 2: Invalid metric
	fmt.Println("\n Test 2: Invalid metric")
	ctx := context.Background()
	me, _ := client.GetMe(ctx)
	_, err = client.GetAccountInsights(context.Background(), threads.ConvertToUserID(me.ID), []string{"invalid_metric"}, "lifetime")
	if err != nil {
		fmt.Printf(" Correctly caught error: %v\n", err)
		if threads.IsValidationError(err) {
			fmt.Println("    Validation error - invalid metric")
		}
	}

	// Test 3: Invalid period
	fmt.Println("\n Test 3: Invalid period")
	_, err = client.GetAccountInsights(context.Background(), threads.ConvertToUserID(me.ID), []string{"views"}, "invalid_period")
	if err != nil {
		fmt.Printf(" Correctly caught error: %v\n", err)
		if threads.IsValidationError(err) {
			fmt.Println("    Validation error - invalid period")
		}
	}

	// Test 4: Invalid date range
	fmt.Println("\n Test 4: Invalid date range")
	since := time.Now()
	until := time.Now().Add(-24 * time.Hour) // Until is before since

	_, err = client.GetAccountInsightsWithOptions(context.Background(), threads.ConvertToUserID(me.ID), &threads.AccountInsightsOptions{
		Metrics: []threads.AccountInsightMetric{threads.AccountInsightViews},
		Period:  threads.InsightPeriodLifetime,
		Since:   &since,
		Until:   &until,
	})

	if err != nil {
		fmt.Printf(" Correctly caught error: %v\n", err)
		if threads.IsValidationError(err) {
			fmt.Println("    Validation error - invalid date range")
		}
	}
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
