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
