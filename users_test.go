package threads

import (
	"context"
	"net/http"
	"testing"
)

func TestGetUser_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"id": "12345",
		"username": "testuser",
		"name": "Test User",
		"followers_count": 100
	}`))

	user, err := client.GetUser(context.Background(), ConvertToUserID("12345"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("expected testuser, got %s", user.Username)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetUser(context.Background(), UserID(""))
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestGetUser_404(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found"}}`))
	_, err := client.GetUser(context.Background(), ConvertToUserID("99999"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetUser_403(t *testing.T) {
	client := testClient(t, jsonHandler(403, `{"error":{"message":"forbidden"}}`))
	_, err := client.GetUser(context.Background(), ConvertToUserID("99999"))
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestGetUser_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.GetUser(context.Background(), ConvertToUserID("12345"))
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetMe_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"12345","username":"me","name":"My Name"}`))

	user, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "me" {
		t.Errorf("expected 'me', got %s", user.Username)
	}
}

func TestGetMe_NoUserID(t *testing.T) {
	// Create a client with no user ID in token info
	handler := jsonHandler(200, `{"id":"12345","username":"me"}`)
	client := testClient(t, handler)
	// Clear the user ID from token info
	client.mu.Lock()
	client.tokenInfo.UserID = ""
	client.mu.Unlock()

	_, err := client.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error when user ID is not available")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestLookupPublicProfile_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"username": "publicuser",
		"name": "Public User",
		"is_verified": true,
		"follower_count": 5000
	}`))

	user, err := client.LookupPublicProfile(context.Background(), "publicuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "publicuser" {
		t.Errorf("expected publicuser, got %s", user.Username)
	}
	if !user.IsVerified {
		t.Error("expected verified user")
	}
}

func TestLookupPublicProfile_EmptyUsername(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.LookupPublicProfile(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty username")
	}
}

