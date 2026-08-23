package threads

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// publishRecoveryWindow is the negative buffer applied to the publishStart
// timestamp when querying recent posts during recovery. It absorbs clock skew
// between the local machine and Meta's servers without widening the search
// enough to risk collisions with prior unrelated publishes.
const publishRecoveryWindow = 5 * time.Second

// Recovery polling bounds. Both gates (container status flip to PUBLISHED
// and the /me/threads index seeing our new post) can race with Meta
// returning the code-10 error response, so do a few brief polls before
// concluding the publish really failed. Total worst-case latency is
// (maxStatusPolls + maxListPolls - 2) * pollInterval, kept small enough
// to not dominate caller-visible latency.
//
// Declared as vars (not consts) so tests in this package can shrink the
// interval without paying the production wait time.
var (
	maxRecoveryStatusPolls = 5
	maxRecoveryListPolls   = 3
	recoveryPollInterval   = 1 * time.Second
)

// errPublishNotRecovered is the sentinel returned by tryRecoverPublishedPost
// when no recovery is possible. Callers should surface the original publish
// error in this case, not the recovery error.
var errPublishNotRecovered = errors.New("publish not recovered")

// publishMatcher reports whether a post returned from /me/threads is the
// post that the current publish attempt produced. Matchers are content-type
// specific (carousel matches by text + topic_tag + children count + reply/
// quote state; image-reply by parent ID + text, etc.); see makeXxxMatcher
// helpers below.
type publishMatcher func(*Post) bool

// shouldAttemptPublishRecovery reports whether err matches the documented
// Meta pattern where /threads_publish returns an error response despite the
// container actually being published. Currently this is code 10
// (GraphMethodException — "Application does not have permission for this
// action"), which Meta returns from /threads_publish for some app/permission
// configurations after-the-fact, with the container moving to PUBLISHED
// regardless. See the Threads API troubleshooting docs (section "Publishing
// Does Not Return a Media ID") for the canonical recovery strategy.
func shouldAttemptPublishRecovery(err error) bool {
	base := extractBaseError(err)
	if base == nil {
		return false
	}
	return isNonRetryablePermanentErrorCode(base.Code)
}

// recoverFromPublishError is the single entry point Create*Post functions
// call after a failed publishContainer. It short-circuits to errPublishNotRecovered
// when the publish error is not one we know to be ambiguous (i.e. not
// code 10), avoiding an extra Meta round-trip for genuine failures.
func (c *Client) recoverFromPublishError(
	ctx context.Context,
	containerID string,
	publishStart time.Time,
	publishErr error,
	match publishMatcher,
) (*Post, error) {
	if !shouldAttemptPublishRecovery(publishErr) {
		return nil, errPublishNotRecovered
	}
	return c.tryRecoverPublishedPost(ctx, containerID, publishStart, match)
}

// recoverOrWrap is the canonical post-publish-error glue for Create*Post:
// it runs recovery, then turns the (recovered, recErr, publishErr) triple
// into a single caller-facing return.
//
// Three outcomes:
//
//  1. Recovery succeeded → return the recovered post.
//  2. Recovery was cut short by caller-context cancellation or deadline →
//     propagate the ctx error verbatim. Callers (and any downstream
//     classification, e.g. retry loops) rely on errors.Is(err, context.Xxx)
//     to distinguish "the publish actually failed" from "we ran out of time
//     during recovery"; swallowing the ctx error into the original publish
//     error breaks that distinction.
//  3. Recovery couldn't help (errPublishNotRecovered, a recovery-side HTTP
//     error that isn't a context error, etc.) → surface the ORIGINAL
//     publish error wrapped by wrapFmt. This is the case where the publish
//     itself is what the caller needs to see.
//
// wrapFmt is a fmt.Errorf format string with a single %w verb, e.g.
// "failed to publish carousel post: %w".
func (c *Client) recoverOrWrap(
	ctx context.Context,
	containerID string,
	publishStart time.Time,
	publishErr error,
	match publishMatcher,
	wrapFmt string,
) (*Post, error) {
	recovered, recErr := c.recoverFromPublishError(ctx, containerID, publishStart, publishErr, match)
	if recErr == nil {
		return recovered, nil
	}
	if errors.Is(recErr, context.Canceled) || errors.Is(recErr, context.DeadlineExceeded) {
		return nil, recErr
	}
	return nil, fmt.Errorf(wrapFmt, publishErr)
}

