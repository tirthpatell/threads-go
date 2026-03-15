package threads

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

func TestGetContainerStatus_APIError(t *testing.T) {
	client := testClient(t, jsonHandler(500, `{"error":{"message":"Server error","type":"OAuthException","code":2}}`))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.GetContainerStatus(context.Background(), ConvertToContainerID("container_1"))
	if err == nil {
		t.Fatal("expected error for API error")
	}
}

func TestGetContainerStatus_MissingStatus(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"container_1"}`))

	_, err := client.GetContainerStatus(context.Background(), ConvertToContainerID("container_1"))
	if err == nil {
		t.Fatal("expected error for missing status")
	}
}

func TestGetContainerStatus_MissingID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"status":"FINISHED"}`))

	_, err := client.GetContainerStatus(context.Background(), ConvertToContainerID("container_1"))
	if err == nil {
		t.Fatal("expected error for missing ID in response")
	}
}

func TestCreateCarouselPost_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_1","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_2"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_2","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/carousel_container"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_container","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/carousel_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_post","media_type":"CAROUSEL"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateCarouselPost(context.Background(), &CarouselPostContent{
		Text:     "My carousel",
		Children: []string{"child_1", "child_2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "carousel_post" {
		t.Errorf("expected carousel_post, got %s", post.ID)
	}
}

func TestCreateCarouselPost_EmptyChildren(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateCarouselPost(context.Background(), &CarouselPostContent{
		Children: []string{},
	})
	if err == nil {
		t.Fatal("expected error for empty children")
	}
}

func TestCreateCarouselPost_NilContent(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateCarouselPost(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
}

func TestCreateCarouselPost_ChildNotReady(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_1","status":"ERROR","error_message":"processing failed"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_2"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_2","status":"FINISHED"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	_, err := client.CreateCarouselPost(context.Background(), &CarouselPostContent{
		Text:     "Carousel",
		Children: []string{"child_1", "child_2"},
	})
	if err == nil {
		t.Fatal("expected error when child container fails")
	}
	if !strings.Contains(err.Error(), "child container") {
		t.Errorf("expected child container error, got: %v", err)
	}
}

func TestCreateCarouselPost_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.CreateCarouselPost(context.Background(), &CarouselPostContent{
		Text:     "Carousel",
		Children: []string{"child_1", "child_2"},
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestCreateCarouselPost_ContainerCreateError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_1","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_2"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_2","status":"FINISHED"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateCarouselPost(context.Background(), &CarouselPostContent{
		Text:     "Carousel",
		Children: []string{"child_1", "child_2"},
	})
	if err == nil {
		t.Fatal("expected error when container creation fails")
	}
	if !strings.Contains(err.Error(), "failed to create carousel container") {
		t.Errorf("expected carousel container error, got: %v", err)
	}
}

func TestCreateQuotePost_TextQuote(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"quote_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.PostForm.Get("quote_post_id") != "original_123" {
				t.Errorf("expected quote_post_id=original_123, got %s", r.PostForm.Get("quote_post_id"))
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"quote_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/quote_container"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"quote_container","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/quote_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"quote_post","text":"Quote this","is_quote_post":true}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateQuotePost(context.Background(), &TextPostContent{
		Text: "Quote this",
	}, "original_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "quote_post" {
		t.Errorf("expected quote_post, got %s", post.ID)
	}
}

func TestCreateQuotePost_ImageQuote(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_quote_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_quote_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/img_quote_container"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_quote_container","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/img_quote_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"img_quote_post","media_type":"IMAGE"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateQuotePost(context.Background(), &ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
	}, "original_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "img_quote_post" {
		t.Errorf("expected img_quote_post, got %s", post.ID)
	}
}

func TestCreateQuotePost_VideoQuote(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_quote_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_quote_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/vid_quote_container"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_quote_container","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/vid_quote_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"vid_quote_post","media_type":"VIDEO"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateQuotePost(context.Background(), &VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
	}, "original_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "vid_quote_post" {
		t.Errorf("expected vid_quote_post, got %s", post.ID)
	}
}

func TestCreateQuotePost_CarouselQuote(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_quote_post"}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_quote_container"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_1","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/child_2"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"child_2","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/carousel_quote_container"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_quote_container","status":"FINISHED"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/carousel_quote_post"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"carousel_quote_post","media_type":"CAROUSEL"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	post, err := client.CreateQuotePost(context.Background(), &CarouselPostContent{
		Children: []string{"child_1", "child_2"},
	}, "original_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post.ID != "carousel_quote_post" {
		t.Errorf("expected carousel_quote_post, got %s", post.ID)
	}
}

