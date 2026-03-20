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

func TestContainerBuilder_SetIsGhostPost_NoMediaType_Allowed(t *testing.T) {
	// When media_type is not yet set, ghost post flag should be allowed
	b := NewContainerBuilder().
		SetIsGhostPost(true)

	params := b.Build()
	if params.Get("is_ghost_post") != "true" {
		t.Error("expected is_ghost_post to be set when no media_type is specified")
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
