package auth

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for the authenticated user's username.
	UsernameKey contextKey = "username"
	// RoleKey is the context key for the authenticated user's role.
	RoleKey contextKey = "role"
)