func TestCreateQuotePost_EmptyQuotedID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateQuotePost(context.Background(), &TextPostContent{Text: "Hello"}, "")
	if err == nil {
		t.Fatal("expected error for empty quoted post ID")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateQuotePost_UnsupportedContentType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateQuotePost(context.Background(), "not a valid content type", "original_123")
	if err == nil {
		t.Fatal("expected error for unsupported content type")
	}
	if !strings.Contains(err.Error(), "unsupported content type") {
		t.Errorf("expected unsupported content type error, got: %v", err)
	}
}

func TestCreateMediaContainer_ImageSuccess(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads") {
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.PostForm.Get("is_carousel_item") != "true" {
				t.Error("expected is_carousel_item=true")
			}
			if r.PostForm.Get("media_type") != "IMAGE" {
				t.Errorf("expected media_type=IMAGE, got %s", r.PostForm.Get("media_type"))
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"media_container_1"}`))
		} else {
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	containerID, err := client.CreateMediaContainer(context.Background(), "IMAGE", "https://example.com/img.jpg", "alt text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(containerID) != "media_container_1" {
		t.Errorf("expected media_container_1, got %s", containerID)
	}
}

func TestCreateMediaContainer_VideoSuccess(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads") {
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			if r.PostForm.Get("media_type") != "VIDEO" {
				t.Errorf("expected media_type=VIDEO, got %s", r.PostForm.Get("media_type"))
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"video_container_1"}`))
		} else {
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	containerID, err := client.CreateMediaContainer(context.Background(), "video", "https://example.com/vid.mp4", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(containerID) != "video_container_1" {
		t.Errorf("expected video_container_1, got %s", containerID)
	}
}

func TestCreateMediaContainer_EmptyMediaType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateMediaContainer(context.Background(), "", "https://example.com/img.jpg", "")
	if err == nil {
		t.Fatal("expected error for empty media type")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateMediaContainer_EmptyMediaURL(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateMediaContainer(context.Background(), "IMAGE", "", "")
	if err == nil {
		t.Fatal("expected error for empty media URL")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateMediaContainer_InvalidMediaType(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateMediaContainer(context.Background(), "AUDIO", "https://example.com/audio.mp3", "")
	if err == nil {
		t.Fatal("expected error for invalid media type")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCreateMediaContainer_InvalidURL(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.CreateMediaContainer(context.Background(), "IMAGE", "not-a-url", "")
	if err == nil {
		t.Fatal("expected error for invalid media URL format")
	}
}

func TestCreateMediaContainer_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.CreateMediaContainer(context.Background(), "IMAGE", "https://example.com/img.jpg", "")
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestWaitForContainerReady_Finished(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"c1","status":"FINISHED"}`))

	err := client.waitForContainerReady(context.Background(), ConvertToContainerID("c1"), 3, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitForContainerReady_Error(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"c1","status":"ERROR","error_message":"upload failed"}`))

	err := client.waitForContainerReady(context.Background(), ConvertToContainerID("c1"), 3, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for container error status")
	}
	if !strings.Contains(err.Error(), "upload failed") {
		t.Errorf("expected error message about upload failure, got: %v", err)
	}
}

func TestWaitForContainerReady_ErrorNoMessage(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"c1","status":"ERROR"}`))

	err := client.waitForContainerReady(context.Background(), ConvertToContainerID("c1"), 3, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for container error status")
	}
	if !strings.Contains(err.Error(), "container processing failed with error status") {
		t.Errorf("expected generic error message, got: %v", err)
	}
}

func TestWaitForContainerReady_Expired(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"c1","status":"EXPIRED"}`))

	err := client.waitForContainerReady(context.Background(), ConvertToContainerID("c1"), 3, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for expired container")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expired error, got: %v", err)
	}
}