func TestLookupPublicProfile_WithAtSymbol(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username != "testuser" {
			t.Errorf("expected username without @, got %s", username)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"username":"testuser","name":"Test"}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.LookupPublicProfile(context.Background(), "@testuser")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLookupPublicProfile_404(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found"}}`))
	_, err := client.LookupPublicProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestLookupPublicProfile_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.LookupPublicProfile(context.Background(), "user")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetUserFields_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		fields := r.URL.Query().Get("fields")
		if fields == "" {
			t.Error("expected fields parameter")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":"12345","username":"testuser"}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	user, err := client.GetUserFields(context.Background(), ConvertToUserID("12345"), []string{"id", "username"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "12345" {
		t.Errorf("expected 12345, got %s", user.ID)
	}
}

func TestGetUserFields_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetUserFields(context.Background(), UserID(""), []string{"id"})
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestGetUserFields_DefaultFields(t *testing.T) {
	// When no fields are specified, defaults should be used
	client := testClient(t, jsonHandler(200, `{"id":"12345","username":"testuser"}`))
	user, err := client.GetUserFields(context.Background(), ConvertToUserID("12345"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "12345" {
		t.Errorf("expected 12345, got %s", user.ID)
	}
}

func TestGetUserFields_InvalidFields(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetUserFields(context.Background(), ConvertToUserID("12345"), []string{"invalid_field"})
	if err == nil {
		t.Fatal("expected error for invalid fields")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestGetUserFields_404(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found"}}`))
	_, err := client.GetUserFields(context.Background(), ConvertToUserID("99999"), []string{"id"})
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetUserFields_403(t *testing.T) {
	client := testClient(t, jsonHandler(403, `{"error":{"message":"forbidden"}}`))
	_, err := client.GetUserFields(context.Background(), ConvertToUserID("99999"), []string{"id"})
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestGetUserFields_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.GetUserFields(context.Background(), ConvertToUserID("12345"), []string{"id"})
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetPublicProfilePosts_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"id":"1","text":"Hello"},{"id":"2","text":"World"}],
		"paging": {"cursors":{"after":"cursor1"}}
	}`))

	resp, err := client.GetPublicProfilePosts(context.Background(), "testuser", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 posts, got %d", len(resp.Data))
	}
}

func TestGetPublicProfilePosts_EmptyUsername(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPublicProfilePosts(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty username")
	}
}

func TestGetPublicProfilePosts_WithAtSymbol(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username != "testuser" {
			t.Errorf("expected username without @, got %s", username)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.GetPublicProfilePosts(context.Background(), "@testuser", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPublicProfilePosts_WithOptions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", q.Get("limit"))
		}
		if q.Get("before") != "cursor_before" {
			t.Errorf("expected before=cursor_before, got %s", q.Get("before"))
		}
		if q.Get("after") != "cursor_after" {
			t.Errorf("expected after=cursor_after, got %s", q.Get("after"))
		}
		if q.Get("since") != "1000000" {
			t.Errorf("expected since=1000000, got %s", q.Get("since"))
		}
		if q.Get("until") != "2000000" {
			t.Errorf("expected until=2000000, got %s", q.Get("until"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.GetPublicProfilePosts(context.Background(), "user", &PostsOptions{
		Limit:  10,
		Before: "cursor_before",
		After:  "cursor_after",
		Since:  1000000,
		Until:  2000000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPublicProfilePosts_LimitTooLarge(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPublicProfilePosts(context.Background(), "user", &PostsOptions{Limit: 101})
	if err == nil {
		t.Fatal("expected error for limit > 100")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestGetPublicProfilePosts_404(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found"}}`))
	_, err := client.GetPublicProfilePosts(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetPublicProfilePosts_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.GetPublicProfilePosts(context.Background(), "user", nil)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetUserReplies_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"id":"r1","text":"Reply 1","is_reply":true},{"id":"r2","text":"Reply 2","is_reply":true}],
		"paging": {}
	}`))

	resp, err := client.GetUserReplies(context.Background(), ConvertToUserID("12345"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 replies, got %d", len(resp.Data))
	}
}

func TestGetUserReplies_InvalidUserID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetUserReplies(context.Background(), UserID(""), nil)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestGetUserReplies_WithOptions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "5" {
			t.Errorf("expected limit=5, got %s", q.Get("limit"))
		}
		if q.Get("before") != "b_cursor" {
			t.Errorf("expected before=b_cursor, got %s", q.Get("before"))
		}
		if q.Get("after") != "a_cursor" {
			t.Errorf("expected after=a_cursor, got %s", q.Get("after"))
		}
		if q.Get("since") != "100" {
			t.Errorf("expected since=100, got %s", q.Get("since"))
		}
		if q.Get("until") != "200" {
			t.Errorf("expected until=200, got %s", q.Get("until"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.GetUserReplies(context.Background(), ConvertToUserID("12345"), &PostsOptions{
		Limit:  5,
		Before: "b_cursor",
		After:  "a_cursor",
		Since:  100,
		Until:  200,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetUserReplies_LimitTooLarge(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetUserReplies(context.Background(), ConvertToUserID("12345"), &PostsOptions{Limit: 101})
	if err == nil {
		t.Fatal("expected error for limit > 100")
	}
}

func TestGetUserReplies_404(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found"}}`))
	_, err := client.GetUserReplies(context.Background(), ConvertToUserID("99999"), nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestGetUserReplies_403(t *testing.T) {
	client := testClient(t, jsonHandler(403, `{"error":{"message":"forbidden"}}`))
	_, err := client.GetUserReplies(context.Background(), ConvertToUserID("99999"), nil)
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestGetUserReplies_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.GetUserReplies(context.Background(), ConvertToUserID("12345"), nil)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetUser_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetUser(context.Background(), ConvertToUserID("12345"))
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestGetUser_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json at all`))
	_, err := client.GetUser(context.Background(), ConvertToUserID("12345"))
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetMe_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestLookupPublicProfile_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.LookupPublicProfile(context.Background(), "testuser")
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestLookupPublicProfile_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))
	_, err := client.LookupPublicProfile(context.Background(), "testuser")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetUserFields_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetUserFields(context.Background(), ConvertToUserID("12345"), []string{"id"})
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestGetUserFields_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))
	_, err := client.GetUserFields(context.Background(), ConvertToUserID("12345"), []string{"id"})
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetPublicProfilePosts_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetPublicProfilePosts(context.Background(), "testuser", nil)
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestGetPublicProfilePosts_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))
	_, err := client.GetPublicProfilePosts(context.Background(), "testuser", nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetUserReplies_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetUserReplies(context.Background(), ConvertToUserID("12345"), nil)
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestGetUserReplies_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))
	_, err := client.GetUserReplies(context.Background(), ConvertToUserID("12345"), nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}
