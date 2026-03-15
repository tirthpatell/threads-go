package threads

import (
	"context"
	"net/http"
	"testing"
)

func TestKeywordSearch_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{
		"data": [{"id": "1", "text": "Go programming"}, {"id": "2", "text": "Golang tips"}],
		"paging": {}
	}`))

	resp, err := client.KeywordSearch(context.Background(), "golang", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Data))
	}
}

func TestKeywordSearch_EmptyQuery(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.KeywordSearch(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestKeywordSearch_WithOptions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("q") != "test" {
			t.Errorf("expected q=test, got q=%s", q.Get("q"))
		}
		if q.Get("search_type") != "TOP" {
			t.Errorf("expected search_type=TOP, got %s", q.Get("search_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		SearchType: SearchTypeTop,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeywordSearch_WithAllOptions(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("search_mode") != "KEYWORD" {
			t.Errorf("expected search_mode=KEYWORD, got %s", q.Get("search_mode"))
		}
		if q.Get("media_type") != "IMAGE" {
			t.Errorf("expected media_type=IMAGE, got %s", q.Get("media_type"))
		}
		if q.Get("author_username") != "testuser" {
			t.Errorf("expected author_username=testuser, got %s", q.Get("author_username"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", q.Get("limit"))
		}
		if q.Get("since") != "1688540400" {
			t.Errorf("expected since=1688540400, got %s", q.Get("since"))
		}
		if q.Get("until") != "1700000000" {
			t.Errorf("expected until=1700000000, got %s", q.Get("until"))
		}
		if q.Get("before") != "b" {
			t.Errorf("expected before=b, got %s", q.Get("before"))
		}
		if q.Get("after") != "a" {
			t.Errorf("expected after=a, got %s", q.Get("after"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		SearchMode:     SearchModeKeyword,
		MediaType:      "IMAGE",
		AuthorUsername: "testuser",
		Limit:          10,
		Since:          1688540400,
		Until:          1700000000,
		Before:         "b",
		After:          "a",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeywordSearch_InvalidMediaType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		MediaType: "AUDIO",
	})
	if err == nil {
		t.Fatal("expected error for invalid media type")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestKeywordSearch_LimitTooLarge(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		Limit: 101,
	})
	if err == nil {
		t.Fatal("expected error for limit > 100")
	}
}

func TestKeywordSearch_InvalidSinceTimestamp(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		Since: 100, // way too early
	})
	if err == nil {
		t.Fatal("expected error for invalid since timestamp")
	}
}

func TestKeywordSearch_AuthorUsernameWithAt(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("author_username") != "user" {
			t.Errorf("expected author_username=user (without @), got %s", q.Get("author_username"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"data":[],"paging":{}}`))
	}
	client := testClient(t, http.HandlerFunc(handler))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		AuthorUsername: "@user",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeywordSearch_EmptyAuthorUsername(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		AuthorUsername: "@",
	})
	if err == nil {
		t.Fatal("expected error for empty author username after trimming @")
	}
}

func TestKeywordSearch_500(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"server error"}}`))
	_, err := client.KeywordSearch(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestKeywordSearch_VideoMediaType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[],"paging":{}}`))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		MediaType: "video", // lowercase should be uppercased
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKeywordSearch_TextMediaType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"data":[],"paging":{}}`))
	_, err := client.KeywordSearch(context.Background(), "test", &SearchOptions{
		MediaType: "text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
