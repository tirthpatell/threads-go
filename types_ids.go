package threads

// PostID represents a unique identifier for a post
type PostID string

// String returns the string representation of the PostID
func (id PostID) String() string {
	return string(id)
}

// Valid checks if the PostID is not empty
func (id PostID) Valid() bool {
	return id != ""
}

// UserID represents a unique identifier for a user
type UserID string

// String returns the string representation of the UserID
func (id UserID) String() string {
	return string(id)
}

// Valid checks if the UserID is not empty
func (id UserID) Valid() bool {
	return id != ""
}

// ContainerID represents a unique identifier for a media container
type ContainerID string

// String returns the string representation of the ContainerID
func (id ContainerID) String() string {
	return string(id)
}

// Valid checks if the ContainerID is not empty
func (id ContainerID) Valid() bool {
	return id != ""
}

// LocationID represents a unique identifier for a location
type LocationID string

// String returns the string representation of the LocationID
func (id LocationID) String() string {
	return string(id)
}

// Valid checks if the LocationID is not empty
func (id LocationID) Valid() bool {
	return id != ""
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
