package threads

// validResourceID reports whether s can be safely embedded as one URL path
// segment. Threads resource IDs are opaque, but the API's IDs use only RFC
// 3986 unreserved characters. Restricting IDs to that set prevents a caller-
// supplied ID from adding a path, query, or fragment to an authenticated
// request.
func validResourceID(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '~' {
			continue
		}
		return false
	}

	return true
}

// PostID represents a unique identifier for a post
type PostID string

// String returns the string representation of the PostID
func (id PostID) String() string {
	return string(id)
}

// Valid reports whether the PostID is non-empty and safe for use as a URL path segment.
func (id PostID) Valid() bool {
	return validResourceID(string(id))
}

// UserID represents a unique identifier for a user
type UserID string

// String returns the string representation of the UserID
func (id UserID) String() string {
	return string(id)
}

// Valid reports whether the UserID is non-empty and safe for use as a URL path segment.
func (id UserID) Valid() bool {
	return validResourceID(string(id))
}

// ContainerID represents a unique identifier for a media container
type ContainerID string

// String returns the string representation of the ContainerID
func (id ContainerID) String() string {
	return string(id)
}

// Valid reports whether the ContainerID is non-empty and safe for use as a URL path segment.
func (id ContainerID) Valid() bool {
	return validResourceID(string(id))
}

// LocationID represents a unique identifier for a location
type LocationID string

// String returns the string representation of the LocationID
func (id LocationID) String() string {
	return string(id)
}

// Valid reports whether the LocationID is non-empty and safe for use as a URL path segment.
func (id LocationID) Valid() bool {
	return validResourceID(string(id))
}

// ConvertToPostID safely converts a string to PostID
func ConvertToPostID(s string) PostID {
	return PostID(s)
}

// ConvertToUserID safely converts a string to UserID
func ConvertToUserID(s string) UserID {
	return UserID(s)
}

// ConvertToContainerID safely converts a string to ContainerID
func ConvertToContainerID(s string) ContainerID {
	return ContainerID(s)
}

// ConvertToLocationID safely converts a string to LocationID
func ConvertToLocationID(s string) LocationID {
	return LocationID(s)
}
