package threads

import (
	"context"
	"net/http"
	"testing"
)

func TestDeletePost_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"post_1","owner":{"id":"12345"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"success":true}`))
	}

	client := testClient(t, http.HandlerFunc(handler))

	err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePost_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.DeletePost(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestDeletePost_NotFound(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found","type":"OAuthException","code":100}}`))
	client.config.RetryConfig.MaxRetries = 0

	err := client.DeletePost(context.Background(), ConvertToPostID("nonexistent"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
