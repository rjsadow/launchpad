package gateway

import (
	"context"

	"github.com/rjsadow/launchpad/internal/db"
	"github.com/rjsadow/launchpad/internal/sessions"
)

// SessionManagerResolver adapts sessions.Manager to the SessionResolver interface
type SessionManagerResolver struct {
	manager *sessions.Manager
}

// NewSessionManagerResolver creates a new resolver from a session manager
func NewSessionManagerResolver(manager *sessions.Manager) *SessionManagerResolver {
	return &SessionManagerResolver{
		manager: manager,
	}
}

// ValidateSession checks if a session exists and is in a running state
func (r *SessionManagerResolver) ValidateSession(ctx context.Context, sessionID string) error {
	session, err := r.manager.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session == nil {
		return ErrSessionNotFound
	}

	if session.Status != db.SessionStatusRunning {
		return ErrSessionNotRunning
	}

	if session.PodIP == "" {
		return ErrUpstreamUnavailable
	}

	return nil
}

// ResolveSession returns the upstream WebSocket URL for a session
func (r *SessionManagerResolver) ResolveSession(ctx context.Context, sessionID string) (string, error) {
	session, err := r.manager.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}

	if session == nil {
		return "", ErrSessionNotFound
	}

	url := r.manager.GetPodWebSocketEndpoint(session)
	if url == "" {
		return "", ErrUpstreamUnavailable
	}

	return url, nil
}
