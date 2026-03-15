package threads

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestPostIterator_MultiplePages(t *testing.T) {
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		switch count {
		case 1:
			_, _ = w.Write([]byte(`{"data":[{"id":"1"},{"id":"2"}],"paging":{"cursors":{"after":"page2"}}}`))
		case 2:
			_, _ = w.Write([]byte(`{"data":[{"id":"3"}],"paging":{}}`))
		default:
			_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	iter := NewPostIterator(client, ConvertToUserID("12345"), &PostsOptions{Limit: 2})
	posts, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 3 {
		t.Errorf("expected 3 posts, got %d", len(posts))
	}
}

func TestPostIterator_EmptyResult(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[],"paging":{}}`))
	iter := NewPostIterator(client, ConvertToUserID("12345"), nil)
	posts, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected 0 posts, got %d", len(posts))
	}
}

func TestPostIterator_Reset(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"id":"1"}],"paging":{}}`))
	iter := NewPostIterator(client, ConvertToUserID("12345"), nil)

	posts1, _ := iter.Collect(context.Background())
	if len(posts1) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts1))
	}
	if iter.HasNext() {
		t.Error("expected iterator to be done")
	}

	iter.Reset()
	if !iter.HasNext() {
		t.Error("expected iterator to have next after reset")
	}
}

func TestPostIterator_PagingAfterFallback(t *testing.T) {
	// Test using Paging.After (deprecated field) instead of Paging.Cursors.After
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if count == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":"1"}],"paging":{"after":"page2_cursor"}}`))
		} else {
			_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	iter := NewPostIterator(client, ConvertToUserID("12345"), nil)
	posts, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}
}

func TestPostIterator_NextWhenDone(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[],"paging":{}}`))
	iter := NewPostIterator(client, ConvertToUserID("12345"), nil)
	_, _ = iter.Collect(context.Background())
	resp, err := iter.Next(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response when done")
	}
}

// ReplyIterator tests

func TestReplyIterator_Success(t *testing.T) {
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if count == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":"r1"},{"id":"r2"}],"paging":{"cursors":{"after":"cursor2"}}}`))
		} else {
			_, _ = w.Write([]byte(`{"data":[{"id":"r3"}],"paging":{}}`))
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	iter := NewReplyIterator(client, ConvertToPostID("post_1"), &RepliesOptions{Limit: 2})
	replies, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(replies) != 3 {
		t.Errorf("expected 3 replies, got %d", len(replies))
	}
}

func TestReplyIterator_NilOpts(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"id":"r1"}],"paging":{}}`))
	iter := NewReplyIterator(client, ConvertToPostID("post_1"), nil)
	if !iter.HasNext() {
		t.Error("expected HasNext to be true")
	}
	replies, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(replies) != 1 {
		t.Errorf("expected 1 reply, got %d", len(replies))
	}
}

func TestReplyIterator_NextWhenDone(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[],"paging":{}}`))
	iter := NewReplyIterator(client, ConvertToPostID("post_1"), nil)
	_, _ = iter.Collect(context.Background())
	if iter.HasNext() {
		t.Error("expected HasNext to be false")
	}
	resp, err := iter.Next(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response when done")
	}
}

func TestReplyIterator_Reset(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"id":"r1"}],"paging":{}}`))
	iter := NewReplyIterator(client, ConvertToPostID("post_1"), nil)
	_, _ = iter.Collect(context.Background())
	if iter.HasNext() {
		t.Error("expected done after collect")
	}
	iter.Reset()
	if !iter.HasNext() {
		t.Error("expected HasNext after reset")
	}
}

func TestReplyIterator_PagingAfterFallback(t *testing.T) {
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if count == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":"r1"}],"paging":{"after":"next"}}`))
		} else {
			_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
		}
	}
	client := testClient(t, http.HandlerFunc(handler))
	iter := NewReplyIterator(client, ConvertToPostID("post_1"), nil)
	replies, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(replies) != 1 {
		t.Errorf("expected 1 reply, got %d", len(replies))
	}
}

func TestReplyIterator_Error(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error","is_transient":true}}`))
	iter := NewReplyIterator(client, ConvertToPostID("post_1"), nil)
	_, err := iter.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

// SearchIterator tests

func TestSearchIterator_Keyword(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"id":"1","text":"match"}],"paging":{}}`))
	iter := NewSearchIterator(client, "test", "keyword", nil)
	posts, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 result, got %d", len(posts))
	}
}

func TestSearchIterator_Tag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"id":"1","text":"tagged"}],"paging":{}}`))
	iter := NewSearchIterator(client, "golang", "tag", nil)
	posts, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 result, got %d", len(posts))
	}
}

func TestSearchIterator_InvalidType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	iter := NewSearchIterator(client, "test", "invalid", nil)
	_, err := iter.Next(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid search type")
	}
}

func TestSearchIterator_Reset(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[{"id":"1"}],"paging":{}}`))
	iter := NewSearchIterator(client, "test", "keyword", nil)
	_, _ = iter.Collect(context.Background())
	if iter.HasNext() {
		t.Error("expected done after collect")
	}
	iter.Reset()
	if !iter.HasNext() {
		t.Error("expected HasNext after reset")
	}
}

func TestSearchIterator_MultiplePages(t *testing.T) {
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if count == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":"1"}],"paging":{"cursors":{"after":"page2"}}}`))
		} else {
			_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
		}
	}
	client := testClient(t, http.HandlerFunc(handler))
	iter := NewSearchIterator(client, "test", "keyword", nil)
	posts, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}
}

func TestSearchIterator_PagingAfterFallback(t *testing.T) {
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if count == 1 {
			_, _ = w.Write([]byte(`{"data":[{"id":"1"}],"paging":{"after":"next"}}`))
		} else {
			_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
		}
	}
	client := testClient(t, http.HandlerFunc(handler))
	iter := NewSearchIterator(client, "test", "keyword", nil)
	posts, err := iter.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}
}

func TestSearchIterator_NextWhenDone(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[],"paging":{}}`))
	iter := NewSearchIterator(client, "test", "keyword", nil)
	_, _ = iter.Collect(context.Background())
	resp, err := iter.Next(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response when done")
	}
}

func TestSearchIterator_Error(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error","is_transient":true}}`))
	iter := NewSearchIterator(client, "test", "keyword", nil)
	_, err := iter.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
