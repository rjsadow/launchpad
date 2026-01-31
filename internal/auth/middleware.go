package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userContextKey contextKey = "user"

// Middleware provides authentication middleware
type Middleware struct {
	sessions    *SessionManager
	authEnabled bool
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(sessions *SessionManager, authEnabled bool) *Middleware {
	return &Middleware{
		sessions:    sessions,
		authEnabled: authEnabled,
	}
}

// RequireAuth middleware ensures the user is authenticated
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, create a mock user
		if !m.authEnabled {
			mockSession := &SessionData{
				UserID:  "anonymous",
				Email:   "anonymous@localhost",
				Name:    "Anonymous User",
				Roles:   []string{},
				IsAdmin: true, // Give anonymous users admin access when auth is disabled
			}
			ctx := context.WithValue(r.Context(), userContextKey, mockSession)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		session, err := m.sessions.GetSession(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add session to context
		ctx := context.WithValue(r.Context(), userContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin middleware ensures the user is an admin
func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetSessionFromContext(r.Context())
		if session == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !session.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OptionalAuth middleware adds user to context if authenticated, but doesn't require it
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, create a mock user
		if !m.authEnabled {
			mockSession := &SessionData{
				UserID:  "anonymous",
				Email:   "anonymous@localhost",
				Name:    "Anonymous User",
				Roles:   []string{},
				IsAdmin: true,
			}
			ctx := context.WithValue(r.Context(), userContextKey, mockSession)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		session, err := m.sessions.GetSession(r)
		if err == nil && session != nil {
			ctx := context.WithValue(r.Context(), userContextKey, session)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// GetSessionFromContext retrieves the session from the request context
func GetSessionFromContext(ctx context.Context) *SessionData {
	session, _ := ctx.Value(userContextKey).(*SessionData)
	return session
}
