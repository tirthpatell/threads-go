//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	threads "github.com/tirthpatell/threads-go"
)

// trackPost registers deletion of a published test post via t.Cleanup, so the
// post is removed even when the test fails, returns early, or calls t.Fatal.
// Previously the deletes sat inline at the end of each test body and were
// skipped whenever a test did not reach them.
func trackPost(t *testing.T, client *threads.Client, postID string) {
	t.Helper()
	if postID == "" {
		return
	}
	t.Cleanup(func() {
		// The API needs a moment after publish before a delete is accepted.
		time.Sleep(1 * time.Second)
		if _, err := client.DeletePost(context.Background(), threads.ConvertToPostID(postID)); err != nil {
			t.Logf("cleanup: failed to delete test post %s: %v", postID, err)
			return
		}
		t.Logf("cleanup: deleted test post %s", postID)
	})
}
