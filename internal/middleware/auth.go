package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/rjsadow/launchpad/internal/auth"
)

// AuthMiddleware provides JWT authentication middleware.
type AuthMiddleware struct {
	jwtService  *auth.JWTService
	authEnabled bool
}

// NewAuthMiddleware creates a new AuthMiddleware instance.
func NewAuthMiddleware(jwtService *auth.JWTService, authEnabled bool) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService:  jwtService,
		authEnabled: authEnabled,
	}
}

// RequireAuth middleware validates JWT tokens and adds user info to the request context.
// If auth is disabled, it passes through without validation.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled (development mode)
		if !m.authEnabled {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error": "Invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				http.Error(w, `{"error": "Token has expired"}`, http.StatusUnauthorized)
				return
			}
			http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Add user info to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, auth.UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, auth.UsernameKey, claims.Username)
		ctx = context.WithValue(ctx, auth.RoleKey, claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuthFunc is a wrapper for handler functions.
func (m *AuthMiddleware) RequireAuthFunc(next http.HandlerFunc) http.HandlerFunc {
	return m.RequireAuth(next).ServeHTTP
}

// OptionalAuth middleware extracts user info from JWT if present, but doesn't require it.
// Useful for endpoints that behave differently for authenticated vs anonymous users.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled
		if !m.authEnabled {
			next.ServeHTTP(w, r)
			return
		}

		// Try to extract token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			next.ServeHTTP(w, r)
			return
		}

		tokenString := parts[1]

		// Validate token (ignore errors for optional auth)
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err == nil {
			ctx := r.Context()
			ctx = context.WithValue(ctx, auth.UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, auth.UsernameKey, claims.Username)
			ctx = context.WithValue(ctx, auth.RoleKey, claims.Role)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// RequireRole middleware checks if the authenticated user has a required role.
// Must be used after RequireAuth.
func (m *AuthMiddleware) RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip role check if auth is disabled
		if !m.authEnabled {
			next.ServeHTTP(w, r)
			return
		}

		userRole, ok := r.Context().Value(auth.RoleKey).(string)
		if !ok || userRole == "" {
			http.Error(w, `{"error": "Authentication required"}`, http.StatusUnauthorized)
			return
		}

		// Admin role has access to everything
		if userRole == "admin" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if user has the required role
		if userRole != role {
			http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
