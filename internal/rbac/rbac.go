package rbac

import (
	"context"
	"fmt"
)

// Role represents a user role in the system
type Role string

const (
	// RoleAdmin has full access to all operations
	RoleAdmin Role = "admin"
	// RoleAppAuthor can create and manage applications
	RoleAppAuthor Role = "app-author"
	// RoleUser can view and launch applications
	RoleUser Role = "user"
)

// Permission represents an action that can be performed
type Permission string

const (
	// Application permissions
	PermissionAppCreate Permission = "app:create"
	PermissionAppRead   Permission = "app:read"
	PermissionAppUpdate Permission = "app:update"
	PermissionAppDelete Permission = "app:delete"

	// Session permissions
	PermissionSessionCreate Permission = "session:create"
	PermissionSessionRead   Permission = "session:read"
	PermissionSessionDelete Permission = "session:delete"
	PermissionSessionAll    Permission = "session:all" // View all sessions (admin only)

	// Analytics permissions
	PermissionAnalyticsRead  Permission = "analytics:read"
	PermissionAnalyticsWrite Permission = "analytics:write"

	// Audit permissions
	PermissionAuditRead Permission = "audit:read"

	// Config permissions
	PermissionConfigRead Permission = "config:read"
)

// rolePermissions defines which permissions each role has
var rolePermissions = map[Role][]Permission{
	RoleAdmin: {
		// Apps - full access
		PermissionAppCreate,
		PermissionAppRead,
		PermissionAppUpdate,
		PermissionAppDelete,
		// Sessions - full access including viewing all users' sessions
		PermissionSessionCreate,
		PermissionSessionRead,
		PermissionSessionDelete,
		PermissionSessionAll,
		// Analytics - full access
		PermissionAnalyticsRead,
		PermissionAnalyticsWrite,
		// Audit - read access
		PermissionAuditRead,
		// Config - read access
		PermissionConfigRead,
	},
	RoleAppAuthor: {
		// Apps - create, read, update (no delete)
		PermissionAppCreate,
		PermissionAppRead,
		PermissionAppUpdate,
		// Sessions - own sessions only
		PermissionSessionCreate,
		PermissionSessionRead,
		PermissionSessionDelete,
		// Analytics - read only
		PermissionAnalyticsRead,
		PermissionAnalyticsWrite,
		// Config - read access
		PermissionConfigRead,
	},
	RoleUser: {
		// Apps - read only
		PermissionAppRead,
		// Sessions - own sessions only
		PermissionSessionCreate,
		PermissionSessionRead,
		PermissionSessionDelete,
		// Analytics - write only (record launches)
		PermissionAnalyticsWrite,
		// Config - read access
		PermissionConfigRead,
	},
}

// ValidRoles returns all valid role names
func ValidRoles() []Role {
	return []Role{RoleAdmin, RoleAppAuthor, RoleUser}
}

// IsValidRole checks if a role string is valid
func IsValidRole(r string) bool {
	switch Role(r) {
	case RoleAdmin, RoleAppAuthor, RoleUser:
		return true
	}
	return false
}

// HasPermission checks if a role has a specific permission
func HasPermission(role Role, permission Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a role
func GetPermissions(role Role) []Permission {
	perms, ok := rolePermissions[role]
	if !ok {
		return nil
	}
	return perms
}

// User represents an authenticated user with their role
type User struct {
	ID    string `json:"id"`
	Role  Role   `json:"role"`
	Email string `json:"email,omitempty"`
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(permission Permission) bool {
	return HasPermission(u.Role, permission)
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const userContextKey contextKey = "rbac_user"

// ContextWithUser adds a user to the context
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext extracts the user from context
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

// RequirePermission returns an error if the user doesn't have the required permission
func RequirePermission(ctx context.Context, permission Permission) error {
	user, ok := UserFromContext(ctx)
	if !ok {
		return fmt.Errorf("no user in context")
	}
	if !user.HasPermission(permission) {
		return fmt.Errorf("permission denied: %s required", permission)
	}
	return nil
}
