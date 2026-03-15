package threads

import (
	"context"
	"testing"
	"time"
)

func TestGetPostInsights_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [
			{"name": "views", "period": "lifetime", "values": [{"value": 100}]},
			{"name": "likes", "period": "lifetime", "values": [{"value": 25}]}
		]
	}`))

	resp, err := client.GetPostInsights(context.Background(), ConvertToPostID("post_1"), []string{"views", "likes"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(resp.Data))
	}
	if resp.Data[0].Name != "views" {
		t.Errorf("expected 'views', got %s", resp.Data[0].Name)
	}
}

func TestGetPostInsights_InvalidPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPostInsights(context.Background(), PostID(""), []string{"views"})
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestGetPostInsights_InvalidMetric(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPostInsights(context.Background(), ConvertToPostID("post_1"), []string{"invalid_metric"})
	if err == nil {
		t.Fatal("expected error for invalid metric")
	}
}

func TestGetPostInsights_DefaultMetrics(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[]}`))
	_, err := client.GetPostInsights(context.Background(), ConvertToPostID("post_1"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPostInsightsWithOptions_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"name":"views","period":"day","values":[{"value":50}]}]}`))
	since := time.Now().Add(-24 * time.Hour)
	until := time.Now()
	resp, err := client.GetPostInsightsWithOptions(context.Background(), ConvertToPostID("post_1"), &PostInsightsOptions{
		Metrics: []PostInsightMetric{PostInsightViews},
		Period:  InsightPeriodDay,
		Since:   &since,
		Until:   &until,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 metric, got %d", len(resp.Data))
	}
}

func TestGetPostInsightsWithOptions_InvalidPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPostInsightsWithOptions(context.Background(), PostID(""), nil)
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestGetPostInsightsWithOptions_NilOpts(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[]}`))
	_, err := client.GetPostInsightsWithOptions(context.Background(), ConvertToPostID("post_1"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPostInsightsWithOptions_InvalidMetric(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPostInsightsWithOptions(context.Background(), ConvertToPostID("post_1"), &PostInsightsOptions{
		Metrics: []PostInsightMetric{"invalid"},
	})
	if err == nil {
		t.Fatal("expected error for invalid metric")
	}
}

func TestGetPostInsightsWithOptions_InvalidPeriod(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPostInsightsWithOptions(context.Background(), ConvertToPostID("post_1"), &PostInsightsOptions{
		Period: "weekly",
	})
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestGetPostInsightsWithOptions_InvalidDateRange(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	since := time.Now()
	until := time.Now().Add(-24 * time.Hour) // until before since
	_, err := client.GetPostInsightsWithOptions(context.Background(), ConvertToPostID("post_1"), &PostInsightsOptions{
		Since: &since,
		Until: &until,
	})
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
}

func TestGetAccountInsights_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"name": "followers_count", "period": "day", "values": [{"value": 500}]}]
	}`))

	resp, err := client.GetAccountInsights(context.Background(), ConvertToUserID("12345"), []string{"followers_count"}, "day")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 metric, got %d", len(resp.Data))
	}
}

func TestGetAccountInsights_InvalidUserID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetAccountInsights(context.Background(), UserID(""), nil, "")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestGetAccountInsights_DefaultMetrics(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[]}`))
	_, err := client.GetAccountInsights(context.Background(), ConvertToUserID("12345"), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAccountInsights_InvalidMetric(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetAccountInsights(context.Background(), ConvertToUserID("12345"), []string{"bad_metric"}, "day")
	if err == nil {
		t.Fatal("expected error for invalid metric")
	}
}

func TestGetAccountInsights_InvalidPeriod(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetAccountInsights(context.Background(), ConvertToUserID("12345"), []string{"views"}, "weekly")
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestGetAccountInsightsWithOptions_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"name":"views","period":"day","values":[{"value":100}]}]}`))
	since := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	resp, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
		Period:  InsightPeriodDay,
		Since:   &since,
		Until:   &until,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 metric, got %d", len(resp.Data))
	}
}

func TestGetAccountInsightsWithOptions_InvalidUserID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetAccountInsightsWithOptions(context.Background(), UserID(""), nil)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestGetAccountInsightsWithOptions_NilOpts(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[]}`))
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAccountInsightsWithOptions_InvalidMetric(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{"invalid"},
	})
	if err == nil {
		t.Fatal("expected error for invalid metric")
	}
}

