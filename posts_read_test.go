package threads

import (
	"context"
	"net/http"
	"testing"
)

func TestGetPost_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"id": "123456",
		"text": "Hello world",
		"media_type": "TEXT",
		"permalink": "https://threads.net/@user/post/123456",
		"username": "testuser",
		"timestamp": "2026-01-15T10:30:00+0000"
	}`))

	post, err := client.GetPost(context.Background(), ConvertToPostID("123456"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "123456" {
		t.Errorf("expected ID 123456, got %s", post.ID)
	}
	if post.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got %s", post.Text)
	}
	if post.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", post.Username)
	}
}

func TestGetPost_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.GetPost(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"Object does not exist","type":"OAuthException","code":100}}`))

	_, err := client.GetPost(context.Background(), ConvertToPostID("nonexistent"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetPost_ServerError(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"Internal error","type":"OAuthException","code":2}}`))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.GetPost(context.Background(), ConvertToPostID("123"))
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetPost_AuthenticationRequired(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.GetPost(context.Background(), ConvertToPostID("123"))
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestGetUserPosts_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [
			{"id": "1", "text": "Post 1"},
			{"id": "2", "text": "Post 2"}
		],
		"paging": {"cursors": {"after": "cursor123"}}
	}`))

	resp, err := client.GetUserPosts(context.Background(), ConvertToUserID("12345"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 posts, got %d", len(resp.Data))
	}
	if resp.Paging.Cursors == nil || resp.Paging.Cursors.After != "cursor123" {
		t.Error("expected paging cursor")
	}
}

func TestGetUserPosts_InvalidUserID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.GetUserPosts(context.Background(), UserID(""), nil)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestGetPublishingLimits_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{
			"quota_usage": 5,
			"config": {"quota_total": 250, "quota_duration": 86400},
			"reply_quota_usage": 10,
			"reply_config": {"quota_total": 1000, "quota_duration": 86400}
		}]
	}`))

	limits, err := client.GetPublishingLimits(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limits.QuotaUsage != 5 {
		t.Errorf("expected quota_usage 5, got %d", limits.QuotaUsage)
	}
	if limits.Config.QuotaTotal != 250 {
		t.Errorf("expected quota_total 250, got %d", limits.Config.QuotaTotal)
	}
}

func TestGetUserMentions_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"id": "1", "text": "@user mentioned you"}],
		"paging": {}
	}`))

	resp, err := client.GetUserMentions(context.Background(), ConvertToUserID("12345"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 mention, got %d", len(resp.Data))
	}
}

func TestGetUserGhostPosts_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		fields := r.URL.Query().Get("fields")
		if fields != GhostPostFields {
			t.Errorf("expected ghost post fields, got %s", fields)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"data": [{"id": "1", "text": "Ghost!", "ghost_post_status": "active"}],
			"paging": {}
		}`))
	}

	client := testClient(t, http.HandlerFunc(handler))

	resp, err := client.GetUserGhostPosts(context.Background(), ConvertToUserID("12345"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 ghost post, got %d", len(resp.Data))
	}
}
