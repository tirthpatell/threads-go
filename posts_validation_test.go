package threads

import (
	"strings"
	"testing"
)

func TestValidateTextPostContent_Nil(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateTextPostContent_Valid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Hello world",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_TooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longText := strings.Repeat("a", MaxTextLength+1)
	err := client.ValidateTextPostContent(&TextPostContent{
		Text: longText,
	})
	if err == nil {
		t.Fatal("expected error for text too long")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateTextPostContent_TooManyLinks(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	// 6 unique links exceeds the limit of 5
	text := "https://a.com https://b.com https://c.com https://d.com https://e.com"
	err := client.ValidateTextPostContent(&TextPostContent{
		Text:           text,
		LinkAttachment: "https://f.com",
	})
	if err == nil {
		t.Fatal("expected error for too many links")
	}
}

func TestValidateTextPostContent_WithTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:     "Hello",
		TopicTag: "golang",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_InvalidTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:     "Hello",
		TopicTag: "go.lang",
	})
	if err == nil {
		t.Fatal("expected error for topic tag with period")
	}
}

func TestValidateTextPostContent_WithCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:                    "Hello",
		AllowlistedCountryCodes: []string{"US", "CA"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_InvalidCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:                    "Hello",
		AllowlistedCountryCodes: []string{"USA"},
	})
	if err == nil {
		t.Fatal("expected error for invalid country code")
	}
}

func TestValidateTextPostContent_GhostPostAsReply(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:        "Ghost",
		IsGhostPost: true,
		ReplyTo:     "some_post",
	})
	if err == nil {
		t.Fatal("expected error for ghost post as reply")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateTextPostContent_GhostPostWithReplyApprovals(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:                 "Ghost",
		IsGhostPost:          true,
		EnableReplyApprovals: true,
	})
	if err == nil {
		t.Fatal("expected error for ghost post with reply approvals")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateTextPostContent_TextAttachmentWithPoll(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Hello",
		TextAttachment: &TextAttachment{
			Plaintext: "Some long text",
		},
		PollAttachment: &PollAttachment{
			OptionA: "Yes",
			OptionB: "No",
		},
	})
	if err == nil {
		t.Fatal("expected error for text attachment with poll")
	}
}

func TestValidateTextPostContent_DuplicateLinkAttachments(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:           "Hello",
		LinkAttachment: "https://example.com",
		TextAttachment: &TextAttachment{
			Plaintext:         "Some long text",
			LinkAttachmentURL: "https://other.com",
		},
	})
	if err == nil {
		t.Fatal("expected error for duplicate link attachments")
	}
}

func TestValidateTextPostContent_WithTextEntities(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Hello world spoiler here",
		TextEntities: []TextEntity{
			{EntityType: "SPOILER", Offset: 12, Length: 7},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_TooManyTextEntities(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	entities := make([]TextEntity, MaxTextEntities+1)
	for i := range entities {
		entities[i] = TextEntity{EntityType: "SPOILER", Offset: i * 5, Length: 3}
	}

	err := client.ValidateTextPostContent(&TextPostContent{
		Text:         strings.Repeat("hello ", MaxTextEntities+2),
		TextEntities: entities,
	})
	if err == nil {
		t.Fatal("expected error for too many text entities")
	}
}

func TestValidateTextPostContent_ValidGIFAttachment(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Check this GIF",
		GIFAttachment: &GIFAttachment{
			GIFID:    "abc123",
			Provider: GIFProviderGiphy,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_InvalidGIFAttachment(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Check this GIF",
		GIFAttachment: &GIFAttachment{
			GIFID:    "",
			Provider: GIFProviderGiphy,
		},
	})
	if err == nil {
		t.Fatal("expected error for empty GIF ID")
	}
}

func TestValidateImagePostContent_Nil(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
}

func TestValidateImagePostContent_Valid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		Text:     "My image",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateImagePostContent_TextTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longText := strings.Repeat("a", MaxTextLength+1)
	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		Text:     longText,
	})
	if err == nil {
		t.Fatal("expected error for text too long")
	}
}

func TestValidateImagePostContent_InvalidURL(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "not-a-url",
	})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestValidateImagePostContent_WithTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		TopicTag: "photography",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateImagePostContent_InvalidTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		TopicTag: "photo.graphy",
	})
	if err == nil {
		t.Fatal("expected error for invalid topic tag")
	}
}