func TestGetAccountInsightsWithOptions_InvalidPeriod(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Period: "weekly",
	})
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
}

func TestGetAccountInsightsWithOptions_FollowerDemographicsWithSince(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	since := time.Now()
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightFollowerDemographics},
		Since:   &since,
	})
	if err == nil {
		t.Fatal("expected error for follower_demographics with since")
	}
}

func TestGetAccountInsightsWithOptions_FollowerDemographicsWithBreakdown(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[]}`))
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics:   []AccountInsightMetric{AccountInsightFollowerDemographics},
		Breakdown: "country",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAccountInsightsWithOptions_FollowerDemographicsInvalidBreakdown(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics:   []AccountInsightMetric{AccountInsightFollowerDemographics},
		Breakdown: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid breakdown")
	}
}

func TestGetAccountInsightsWithOptions_FollowersCountWithSince(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	since := time.Now()
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightFollowersCount},
		Since:   &since,
	})
	if err == nil {
		t.Fatal("expected error for followers_count with since")
	}
}

func TestGetAccountInsightsWithOptions_MinSinceTimestamp(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	// Timestamp before MinInsightTimestamp
	early := time.Unix(100, 0)
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
		Since:   &early,
	})
	if err == nil {
		t.Fatal("expected error for since before MinInsightTimestamp")
	}
}

func TestGetAccountInsightsWithOptions_MinUntilTimestamp(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	early := time.Unix(100, 0)
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
		Until:   &early,
	})
	if err == nil {
		t.Fatal("expected error for until before MinInsightTimestamp")
	}
}

func TestGetAccountInsightsWithOptions_InvalidDateRange(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	since := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := client.GetAccountInsightsWithOptions(context.Background(), ConvertToUserID("12345"), &AccountInsightsOptions{
		Metrics: []AccountInsightMetric{AccountInsightViews},
		Since:   &since,
		Until:   &until,
	})
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
}

func TestValidateFollowerDemographicsBreakdown_ValidValues(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	for _, b := range []string{"country", "city", "age", "gender"} {
		if err := client.validateFollowerDemographicsBreakdown(b); err != nil {
			t.Errorf("expected no error for breakdown %s, got %v", b, err)
		}
	}
}

func TestValidateFollowerDemographicsBreakdown_Invalid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.validateFollowerDemographicsBreakdown("invalid")
	if err == nil {
		t.Fatal("expected error for invalid breakdown")
	}
}

func TestGetAvailablePostInsightMetrics(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	metrics := client.GetAvailablePostInsightMetrics()
	if len(metrics) != 6 {
		t.Errorf("expected 6 metrics, got %d", len(metrics))
	}
}

func TestGetAvailableAccountInsightMetrics(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	metrics := client.GetAvailableAccountInsightMetrics()
	if len(metrics) != 8 {
		t.Errorf("expected 8 metrics, got %d", len(metrics))
	}
}

func TestGetAvailableInsightPeriods(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	periods := client.GetAvailableInsightPeriods()
	if len(periods) != 2 {
		t.Errorf("expected 2 periods, got %d", len(periods))
	}
}

func TestGetAvailableFollowerDemographicsBreakdowns(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	breakdowns := client.GetAvailableFollowerDemographicsBreakdowns()
	if len(breakdowns) != 4 {
		t.Errorf("expected 4 breakdowns, got %d", len(breakdowns))
	}
}