func TestWaitForContainerReady_Timeout(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"c1","status":"IN_PROGRESS"}`))

	err := client.waitForContainerReady(context.Background(), ConvertToContainerID("c1"), 2, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestWaitForContainerReady_ContextCancelled(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":"c1","status":"IN_PROGRESS"}`))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := client.waitForContainerReady(ctx, ConvertToContainerID("c1"), 10, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestWaitForContainerReady_ProgressThenFinished(t *testing.T) {
	var callCount int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		if count <= 2 {
			_, _ = w.Write([]byte(`{"id":"c1","status":"IN_PROGRESS"}`))
		} else {
			_, _ = w.Write([]byte(`{"id":"c1","status":"FINISHED"}`))
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	err := client.waitForContainerReady(context.Background(), ConvertToContainerID("c1"), 5, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateImagePost_ContainerCreateError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
		} else {
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateImagePost(context.Background(), &ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
	})
	if err == nil {
		t.Fatal("expected error when container creation fails")
	}
	if !strings.Contains(err.Error(), "failed to create image container") {
		t.Errorf("expected image container error, got: %v", err)
	}
}

func TestCreateImagePost_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.CreateImagePost(context.Background(), &ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestCreateVideoPost_ContainerCreateError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
		} else {
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateVideoPost(context.Background(), &VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
	})
	if err == nil {
		t.Fatal("expected error when container creation fails")
	}
	if !strings.Contains(err.Error(), "failed to create video container") {
		t.Errorf("expected video container error, got: %v", err)
	}
}

func TestCreateVideoPost_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.CreateVideoPost(context.Background(), &VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestRepostPost_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.RepostPost(context.Background(), ConvertToPostID("original_post"))
	if err == nil {
		t.Fatal("expected error for API error")
	}
}

func TestRepostPost_InvalidResponse(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":""}`))

	_, err := client.RepostPost(context.Background(), ConvertToPostID("original_post"))
	if err == nil {
		t.Fatal("expected error for empty repost ID in response")
	}
}

func TestRepostPost_MalformedResponse(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))

	_, err := client.RepostPost(context.Background(), ConvertToPostID("original_post"))
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestRepostPost_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.RepostPost(context.Background(), ConvertToPostID("original_post"))
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestCreateTextPost_ContainerCreateError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
		} else {
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text: "Hello",
	})
	if err == nil {
		t.Fatal("expected error when container creation fails")
	}
	if !strings.Contains(err.Error(), "failed to create text container") {
		t.Errorf("expected text container error, got: %v", err)
	}
}

func TestCreateTextPost_NotAuthenticated(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	_ = client.ClearToken()

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text: "Hello",
	})
	if err == nil {
		t.Fatal("expected error when not authenticated")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestCreateTextPost_PublishError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads_publish"):
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"publish failed","type":"OAuthException","code":100}}`))
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"container_1"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/container_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"container_1","status":"FINISHED"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text: "Hello",
	})
	if err == nil {
		t.Fatal("expected error when publish fails")
	}
	if !strings.Contains(err.Error(), "failed to publish text post") {
		t.Errorf("expected publish error, got: %v", err)
	}
}

func TestCreateTextPost_WaitForContainerError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/12345/threads"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"container_1"}`))
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/container_1"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":"container_1","status":"ERROR","error_message":"failed"}`))
		default:
			http.NotFound(w, r)
		}
	}

	client := testClient(t, http.HandlerFunc(handler))

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text: "Hello",
	})
	if err == nil {
		t.Fatal("expected error when container not ready")
	}
	if !strings.Contains(err.Error(), "container not ready for publishing") {
		t.Errorf("expected container not ready error, got: %v", err)
	}
}

func TestCreateAndPublishTextPostDirectly_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text:            "Hello",
		AutoPublishText: true,
	})
	if err == nil {
		t.Fatal("expected error for API error")
	}
}

func TestCreateAndPublishTextPostDirectly_EmptyPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":""}`))

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text:            "Hello",
		AutoPublishText: true,
	})
	if err == nil {
		t.Fatal("expected error for empty post ID in response")
	}
}

func TestCreateAndPublishTextPostDirectly_MalformedResponse(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))

	_, err := client.CreateTextPost(context.Background(), &TextPostContent{
		Text:            "Hello",
		AutoPublishText: true,
	})
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestCreateContainer_EmptyUserID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	// Clear token info to have empty user ID
	client.mu.Lock()
	client.tokenInfo.UserID = ""
	client.mu.Unlock()

	_, err := client.createContainer(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestCreateContainer_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.createContainer(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for API error")
	}
}

func TestCreateContainer_MalformedResponse(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))

	_, err := client.createContainer(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestCreateContainer_EmptyContainerID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":""}`))

	_, err := client.createContainer(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty container ID in response")
	}
}

func TestPublishContainer_EmptyContainerID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	_, err := client.publishContainer(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty container ID")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestPublishContainer_EmptyUserID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))
	client.mu.Lock()
	client.tokenInfo.UserID = ""
	client.mu.Unlock()

	_, err := client.publishContainer(context.Background(), "container_1")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
	if !IsAuthenticationError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestPublishContainer_APIError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"message":"Bad request","type":"OAuthException","code":100}}`))
	}

	client := testClient(t, http.HandlerFunc(handler))
	client.config.RetryConfig.MaxRetries = 0

	_, err := client.publishContainer(context.Background(), "container_1")
	if err == nil {
		t.Fatal("expected error for API error")
	}
}

func TestPublishContainer_MalformedResponse(t *testing.T) {
	client := testClient(t, jsonHandler(200, `not json`))

	_, err := client.publishContainer(context.Background(), "container_1")
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
}

func TestPublishContainer_EmptyPostID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{"id":""}`))

	_, err := client.publishContainer(context.Background(), "container_1")
	if err == nil {
		t.Fatal("expected error for empty post ID in response")
	}
}