func TestValidateImagePostContent_WithCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL:                "https://example.com/img.jpg",
		AllowlistedCountryCodes: []string{"US", "GB"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateImagePostContent_InvalidCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL:                "https://example.com/img.jpg",
		AllowlistedCountryCodes: []string{"TOOLONG"},
	})
	if err == nil {
		t.Fatal("expected error for invalid country code")
	}
}

func TestValidateVideoPostContent_Nil(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
}

func TestValidateVideoPostContent_Valid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVideoPostContent_TextTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longText := strings.Repeat("a", MaxTextLength+1)
	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
		Text:     longText,
	})
	if err == nil {
		t.Fatal("expected error for text too long")
	}
}

func TestValidateVideoPostContent_InvalidURL(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "ftp://invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestValidateVideoPostContent_WithTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
		TopicTag: "video",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVideoPostContent_InvalidTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
		TopicTag: "vid&eo",
	})
	if err == nil {
		t.Fatal("expected error for topic tag with ampersand")
	}
}

func TestValidateVideoPostContent_WithCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL:                "https://example.com/vid.mp4",
		AllowlistedCountryCodes: []string{"US"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVideoPostContent_InvalidCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL:                "https://example.com/vid.mp4",
		AllowlistedCountryCodes: []string{"1A"},
	})
	if err == nil {
		t.Fatal("expected error for invalid country code")
	}
}

func TestValidateCarouselPostContent_Nil(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(nil)
	if err == nil {
		t.Fatal("expected error for nil content")
	}
}

func TestValidateCarouselPostContent_Valid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Text:     "Carousel",
		Children: []string{"child_1", "child_2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCarouselPostContent_TooFewChildren(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Text:     "Carousel",
		Children: []string{"child_1"},
	})
	if err == nil {
		t.Fatal("expected error for too few children")
	}
}

func TestValidateCarouselPostContent_TooManyChildren(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	children := make([]string, MaxCarouselItems+1)
	for i := range children {
		children[i] = "child"
	}

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Text:     "Carousel",
		Children: children,
	})
	if err == nil {
		t.Fatal("expected error for too many children")
	}
}

func TestValidateCarouselPostContent_TextTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longText := strings.Repeat("a", MaxTextLength+1)
	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Text:     longText,
		Children: []string{"child_1", "child_2"},
	})
	if err == nil {
		t.Fatal("expected error for text too long")
	}
}

func TestValidateCarouselPostContent_WithTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Children: []string{"child_1", "child_2"},
		TopicTag: "carousel",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCarouselPostContent_InvalidTopicTag(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Children: []string{"child_1", "child_2"},
		TopicTag: "carou.sel",
	})
	if err == nil {
		t.Fatal("expected error for invalid topic tag")
	}
}

func TestValidateCarouselPostContent_WithCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Children:                []string{"child_1", "child_2"},
		AllowlistedCountryCodes: []string{"US", "CA"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCarouselPostContent_InvalidCountryCodes(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Children:                []string{"child_1", "child_2"},
		AllowlistedCountryCodes: []string{"XYZ"},
	})
	if err == nil {
		t.Fatal("expected error for invalid country code")
	}
}

func TestValidateCarouselChildren_Empty(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselChildren([]string{})
	if err == nil {
		t.Fatal("expected error for empty children")
	}
}

