package threads

import (
	"context"
	"testing"
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