// tryRecoverPublishedPost implements the Meta-documented recovery flow for
// /threads_publish calls that error out despite succeeding server-side:
//
//  1. Poll the container's status briefly until it flips to PUBLISHED.
//     The publish response can arrive before the container's status row
//     has been updated, so a single read can spuriously see FINISHED and
//     mistake a successful publish for a real failure.
//  2. List recent posts authored by this user since publishStart (minus a
//     small skew buffer) and apply the caller-supplied matcher. The post
//     can lag the container flip in the /me/threads index, so retry the
//     list lookup briefly before giving up.
//  3. Return the unique matching post, or errPublishNotRecovered if zero
//     or more than one posts match across all attempts.
//
// The matcher MUST be content-specific enough to be unique per request even
// under concurrent publishes from the same user. See each makeXxxMatcher
// helper for the per-type guarantees.
//
// Returns errPublishNotRecovered (not the original publish error) when
// recovery isn't possible — the caller is expected to propagate the
// original error in that case.
func (c *Client) tryRecoverPublishedPost(
	ctx context.Context,
	containerID string,
	publishStart time.Time,
	match publishMatcher,
) (*Post, error) {
	// Gate 1: poll until the container reports PUBLISHED. Terminal failure
	// states (ERROR, EXPIRED) short-circuit to "not recovered" — the
	// publish really didn't succeed.
	if err := c.waitForContainerPublished(ctx, ContainerID(containerID)); err != nil {
		return nil, err
	}

	// Gate 2: poll /me/threads for a matching post. The carousel just
	// flipped to PUBLISHED but indexing may lag slightly; retry briefly.
	userID := c.getUserID()
	if !UserID(userID).Valid() {
		return nil, NewAuthenticationError(401, "Invalid user ID", "Token contains an invalid user ID")
	}
	sinceTS := publishStart.Add(-publishRecoveryWindow).Unix()

	for attempt := 0; attempt < maxRecoveryListPolls; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, recoveryPollInterval); err != nil {
				return nil, err
			}
		}

		posts, err := c.GetUserPostsWithOptions(ctx, UserID(userID), &PostsOptions{
			Limit: 25,
			Since: sinceTS,
		})
		if err != nil {
			return nil, fmt.Errorf("recovery: failed to list recent posts: %w", err)
		}

		// Find a unique match. Fail closed on multiple matches — matchers
		// are designed to be unique per request, so >1 match means we'd
		// be guessing; better to surface the original publish error.
		var found *Post
		ambiguous := false
		for i := range posts.Data {
			if match(&posts.Data[i]) {
				if found != nil {
					ambiguous = true
					break
				}
				found = &posts.Data[i]
			}
		}
		if ambiguous {
			return nil, errPublishNotRecovered
		}
		if found != nil {
			return found, nil
		}
		// Zero matches yet — fall through to next poll.
	}

	return nil, errPublishNotRecovered
}

// waitForContainerPublished polls the container's status endpoint until it
// reports PUBLISHED, returns a terminal-failure sentinel, or the poll budget
// is exhausted. Returns nil only when PUBLISHED is observed.
func (c *Client) waitForContainerPublished(ctx context.Context, containerID ContainerID) error {
	for attempt := 0; attempt < maxRecoveryStatusPolls; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, recoveryPollInterval); err != nil {
				return err
			}
		}

		status, err := c.GetContainerStatus(ctx, containerID)
		if err != nil {
			return fmt.Errorf("recovery: failed to fetch container status: %w", err)
		}
		switch status.Status {
		case ContainerStatusPublished:
			return nil
		case ContainerStatusError, ContainerStatusExpired:
			// Terminal — the publish really did fail. No point polling.
			return errPublishNotRecovered
		}
		// FINISHED or IN_PROGRESS — keep polling.
	}
	return errPublishNotRecovered
}

// sleepCtx waits for d or returns ctx's error if it's cancelled first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// makeCarouselMatcher returns a matcher for a published carousel post.
//
// Note on why we don't match by child IDs: Meta's create endpoint takes
// MEDIA CONTAINER IDs in the `children` parameter, but the read endpoint
// (/me/threads with `children` field) returns CHILD POST IDs of the
// individual published children — these are different IDs. Verified in
// production: a carousel published with container IDs [C1..Cn] reads
// back with children [P1..Pn] where each Pi is its own /post/{shortcode}
// permalink. So matching by ID-set never works after publish.
//
// Instead we match by content: text + topic_tag + reply/quote state +
// children count. Same uniqueness rules as image-root: a unique
// discriminator (non-empty text, topic_tag, or quoted_post_id) is
// required for non-reply carousels, and text-or-quote is required for
// reply carousels. Caller-side carousels with blank text and no quote
// can't be safely recovered — but the bot-style "chunked reply chain"
// flow that hits this case is already loss-tolerant by design.
func makeCarouselMatcher(content *CarouselPostContent) publishMatcher {
	want := len(content.Children)
	return func(p *Post) bool {
		if p.MediaType != MediaTypeResponseCarousel {
			return false
		}
		if p.Children == nil || len(p.Children.Data) != want {
			return false
		}
		if !quoteMatches(p, content.QuotedPostID) {
			return false
		}
		if content.ReplyTo == "" {
			if !rootHasUniqueDiscriminator(content.Text, content.TopicTag, content.QuotedPostID) {
				return false
			}
			return !p.IsReply && p.Text == content.Text && p.TopicTag == content.TopicTag
		}
		if !p.IsReply || repliedToID(p) != content.ReplyTo {
			return false
		}
		if !replyHasUniqueDiscriminator(content.Text, content.QuotedPostID) {
			return false
		}
		return p.Text == content.Text
	}
}