func TestValidateCarouselChildren_Valid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselChildren([]string{"child_1", "child_2", "child_3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCarouselChildren_TooFew(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselChildren([]string{"child_1"})
	if err == nil {
		t.Fatal("expected error for too few children")
	}
}

func TestValidateCarouselChildren_TooMany(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	children := make([]string, MaxCarouselItems+1)
	for i := range children {
		children[i] = "child"
	}

	err := client.ValidateCarouselChildren(children)
	if err == nil {
		t.Fatal("expected error for too many children")
	}
}

func TestValidateCarouselChildren_EmptyChildID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselChildren([]string{"child_1", "", "child_3"})
	if err == nil {
		t.Fatal("expected error for empty child ID")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateCarouselChildren_WhitespaceChildID(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselChildren([]string{"child_1", "   ", "child_3"})
	if err == nil {
		t.Fatal("expected error for whitespace-only child ID")
	}
}

func TestValidateTopicTag_Valid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTopicTag("golang")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTopicTag_Empty(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTopicTag("")
	if err != nil {
		t.Fatalf("unexpected error for empty tag: %v", err)
	}
}

func TestValidateTopicTag_WithPeriod(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTopicTag("go.lang")
	if err == nil {
		t.Fatal("expected error for topic tag with period")
	}
}

func TestValidateTopicTag_WithAmpersand(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTopicTag("go&lang")
	if err == nil {
		t.Fatal("expected error for topic tag with ampersand")
	}
}

func TestValidateCountryCodes_Valid(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCountryCodes([]string{"US", "CA", "GB"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCountryCodes_Empty(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCountryCodes([]string{})
	if err != nil {
		t.Fatalf("unexpected error for empty codes: %v", err)
	}
}

func TestValidateCountryCodes_InvalidLength(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCountryCodes([]string{"USA"})
	if err == nil {
		t.Fatal("expected error for 3-character code")
	}
}

func TestValidateCountryCodes_NonAlphabetic(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCountryCodes([]string{"1A"})
	if err == nil {
		t.Fatal("expected error for non-alphabetic code")
	}
}

func TestValidateTextPostContent_WithTextAttachment(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Hello",
		TextAttachment: &TextAttachment{
			Plaintext: "Some extended text content here.",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_TextAttachmentTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longPlaintext := strings.Repeat("a", MaxTextAttachmentLength+1)
	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Hello",
		TextAttachment: &TextAttachment{
			Plaintext: longPlaintext,
		},
	})
	if err == nil {
		t.Fatal("expected error for text attachment too long")
	}
}

func TestValidateImagePostContent_WithTextEntities(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		Text:     "Hello spoiler here",
		TextEntities: []TextEntity{
			{EntityType: "SPOILER", Offset: 6, Length: 7},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVideoPostContent_WithTextEntities(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
		Text:     "Hello spoiler here",
		TextEntities: []TextEntity{
			{EntityType: "SPOILER", Offset: 6, Length: 7},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCarouselPostContent_WithTextEntities(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Children: []string{"child_1", "child_2"},
		Text:     "Hello spoiler here",
		TextEntities: []TextEntity{
			{EntityType: "SPOILER", Offset: 6, Length: 7},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_WithValidPoll(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Which do you prefer?",
		PollAttachment: &PollAttachment{
			OptionA: "Option A",
			OptionB: "Option B",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTextPostContent_PollMissingOptionA(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Which do you prefer?",
		PollAttachment: &PollAttachment{
			OptionA: "",
			OptionB: "Option B",
		},
	})
	if err == nil {
		t.Fatal("expected error for poll missing option A")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateTextPostContent_PollOptionTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longOption := strings.Repeat("a", MaxPollOptionLength+1)
	err := client.ValidateTextPostContent(&TextPostContent{
		Text: "Which do you prefer?",
		PollAttachment: &PollAttachment{
			OptionA: longOption,
			OptionB: "Option B",
		},
	})
	if err == nil {
		t.Fatal("expected error for poll option too long")
	}
}

func TestValidateImagePostContent_WithValidAltText(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		AltText:  "A beautiful photo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateImagePostContent_AltTextTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longAltText := strings.Repeat("a", MaxAltTextLength+1)
	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		AltText:  longAltText,
	})
	if err == nil {
		t.Fatal("expected error for alt text too long")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestValidateVideoPostContent_WithValidAltText(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
		AltText:  "A short video clip",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVideoPostContent_AltTextTooLong(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	longAltText := strings.Repeat("a", MaxAltTextLength+1)
	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
		AltText:  longAltText,
	})
	if err == nil {
		t.Fatal("expected error for alt text too long")
	}
}

func TestValidateImagePostContent_TooManyLinks(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	text := "https://a.com https://b.com https://c.com https://d.com https://e.com https://f.com"
	err := client.ValidateImagePostContent(&ImagePostContent{
		ImageURL: "https://example.com/img.jpg",
		Text:     text,
	})
	if err == nil {
		t.Fatal("expected error for too many links")
	}
}

func TestValidateVideoPostContent_TooManyLinks(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	text := "https://a.com https://b.com https://c.com https://d.com https://e.com https://f.com"
	err := client.ValidateVideoPostContent(&VideoPostContent{
		VideoURL: "https://example.com/vid.mp4",
		Text:     text,
	})
	if err == nil {
		t.Fatal("expected error for too many links")
	}
}

func TestValidateCarouselPostContent_TooManyLinks(t *testing.T) {
	client := testClient(t, jsonHandler(200, `{}`))

	text := "https://a.com https://b.com https://c.com https://d.com https://e.com https://f.com"
	err := client.ValidateCarouselPostContent(&CarouselPostContent{
		Children: []string{"child_1", "child_2"},
		Text:     text,
	})
	if err == nil {
		t.Fatal("expected error for too many links")
	}
}
