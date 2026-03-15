package threads

import (
	"context"
	"net/http"
	"testing"
)

func TestGetReplies_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [
			{"id": "reply_1", "text": "Great post!", "is_reply": true},
			{"id": "reply_2", "text": "Thanks!", "is_reply": true}
		],
		"paging": {"cursors": {"after": "next_cursor"}}
	}`))

	resp, err := client.GetReplies(context.Background(), ConvertToPostID("post_1"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 replies, got %d", len(resp.Data))
	}
}

func TestGetReplies_InvalidPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetReplies(context.Background(), PostID(""), nil)
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestGetReplies_WithOptions(t *testing.T) {
	reverse := true
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", q.Get("limit"))
		}
		if q.Get("before") != "before_cursor" {
			t.Errorf("expected before=before_cursor, got %s", q.Get("before"))
		}
		if q.Get("after") != "after_cursor" {
			t.Errorf("expected after=after_cursor, got %s", q.Get("after"))
		}
		if q.Get("reverse") != "true" {
			t.Errorf("expected reverse=true, got %s", q.Get("reverse"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.GetReplies(context.Background(), ConvertToPostID("post_1"), &RepliesOptions{
		Limit:   10,
		Before:  "before_cursor",
		After:   "after_cursor",
		Reverse: &reverse,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetReplies_LimitTooLarge(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetReplies(context.Background(), ConvertToPostID("post_1"), &RepliesOptions{Limit: 101})
	if err == nil {
		t.Fatal("expected error for limit > 100")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestBuildRepliesParams_NilOpts(t *testing.T) {
	params, err := buildRepliesParams(nil, 100, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Get("fields") == "" {
		t.Error("expected fields parameter")
	}
}

func TestBuildRepliesParams_AllOptions(t *testing.T) {
	reverse := false
	params, err := buildRepliesParams(&RepliesOptions{
		Limit:   50,
		Before:  "b",
		After:   "a",
		Reverse: &reverse,
	}, 100, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Get("limit") != "50" {
		t.Errorf("expected limit=50, got %s", params.Get("limit"))
	}
	if params.Get("before") != "b" {
		t.Errorf("expected before=b, got %s", params.Get("before"))
	}
	if params.Get("after") != "a" {
		t.Errorf("expected after=a, got %s", params.Get("after"))
	}
	if params.Get("reverse") != "false" {
		t.Errorf("expected reverse=false, got %s", params.Get("reverse"))
	}
}

func TestFetchRepliesData_404(t *testing.T) {
	client := testClient(t, jsonHandler(404, `{"error":{"message":"not found"}}`))
	_, err := client.fetchRepliesData("/test/replies", nil, ConvertToPostID("post_1"), "replies")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFetchRepliesData_403(t *testing.T) {
	client := testClient(t, jsonHandler(403, `{"error":{"message":"forbidden"}}`))
	_, err := client.fetchRepliesData("/test/replies", nil, ConvertToPostID("post_1"), "replies")
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestFetchRepliesData_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.fetchRepliesData("/test/replies", nil, ConvertToPostID("post_1"), "replies")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetConversation_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"id": "msg_1", "text": "Thread message"}],
		"paging": {}
	}`))

	resp, err := client.GetConversation(context.Background(), ConvertToPostID("post_1"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 message, got %d", len(resp.Data))
	}
}

func TestGetConversation_InvalidPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetConversation(context.Background(), PostID(""), nil)
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestGetConversation_WithOptions(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"id":"1"}],"paging":{}}`))
	_, err := client.GetConversation(context.Background(), ConvertToPostID("post_1"), &RepliesOptions{
		Limit:  20,
		Before: "b",
		After:  "a",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetConversation_LimitTooLarge(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetConversation(context.Background(), ConvertToPostID("post_1"), &RepliesOptions{Limit: 101})
	if err == nil {
		t.Fatal("expected error for limit > 100")
	}
}

func TestHideReply_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"success":true}`))
	err := client.HideReply(context.Background(), ConvertToPostID("reply_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHideReply_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.HideReply(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty reply ID")
	}
}

