package threads

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestCreateReply_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.PostForm.Get("reply_to_id") != "parent_post_123" {
				t.Errorf("expected reply_to_id=parent_post_123, got %s", r.PostForm.Get("reply_to_id"))
			}
			if r.PostForm.Get("media_type") != "TEXT" {
				t.Errorf("expected media_type=TEXT, got %s", r.PostForm.Get("media_type"))
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/reply_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_post","text":"My reply","media_type":"TEXT"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateReply(context.Background(), &PostContent{
		Text:    "My reply",
		ReplyTo: "parent_post_123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "reply_post" {
		t.Errorf("expected reply_post, got %s", post.ID)
	}
}

func TestCreateReply_NilContent(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateReply(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateReply_EmptyReplyTo(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateReply(context.Background(), &PostContent{
		Text:    "My reply",
		ReplyTo: "",
	})
	if err == nil {
		t.Fatal("expected error for empty reply_to")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateReply_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.CreateReply(context.Background(), &PostContent{
		Text:    "My reply",
		ReplyTo: "parent_post_123",
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestCreateReply_ContainerCreateError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
		} else {
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateReply(context.Background(), &PostContent{
		Text:    "My reply",
		ReplyTo: "parent_post_123",
	})
	if err == nil {
		t.Fatal("expected error when container creation fails")
	}
	if !strings.Contains(err.Error(), "failed to create reply container") {
		t.Errorf("expected reply container error, got: %v", err)
	}
}

func TestCreateReply_DefaultMediaType(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			// Default media type should be TEXT
			if r.PostForm.Get("media_type") != "TEXT" {
				t.Errorf("expected default media_type=TEXT, got %s", r.PostForm.Get("media_type"))
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/reply_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_post","text":"reply","media_type":"TEXT"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	// No MediaType set - should default to TEXT
	post, err := client.CreateReply(context.Background(), &PostContent{
		Text:    "reply",
		ReplyTo: "parent_post_123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "reply_post" {
		t.Errorf("expected reply_post, got %s", post.ID)
	}
}

func TestCreateReply_ContextCancelled(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads") {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_container"}`))
		} else {
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the reply delay

	_, err := client.CreateReply(ctx, &PostContent{
		Text:    "My reply",
		ReplyTo: "parent_post_123",
	})
	// CreateReply creates container, then waits ReplyPublishDelay.
	// If context is already cancelled, it should fail during the POST or the select.
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestReplyToPost_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.PostForm.Get("reply_to_id") != "target_post" {
				t.Errorf("expected reply_to_id=target_post, got %s", r.PostForm.Get("reply_to_id"))
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/reply_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_post","text":"reply text","media_type":"TEXT"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.ReplyToPost(context.Background(), ConvertToPostID("target_post"), &PostContent{
		Text: "reply text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "reply_post" {
		t.Errorf("expected reply_post, got %s", post.ID)
	}
}

func TestReplyToPost_InvalidPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.ReplyToPost(context.Background(), PostID(""), &PostContent{Text: "hello"})
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestReplyToPost_NilContent(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.ReplyToPost(context.Background(), ConvertToPostID("target_post"), nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateReply_PublishError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"publish failed","type":"OAuthException","code":100}}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"reply_container"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateReply(context.Background(), &PostContent{
		Text:    "My reply",
		ReplyTo: "parent_post_123",
	})
	if err == nil {
		t.Fatal("expected error when publish fails")
	}
	if !strings.Contains(err.Error(), "failed to publish reply") {
		t.Errorf("expected publish error, got: %v", err)
	}
}
