//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	threads "github.com/tirthpatell/threads-go"
)

// testPostMarker is appended to the text of every post these tests publish so
// that leftovers can be identified and removed later. It must be distinctive
// enough that no real post on the account contains it.
const testPostMarker = "[threads-go-integration]"

// sweepLookback bounds how far back the leftover sweep looks. Test posts are
// only ever minutes old; anything older is not ours to touch.
const sweepLookback = 24 * time.Hour

// markTestPost appends the marker that makes a published post identifiable as
// test residue. The marker is appended rather than prefixed so that text-entity
// offsets in the original text stay valid.
func markTestPost(text string) string {
	return text + " " + testPostMarker
}

// trackPost registers deletion of a published test post via t.Cleanup, so the
// post is removed even when the test fails, returns early, or calls t.Fatal.
// Deleting an already-deleted post is not treated as a failure.
func trackPost(t *testing.T, client *threads.Client, postID string) {
	t.Helper()
	if postID == "" {
		return
	}
	t.Cleanup(func() {
		// The API needs a moment after publish before a delete is accepted.
		time.Sleep(1 * time.Second)
		if _, err := client.DeletePost(context.Background(), threads.ConvertToPostID(postID)); err != nil {
			t.Logf("cleanup: failed to delete test post %s: %v (sweep will retry on the next run)", postID, err)
			return
		}
		t.Logf("cleanup: deleted test post %s", postID)
	})
}

// sweepLeftoverTestPosts deletes any post on the account still carrying
// testPostMarker. t.Cleanup cannot help when a run is killed outright — a
// cancelled CI job, a panic, SIGKILL — so the suite sweeps before it starts.
func sweepLeftoverTestPosts(client *threads.Client) (deleted int, err error) {
	cutoff := time.Now().Add(-sweepLookback)

	posts, err := client.GetUserPostsWithOptions(context.Background(), threads.ConvertToUserID(testUserID), &threads.PostsOptions{
		Limit: 100,
		Since: cutoff.Unix(),
	})
	if err != nil {
		return 0, fmt.Errorf("listing posts for sweep: %w", err)
	}

	for _, post := range posts.Data {
		if !strings.Contains(post.Text, testPostMarker) {
			continue
		}
		if _, err := client.DeletePost(context.Background(), threads.ConvertToPostID(post.ID)); err != nil {
			// Keep going: one undeletable leftover must not block the run.
			fmt.Fprintf(os.Stderr, "sweep: failed to delete leftover post %s: %v\n", post.ID, err)
			continue
		}
		deleted++
	}
	return deleted, nil
}

// TestMain sweeps leftover test posts before handing control to the suite.
func TestMain(m *testing.M) {
	if !hasCredentials() {
		// Without credentials every test skips; nothing to sweep.
		os.Exit(m.Run())
	}

	client, err := newSweepClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sweep: skipped, could not build client: %v\n", err)
		os.Exit(m.Run())
	}

	if deleted, err := sweepLeftoverTestPosts(client); err != nil {
		fmt.Fprintf(os.Stderr, "sweep: skipped: %v\n", err)
	} else if deleted > 0 {
		fmt.Fprintf(os.Stderr, "sweep: deleted %d leftover test post(s) from a previous run\n", deleted)
	}

	os.Exit(m.Run())
}

// newSweepClient builds a client outside of any test, for use by TestMain.
func newSweepClient() (*threads.Client, error) {
	config := &threads.Config{
		ClientID:     testClientID,
		ClientSecret: testSecret,
		RedirectURI:  testRedirectURI,
	}

	client, err := threads.NewClient(config)
	if err != nil {
		return nil, err
	}

	err = client.SetTokenInfo(&threads.TokenInfo{
		AccessToken: testAccessToken,
		TokenType:   threads.TokenTypeBearer,
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		UserID:      testUserID,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}