func TestHideReply_404(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	err := client.HideReply(context.Background(), ConvertToPostID("reply_1"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestHideReply_403(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":{"message":"forbidden"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	err := client.HideReply(context.Background(), ConvertToPostID("reply_1"))
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestHideReply_500(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":{"message":"server error"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	err := client.HideReply(context.Background(), ConvertToPostID("reply_1"))
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestUnhideReply_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"success":true}`))
	err := client.UnhideReply(context.Background(), ConvertToPostID("reply_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnhideReply_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.UnhideReply(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty reply ID")
	}
}

func TestManageReplyVisibility_UnparseableBody(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`not json`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	config := testClientConfig(t, http.HandlerFunc(handler))
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)
	// Should still succeed (200 status) even with unparseable body
	err := client.HideReply(context.Background(), ConvertToPostID("reply_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPendingReplies_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"id": "pending_1", "text": "Awaiting approval", "reply_approval_status": "pending"}],
		"paging": {}
	}`))

	resp, err := client.GetPendingReplies(context.Background(), ConvertToPostID("post_1"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 pending reply, got %d", len(resp.Data))
	}
}

func TestGetPendingReplies_InvalidPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPendingReplies(context.Background(), PostID(""), nil)
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestGetPendingReplies_WithOptions(t *testing.T) {
	reverse := true
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "5" {
			t.Errorf("expected limit=5, got %s", q.Get("limit"))
		}
		if q.Get("before") != "b" {
			t.Errorf("expected before=b, got %s", q.Get("before"))
		}
		if q.Get("after") != "a" {
			t.Errorf("expected after=a, got %s", q.Get("after"))
		}
		if q.Get("reverse") != "true" {
			t.Errorf("expected reverse=true, got %s", q.Get("reverse"))
		}
		if q.Get("approval_status") != "pending" {
			t.Errorf("expected approval_status=pending, got %s", q.Get("approval_status"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.GetPendingReplies(context.Background(), ConvertToPostID("post_1"), &PendingRepliesOptions{
		Limit:          5,
		Before:         "b",
		After:          "a",
		Reverse:        &reverse,
		ApprovalStatus: ApprovalStatusPending,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPendingReplies_IgnoredApprovalStatus(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[],"paging":{}}`))
	_, err := client.GetPendingReplies(context.Background(), ConvertToPostID("post_1"), &PendingRepliesOptions{
		ApprovalStatus: ApprovalStatusIgnored,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPendingReplies_InvalidApprovalStatus(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPendingReplies(context.Background(), ConvertToPostID("post_1"), &PendingRepliesOptions{
		ApprovalStatus: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid approval status")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestGetPendingReplies_LimitTooLarge(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.GetPendingReplies(context.Background(), ConvertToPostID("post_1"), &PendingRepliesOptions{
		Limit: 101,
	})
	if err == nil {
		t.Fatal("expected error for limit > 100")
	}
}

func TestApprovePendingReply_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"success":true}`))
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApprovePendingReply_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.ApprovePendingReply(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty reply ID")
	}
}

func TestApprovePendingReply_404(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestApprovePendingReply_403(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":{"message":"forbidden"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestApprovePendingReply_500(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"error":{"message":"server error"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestIgnorePendingReply_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"success":true}`))
	err := client.IgnorePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIgnorePendingReply_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	err := client.IgnorePendingReply(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty reply ID")
	}
}

func TestManagePendingReply_UnparseableBody(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`not json`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	config := testClientConfig(t, http.HandlerFunc(handler))
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManagePendingReply_WithLogger(t *testing.T) {
	handler := jsonHandler(200, `{"success":true}`)
	config := testClientConfig(t, handler)
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManageReplyVisibility_WithLogger(t *testing.T) {
	handler := jsonHandler(200, `{"success":true}`)
	config := testClientConfig(t, handler)
	config.Logger = &noopLogger{}
	client := testClientWithConfig(t, config)
	err := client.HideReply(context.Background(), ConvertToPostID("reply_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchRepliesData_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"id":"r1","text":"reply"}],
		"paging": {"cursors":{"after":"c1"}}
	}`))
	resp, err := client.fetchRepliesData("/test/replies", nil, ConvertToPostID("post_1"), "replies")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 reply, got %d", len(resp.Data))
	}
}

func TestFetchRepliesData_InvalidJSON(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))
	_, err := client.fetchRepliesData("/test/replies", nil, ConvertToPostID("post_1"), "replies")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetReplies_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetReplies(context.Background(), ConvertToPostID("post_1"), nil)
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestGetConversation_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetConversation(context.Background(), ConvertToPostID("post_1"), nil)
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestGetPendingReplies_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	_, err := client.GetPendingReplies(context.Background(), ConvertToPostID("post_1"), nil)
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestManageReplyVisibility_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	err := client.HideReply(context.Background(), ConvertToPostID("reply_1"))
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestManagePendingReply_NoAuth(t *testing.T) {
	client := testClientNoAuth(t, jsonHandler(200, `{}`))
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err == nil {
		t.Fatal("expected error for unauthenticated client")
	}
}

func TestManagePendingReply_EmptyBody(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			// Empty body
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
