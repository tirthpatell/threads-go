package threads

import (
	"context"
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

func TestUnhideReply_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"success":true}`))
	err := client.UnhideReply(context.Background(), ConvertToPostID("reply_1"))
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

func TestApprovePendingReply_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"success":true}`))
	err := client.ApprovePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIgnorePendingReply_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"success":true}`))
	err := client.IgnorePendingReply(context.Background(), ConvertToPostID("pending_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
