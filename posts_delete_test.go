package threads

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// ownedPostHandler serves GETs that make post_1 look owned by user 12345
// and routes DELETE /post_1 to the given status/body.
func ownedPostHandler(deleteStatus int, deleteBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "DELETE" {
			w.WriteHeader(deleteStatus)
			_, _ = w.Write([]byte(deleteBody))
			return
		}
		w.WriteHeader(200)
		if strings.HasPrefix(r.URL.Path, "/12345") {
			_, _ = w.Write([]byte(`{"id":"12345","username":"me"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"post_1","username":"me","owner":{"id":"12345"}}`))
	}
}

func TestDeletePost_Success(t *testing.T) {
	handler := ownedPostHandler(200, `{"success":true,"deleted_id":"post_1"}`)

	client := testClient(t, handler)

	deletedID, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "post_1" {
		t.Errorf("expected deleted_id %q, got %q", "post_1", deletedID)
	}
}

func TestDeletePost_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.DeletePost(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestDeletePost_NotFound(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found","type":"OAuthException","code":100}}`))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.DeletePost(context.Background(), ConvertToPostID("nonexistent"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !IsAPIError(err) {
		t.Errorf("expected APIError, got %T", err)
	}
}

func TestDeletePostWithConfirmation_Success(t *testing.T) {
	client := testClient(t, ownedPostHandler(200, `{"success":true}`))

	confirmed := false
	_, err := client.DeletePostWithConfirmation(context.Background(), ConvertToPostID("post_1"), func(post *Post) bool {
		confirmed = true
		return true
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confirmed {
		t.Error("confirmation callback was not called")
	}
}

func TestDeletePostWithConfirmation_Cancelled(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":"post_1","text":"hello"}`))
	}

	client := testClient(t, http.HandlerFunc(handler))

	_, err := client.DeletePostWithConfirmation(context.Background(), ConvertToPostID("post_1"), func(post *Post) bool {
		return false // user cancels
	})
	if err == nil {
		t.Fatal("expected error when user cancels deletion")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestDeletePostWithConfirmation_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.DeletePostWithConfirmation(context.Background(), PostID(""), func(post *Post) bool {
		return true
	})
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestDeletePostWithConfirmation_NilCallback(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.DeletePostWithConfirmation(context.Background(), ConvertToPostID("post_1"), nil)
	if err == nil {
		t.Fatal("expected error for nil callback")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestDeletePostWithConfirmation_GetPostError(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found","type":"OAuthException","code":100}}`))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.DeletePostWithConfirmation(context.Background(), ConvertToPostID("nonexistent"), func(post *Post) bool {
		return true
	})
	if err == nil {
		t.Fatal("expected error when post not found")
	}
}

func TestDeletePost_Forbidden(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "DELETE" {
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":{"message":"access denied","type":"OAuthException","code":200}}`))
			return
		}
		// GET requests for post and user info
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":"post_1","owner":{"id":"12345"}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestDeletePost_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestDeletePost_ServerError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "DELETE" {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":{"message":"internal error","type":"OAuthException","code":2}}`))
			return
		}
		// GET requests for post and user info
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":"post_1","owner":{"id":"12345"}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestDeletePost_Delete404(t *testing.T) {
	client := testClient(t, ownedPostHandler(404, `{"error":{"message":"not found","type":"OAuthException","code":100}}`))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err == nil {
		t.Fatal("expected error for 404 DELETE")
	}
	// The DeletePost method returns a ValidationError for 404 on the DELETE call
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "Post not found") {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestDeletePost_WithLogger(t *testing.T) {
	client := testClient(t, ownedPostHandler(200, `{"success":true}`))
	client.config.Logger = &noopLogger{}

	_, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePost_MalformedDeleteResponse(t *testing.T) {
	client := testClient(t, ownedPostHandler(200, `not json`))

	// Should succeed even with malformed response (200 status assumed success)
	_, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePost_MalformedDeleteResponseWithLogger(t *testing.T) {
	client := testClient(t, ownedPostHandler(200, `not json`))
	client.config.Logger = &noopLogger{}

	// Should succeed even with malformed response and logger
	_, err := client.DeletePost(context.Background(), ConvertToPostID("post_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePostOwnership_DifferentUser(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if strings.HasPrefix(r.URL.Path, "/12345") {
			// GetMe -> GetUser returns our user
			_, _ = w.Write([]byte(`{"id":"12345","username":"me"}`))
		} else {
			// GetPost returns a post owned by someone else (different owner ID)
			_, _ = w.Write([]byte(`{"id":"post_1","username":"other_user","owner":{"id":"99999"}}`))
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	err := client.validatePostOwnership(context.Background(), ConvertToPostID("post_1"))
	if err == nil {
		t.Fatal("expected error for different user ownership")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

// TestValidatePostOwnership_BothUsernamesEmpty: empty identifiers on both
// sides must not compare equal.
func TestValidatePostOwnership_BothUsernamesEmpty(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if strings.HasPrefix(r.URL.Path, "/12345") {
			// GetMe returns a user with no username set and no id echoed
			_, _ = w.Write([]byte(`{"username":""}`))
		} else {
			// GetPost returns a post with no owner object and no username
			_, _ = w.Write([]byte(`{"id":"post_1"}`))
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	err := client.validatePostOwnership(context.Background(), ConvertToPostID("post_1"))
	if err == nil {
		t.Fatal("expected error when neither ID nor username is available on either side")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestValidatePostOwnership_MatchingOwnerID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if strings.HasPrefix(r.URL.Path, "/12345") {
			_, _ = w.Write([]byte(`{"id":"12345","username":"me"}`))
		} else {
			// Different username but same owner ID — ownership holds.
			_, _ = w.Write([]byte(`{"id":"post_1","username":"old_handle","owner":{"id":"12345"}}`))
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	if err := client.validatePostOwnership(context.Background(), ConvertToPostID("post_1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
