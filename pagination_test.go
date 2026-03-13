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

func TestSearchIterator_InvalidType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	iter := NewSearchIterator(client, "test", "invalid", nil)
	_, err := iter.Next(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid search type")
	}
}
