package threads

import "testing"

func TestContainerBuilder_SetIsGhostPost_TextMediaType(t *testing.T) {
	b := NewContainerBuilder().
		SetMediaType(MediaTypeText).
		SetIsGhostPost(true)

	params := b.Build()
	if params.Get("is_ghost_post") != "true" {
		t.Error("expected is_ghost_post to be set for TEXT media type")
	}
}

func TestContainerBuilder_SetIsGhostPost_ImageMediaType_Ignored(t *testing.T) {
	b := NewContainerBuilder().
		SetMediaType(MediaTypeImage).
		SetIsGhostPost(true)

	params := b.Build()
	if params.Get("is_ghost_post") != "" {
		t.Error("expected is_ghost_post to be ignored for IMAGE media type")
	}
}

func TestContainerBuilder_SetIsGhostPost_VideoMediaType_Ignored(t *testing.T) {
	b := NewContainerBuilder().
		SetMediaType(MediaTypeVideo).
		SetIsGhostPost(true)

	params := b.Build()
	if params.Get("is_ghost_post") != "" {
		t.Error("expected is_ghost_post to be ignored for VIDEO media type")
	}
}

func TestContainerBuilder_SetIsGhostPost_CarouselMediaType_Ignored(t *testing.T) {
	b := NewContainerBuilder().
		SetMediaType(MediaTypeCarousel).
		SetIsGhostPost(true)

	params := b.Build()
	if params.Get("is_ghost_post") != "" {
		t.Error("expected is_ghost_post to be ignored for CAROUSEL media type")
	}
}

func TestContainerBuilder_SetIsGhostPost_NoMediaType_ThenText(t *testing.T) {
	// Ghost post set before media type; TEXT should preserve it
	b := NewContainerBuilder().
		SetIsGhostPost(true).
		SetMediaType(MediaTypeText)

	params := b.Build()
	if params.Get("is_ghost_post") != "true" {
		t.Error("expected is_ghost_post to be preserved when media_type is set to TEXT after")
	}
}

func TestContainerBuilder_SetIsGhostPost_NoMediaType_ThenImage(t *testing.T) {
	// Ghost post set before media type; IMAGE should clear it
	b := NewContainerBuilder().
		SetIsGhostPost(true).
		SetMediaType(MediaTypeImage)

	params := b.Build()
	if params.Get("is_ghost_post") != "" {
		t.Error("expected is_ghost_post to be cleared when media_type is set to IMAGE after")
	}
}

func TestContainerBuilder_SetIsGhostPost_ToggleTrueToFalse(t *testing.T) {
	// Setting true then false should clear the flag
	b := NewContainerBuilder().
		SetMediaType(MediaTypeText).
		SetIsGhostPost(true).
		SetIsGhostPost(false)

	params := b.Build()
	if params.Get("is_ghost_post") != "" {
		t.Error("expected is_ghost_post to be cleared after SetIsGhostPost(false)")
	}
}

func TestContainerBuilder_SetIsGhostPost_False_NotSet(t *testing.T) {
	b := NewContainerBuilder().
		SetMediaType(MediaTypeText).
		SetIsGhostPost(false)

	params := b.Build()
	if params.Get("is_ghost_post") != "" {
		t.Error("expected is_ghost_post to not be set when isGhostPost=false")
	}
}
