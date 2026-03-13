package threads

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCreateTextPost_Success(t *testing.T) {
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"post_1"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"container_1"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/container_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"container_1","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/post_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"post_1","text":"Hello","media_type":"TEXT","permalink":"https://threads.net/p/1"}`))
		default:
			t.Logf("call %d: unexpected request: %s %s", count, r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text: "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "post_1" {
		t.Errorf("expected post ID post_1, got %s", post.ID)
	}
}

func TestCreateTextPost_EmptyText(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text: "",
	})
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestCreateTextPost_TextTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longText := make([]byte, MaxTextLength+1)
	for i := range longText {
		longText[i] = 'a'
	}

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text: string(longText),
	})
	if err == nil {
		t.Fatal("expected error for text too long")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateTextPost_AutoPublish(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.PostForm.Get("auto_publish_text") != "true" {
				t.Error("expected auto_publish_text=true")
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"post_auto"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/post_auto"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"post_auto","text":"Auto","media_type":"TEXT"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text:            "Auto",
		AutoPublishText: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "post_auto" {
		t.Errorf("expected post ID post_auto, got %s", post.ID)
	}
}

func TestCreateImagePost_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/img_container"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_container","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/img_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_post","media_type":"IMAGE","media_url":"https://example.com/img.jpg"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateImagePost(context.Background(), &ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		Text:     "Check this out",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "img_post" {
		t.Errorf("expected img_post, got %s", post.ID)
	}
}

func TestCreateImagePost_MissingURL(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateImagePost(context.Background(), &ImagePostContent{ImageURL: ""})
	if err == nil {
		t.Fatal("expected error for missing image URL")
	}
}

func TestCreateVideoPost_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/vid_container"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_container","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/vid_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_post","media_type":"VIDEO"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateVideoPost(context.Background(), &VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "vid_post" {
		t.Errorf("expected vid_post, got %s", post.ID)
	}
}

func TestCreateVideoPost_MissingURL(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateVideoPost(context.Background(), &VideoPostContent{VideoURL: ""})
	if err == nil {
		t.Fatal("expected error for missing video URL")
	}
}

func TestRepostPost_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/original_post/repost"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"repost_1"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repost_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"repost_1","media_type":"TEXT"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.RepostPost(context.Background(), ConvertToPostID("original_post"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "repost_1" {
		t.Errorf("expected repost_1, got %s", post.ID)
	}
}

func TestRepostPost_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.RepostPost(context.Background(), PostID(""))
	if err == nil {
		t.Fatal("expected error for empty post ID")
	}
}

func TestGetContainerStatus_Success(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"container_1","status":"FINISHED"}`))

	status, err := client.GetContainerStatus(context.Background(), ConvertToContainerID("container_1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "FINISHED" {
		t.Errorf("expected FINISHED, got %s", status.Status)
	}
}

func TestGetContainerStatus_InvalidID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.GetContainerStatus(context.Background(), ContainerID(""))
	if err == nil {
		t.Fatal("expected error for empty container ID")
	}
}
