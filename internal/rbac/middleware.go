package rbac

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// UserStore is an interface for retrieving user role information
type UserStore interface {
	// GetUserRole returns the role for a given user ID
	// Returns RoleUser as default if user not found
	GetUserRole(userID string) (Role, error)
}

// Middleware provides RBAC middleware functionality
type Middleware struct {
	store         UserStore
	getUserIDFunc func(r *http.Request) string
	bypassPaths   map[string]bool
	defaultRole   Role
	authEnabled   bool
}

// MiddlewareOption configures the middleware
type MiddlewareOption func(*Middleware)

// WithBypassPaths sets paths that bypass authentication
func WithBypassPaths(paths []string) MiddlewareOption {
	return func(m *Middleware) {
		for _, p := range paths {
			m.bypassPaths[p] = true
		}
	}
}

// WithDefaultRole sets the default role for unauthenticated users
func WithDefaultRole(role Role) MiddlewareOption {
	return func(m *Middleware) {
		m.defaultRole = role
	}
}

// WithAuthEnabled enables or disables authentication enforcement
func WithAuthEnabled(enabled bool) MiddlewareOption {
	return func(m *Middleware) {
		m.authEnabled = enabled
	}
}

// WithUserIDFunc sets a custom function to extract user ID from request
func WithUserIDFunc(fn func(r *http.Request) string) MiddlewareOption {
	return func(m *Middleware) {
		m.getUserIDFunc = fn
	}
}

// NewMiddleware creates a new RBAC middleware
func NewMiddleware(store UserStore, opts ...MiddlewareOption) *Middleware {
	m := &Middleware{
		store:       store,
		bypassPaths: make(map[string]bool),
		defaultRole: RoleUser,
		authEnabled: true,
		getUserIDFunc: func(r *http.Request) string {
			// Default: check X-User-ID header (for proxy-based auth)
			// In production, this would come from JWT, session, or SSO
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				// Fallback to query param for testing
				userID = r.URL.Query().Get("user_id")
			}
			return userID
		},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Authenticate wraps a handler with authentication
// It extracts user info and adds it to the request context
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check bypass paths
		if m.shouldBypass(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// If auth is disabled, use anonymous user with default role
		if !m.authEnabled {
			user := &User{
				ID:   "anonymous",
				Role: m.defaultRole,
			}
			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Extract user ID
		userID := m.getUserIDFunc(r)
		if userID == "" {
			// Allow anonymous access with restricted role
			user := &User{
				ID:   "anonymous",
				Role: m.defaultRole,
			}
			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Get user's role from store
		role, err := m.store.GetUserRole(userID)
		if err != nil {
			log.Printf("RBAC: error getting role for user %s: %v", userID, err)
			role = m.defaultRole
		}

		user := &User{
			ID:   userID,
			Role: role,
		}

		ctx := ContextWithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission returns middleware that requires a specific permission
func (m *Middleware) RequirePermission(permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !user.HasPermission(permission) {
				log.Printf("RBAC: user %s (role=%s) denied permission %s", user.ID, user.Role, permission)
				http.Error(w, `{"error":"forbidden","message":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole returns middleware that requires a specific role
func (m *Middleware) RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			log.Printf("RBAC: user %s (role=%s) denied, required one of %v", user.ID, user.Role, roles)
			http.Error(w, `{"error":"forbidden","message":"insufficient role"}`, http.StatusForbidden)
		})
	}
}

// RequireAdmin is a convenience wrapper for RequireRole(RoleAdmin)
func (m *Middleware) RequireAdmin() func(http.Handler) http.Handler {
	return m.RequireRole(RoleAdmin)
}

// shouldBypass checks if a path should bypass authentication
func (m *Middleware) shouldBypass(path string) bool {
	// Check exact match
	if m.bypassPaths[path] {
		return true
	}
	// Check prefix match for paths ending with *
	for bypassPath := range m.bypassPaths {
		if strings.HasSuffix(bypassPath, "*") {
			prefix := strings.TrimSuffix(bypassPath, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// WriteError writes a JSON error response
func WriteError(w http.ResponseWriter, status int, err string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   err,
		Message: message,
	})
}

// CheckPermission is a helper for handlers to check permissions
// Returns true if the user has the permission, false otherwise
// If false, it also writes an appropriate error response
func CheckPermission(w http.ResponseWriter, r *http.Request, permission Permission) bool {
	user, ok := UserFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized", "no user context")
		return false
	}
	if !user.HasPermission(permission) {
		WriteError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
		return false
	}
	return true
}

// GetUserFromRequest is a helper to get the user from a request
func GetUserFromRequest(r *http.Request) *User {
	user, _ := UserFromContext(r.Context())
	return user
}
