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
