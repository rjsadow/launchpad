package auth

import (
	"context"
	"net/http"
	"strings"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserContextKey is the key for storing user info in request context
	UserContextKey ContextKey = "user"
)

// UserInfo represents authenticated user information stored in request context
type UserInfo struct {
	UserID   string
	Username string
	IsAdmin  bool
}

// Middleware provides authentication middleware functionality
type Middleware struct {
	tokenService *TokenService
	enabled      bool
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(tokenService *TokenService, enabled bool) *Middleware {
	return &Middleware{
		tokenService: tokenService,
		enabled:      enabled,
	}
}

// RequireAuth returns middleware that requires authentication
func (m *Middleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			// Auth disabled, allow request with anonymous user
			ctx := context.WithValue(r.Context(), UserContextKey, &UserInfo{
				UserID:   "anonymous",
				Username: "anonymous",
				IsAdmin:  true, // Admin by default when auth is disabled
			})
			next(w, r.WithContext(ctx))
			return
		}

		user, err := m.extractUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// RequireAdmin returns middleware that requires admin privileges
func (m *Middleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			// Auth disabled, allow request with admin user
			ctx := context.WithValue(r.Context(), UserContextKey, &UserInfo{
				UserID:   "anonymous",
				Username: "anonymous",
				IsAdmin:  true,
			})
			next(w, r.WithContext(ctx))
			return
		}

		user, err := m.extractUser(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !user.IsAdmin {
			http.Error(w, "Forbidden: admin access required", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// OptionalAuth returns middleware that extracts user info if available but doesn't require it
func (m *Middleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			ctx := context.WithValue(r.Context(), UserContextKey, &UserInfo{
				UserID:   "anonymous",
				Username: "anonymous",
				IsAdmin:  true,
			})
			next(w, r.WithContext(ctx))
			return
		}

		user, _ := m.extractUser(r)
		if user != nil {
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
		}
		next(w, r)
	}
}

// extractUser extracts user information from the request
func (m *Middleware) extractUser(r *http.Request) (*UserInfo, error) {
	tokenString := m.extractToken(r)
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	claims, err := m.tokenService.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		UserID:   claims.UserID,
		Username: claims.Username,
		IsAdmin:  claims.IsAdmin,
	}, nil
}

// extractToken extracts the JWT token from the request
// Checks: 1) Authorization header (Bearer token), 2) Cookie
func (m *Middleware) extractToken(r *http.Request) string {
	// Check Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Check cookie
	cookie, err := r.Cookie(m.tokenService.CookieName())
	if err == nil {
		return cookie.Value
	}

	return ""
}

// GetUser retrieves user info from request context
func GetUser(r *http.Request) *UserInfo {
	user, ok := r.Context().Value(UserContextKey).(*UserInfo)
	if !ok {
		return nil
	}
	return user
}

// GetUserID retrieves user ID from request context, returns "anonymous" if not found
func GetUserID(r *http.Request) string {
	user := GetUser(r)
	if user == nil {
		return "anonymous"
	}
	return user.UserID
}