// makeImageMatcher returns a matcher for a single-image post.
//
// For non-replies, exact text + topic_tag + non-reply state is the match
// key; we additionally require at least one of (text, topic_tag,
// quoted_post_id) to be non-empty, because an image-only root with no
// text and no tag has no unique signal — matching by media_type +
// !is_reply alone would let any prior unrelated image post in the window
// pass.
//
// For replies, parent ID is necessary but not sufficient (multiple
// replies to the same parent are valid). Require non-empty text or a
// matching quoted-post target; blank-text non-quote replies fail closed.
//
// The image URL stored on the post is Meta's CDN URL, not ours, so we
// don't compare it.
func makeImageMatcher(content *ImagePostContent) publishMatcher {
	return func(p *Post) bool {
		if p.MediaType != MediaTypeImage {
			return false
		}
		if !quoteMatches(p, content.QuotedPostID) {
			return false
		}
		if content.ReplyTo == "" {
			if !rootHasUniqueDiscriminator(content.Text, content.TopicTag, content.QuotedPostID) {
				return false
			}
			return !p.IsReply && p.Text == content.Text && p.TopicTag == content.TopicTag
		}
		if !p.IsReply || repliedToID(p) != content.ReplyTo {
			return false
		}
		if !replyHasUniqueDiscriminator(content.Text, content.QuotedPostID) {
			return false
		}
		return p.Text == content.Text
	}
}

// makeVideoMatcher mirrors makeImageMatcher for video posts. After publish,
// Meta may report video posts as media_type == "VIDEO" or "AUDIO" (for
// audio-only uploads). Accept either. Same uniqueness rules: blank-text
// non-quote replies fail closed; non-reply posts require a non-empty
// discriminator (text, topic_tag, or quoted_post_id).
func makeVideoMatcher(content *VideoPostContent) publishMatcher {
	return func(p *Post) bool {
		if p.MediaType != MediaTypeVideo && p.MediaType != MediaTypeAudio {
			return false
		}
		if !quoteMatches(p, content.QuotedPostID) {
			return false
		}
		if content.ReplyTo == "" {
			if !rootHasUniqueDiscriminator(content.Text, content.TopicTag, content.QuotedPostID) {
				return false
			}
			return !p.IsReply && p.Text == content.Text && p.TopicTag == content.TopicTag
		}
		if !p.IsReply || repliedToID(p) != content.ReplyTo {
			return false
		}
		if !replyHasUniqueDiscriminator(content.Text, content.QuotedPostID) {
			return false
		}
		return p.Text == content.Text
	}
}

// makeTextMatcher returns a matcher for a text-only post. Text posts always
// have non-empty text (validated upstream), so text equality is itself a
// strong discriminator. Quote state must match in both directions, and
// non-replies additionally compare topic_tag to disambiguate same-text
// posts across different topics.
func makeTextMatcher(content *TextPostContent) publishMatcher {
	wantTextType := MediaTypeResponseText
	return func(p *Post) bool {
		// Some text post variants come back as TEXT_POST or omit media_type
		// entirely. Accept both shapes.
		if p.MediaType != "" && p.MediaType != wantTextType && p.MediaType != MediaTypeText {
			return false
		}
		if !quoteMatches(p, content.QuotedPostID) {
			return false
		}
		if p.Text != content.Text {
			return false
		}
		if content.ReplyTo != "" {
			return p.IsReply && repliedToID(p) == content.ReplyTo
		}
		return !p.IsReply && p.TopicTag == content.TopicTag
	}
}

// quoteMatches reports whether a retrieved post's quote state aligns with
// what the caller asked for. wantQuotedID == "" means "must not be a quote
// post"; non-empty means "must be a quote of that specific post". Matching
// across the quote/non-quote boundary would let a regular post masquerade
// as a quote post (or vice-versa) during recovery.
func quoteMatches(p *Post, wantQuotedID string) bool {
	if wantQuotedID == "" {
		return !p.IsQuotePost
	}
	if !p.IsQuotePost || p.QuotedPost == nil {
		return false
	}
	return p.QuotedPost.ID == wantQuotedID
}

// replyHasUniqueDiscriminator reports whether the (text, quotedPostID) pair
// is strong enough to single out one reply among potentially many to the
// same parent. Without one of these signals we'd be guessing.
func replyHasUniqueDiscriminator(text, quotedPostID string) bool {
	return text != "" || quotedPostID != ""
}

// rootHasUniqueDiscriminator reports whether a non-reply post has any
// content signal that distinguishes it from prior unrelated posts in the
// recovery window. An image-only or video-only root with empty text and
// empty topic_tag has no such signal — matching by media_type + !is_reply
// alone would accept any prior post of the same type. Such recoveries
// fail closed.
func rootHasUniqueDiscriminator(text, topicTag, quotedPostID string) bool {
	return text != "" || topicTag != "" || quotedPostID != ""
}

// repliedToID returns the parent post ID for a reply, preferring the
// dedicated reply_to string field and falling back to the embedded
// replied_to object. Returns "" if neither is populated.
func repliedToID(p *Post) string {
	if p.ReplyTo != "" {
		return p.ReplyTo
	}
	if p.RepliedTo != nil {
		return p.RepliedTo.ID
	}
	return ""
}
