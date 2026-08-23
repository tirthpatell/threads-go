package threads

import "testing"

func TestResourceIDValidation(t *testing.T) {
	idTypes := map[string]func(string) bool{
		"post":      func(value string) bool { return PostID(value).Valid() },
		"user":      func(value string) bool { return UserID(value).Valid() },
		"container": func(value string) bool { return ContainerID(value).Valid() },
		"location":  func(value string) bool { return LocationID(value).Valid() },
	}

	validIDs := []string{
		"1234567890",
		"post_1",
		"location-456",
		"opaque.id~suffix",
	}
	invalidIDs := []string{
		"",
		".",
		"..",
		"victim/threads",
		"victim?fields=id",
		"victim#fragment",
		"victim%2Fthreads",
		`victim\threads`,
		"victim id",
		"victim\nid",
		"victim:id",
		"üser",
	}

	for typeName, valid := range idTypes {
		t.Run(typeName, func(t *testing.T) {
			for _, id := range validIDs {
				if !valid(id) {
					t.Errorf("expected %q to be valid", id)
				}
			}
			for _, id := range invalidIDs {
				if valid(id) {
					t.Errorf("expected %q to be rejected", id)
				}
			}
		})
	}
}
